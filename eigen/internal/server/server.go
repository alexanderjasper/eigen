package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/alexanderjasper/eigen/internal/worktree"
)

//go:embed ui
var uiFS embed.FS

// activeWorktree holds runtime state for a single registered worktree.
type activeWorktree struct {
	Entry       worktree.Entry
	SpecsRoot   string
	Branch      string
	CancelWatch func()
}

// serveState is the live server state, protected by a read-write mutex.
type serveState struct {
	mu        sync.RWMutex
	order     []string // insertion order; "main" always first
	worktrees map[string]*activeWorktree
}

// Start wires routes and blocks serving on the given port.
func Start(gitRoot, specsRoot string, port int, open bool) error {
	state := &serveState{
		worktrees: make(map[string]*activeWorktree),
	}

	// Determine main branch.
	mainBranch, err := worktree.CurrentBranch(gitRoot)
	if err != nil {
		mainBranch = "main"
	}

	// Register the main worktree.
	mainEntry := worktree.Entry{
		Name:   "main",
		Branch: mainBranch,
		Path:   gitRoot,
	}
	mainCancel, _ := watchSpecsDir(specsRoot)
	state.order = append(state.order, "main")
	state.worktrees["main"] = &activeWorktree{
		Entry:       mainEntry,
		SpecsRoot:   specsRoot,
		Branch:      mainBranch,
		CancelWatch: mainCancel,
	}

	// Auto-discover worktrees in .claude/worktrees/.
	discovered, err := worktree.ScanWorktreesDir(gitRoot)
	if err != nil {
		log.Printf("warning: scanning worktrees dir: %v", err)
	}
	for _, e := range discovered {
		addWorktreeToState(state, e)
	}

	// Watch .claude/worktrees/ for directories created/removed while serve is running.
	if gitRoot != "" {
		go watchWorktreesDir(gitRoot, state)
	}

	mux := http.NewServeMux()

	// JSON API
	mux.HandleFunc("/api/worktrees", worktreesHandler(state))
	mux.HandleFunc("/api/modules", modulesHandler(state))
	mux.HandleFunc("/api/modules/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			changeApproveHandler(state)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/reject") {
			changeRejectHandler(state)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/changes") {
			moduleChangesHandler(state)(w, r)
		} else {
			moduleDetailHandler(state)(w, r)
		}
	})

	// Static UI
	sub, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return fmt.Errorf("embedding ui: %w", err)
	}
	fileServer := http.FileServer(http.FS(sub))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// All non-API paths serve index.html (SPA fallback) or a static asset.
		if r.URL.Path != "/" {
			// Check if it's a known asset; otherwise serve index.html.
			f, err := sub.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			// Unknown path → serve index.html so client-side routing works.
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	if open {
		go openBrowser(url)
	}

	return http.ListenAndServe(addr, mux)
}

// addWorktreeToState adds an entry to state.
// Must be called with lock held or during init (before serving).
func addWorktreeToState(state *serveState, e worktree.Entry) {
	entrySpecsRoot := filepath.Join(e.Path, "specs")
	cancel, _ := watchSpecsDir(entrySpecsRoot)
	state.order = append(state.order, e.Name)
	state.worktrees[e.Name] = &activeWorktree{
		Entry:       e,
		SpecsRoot:   entrySpecsRoot,
		Branch:      e.Branch,
		CancelWatch: cancel,
	}
}

// watchWorktreesDir watches .claude/worktrees/ for new subdirectories and adds
// them to state when they appear (e.g. after `eigen worktree create` or Claude
// creates a new worktree).
func watchWorktreesDir(gitRoot string, state *serveState) {
	wtDir := worktree.WorktreesDir(gitRoot)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("warning: creating worktrees-dir watcher: %v", err)
		return
	}
	defer watcher.Close()

	// Watch the parent .claude/ dir so we notice when worktrees/ is created.
	claudeDir := filepath.Join(gitRoot, ".claude")
	_ = watcher.Add(claudeDir)
	_ = watcher.Add(wtDir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create != 0 {
				// Re-watch wtDir if it was just created.
				if event.Name == wtDir {
					_ = watcher.Add(wtDir)
					continue
				}
				// A new entry appeared inside .claude/worktrees/
				if filepath.Dir(event.Name) == wtDir {
					name := filepath.Base(event.Name)
					state.mu.RLock()
					_, exists := state.worktrees[name]
					state.mu.RUnlock()
					if exists {
						continue
					}
					// Give git a moment to finish setting up the worktree.
					branch, err := worktree.CurrentBranch(event.Name)
					if err != nil {
						continue
					}
					e := worktree.Entry{Name: name, Branch: branch, Path: event.Name}
					state.mu.Lock()
					addWorktreeToState(state, e)
					state.mu.Unlock()
					log.Printf("auto-discovered worktree: %s (%s)", name, branch)
				}
			}
			if event.Op&fsnotify.Remove != 0 || event.Op&fsnotify.Rename != 0 {
				if filepath.Dir(event.Name) == wtDir {
					name := filepath.Base(event.Name)
					state.mu.Lock()
					if aw, ok := state.worktrees[name]; ok {
						if aw.CancelWatch != nil {
							aw.CancelWatch()
						}
						delete(state.worktrees, name)
						newOrder := make([]string, 0, len(state.order))
						for _, n := range state.order {
							if n != name {
								newOrder = append(newOrder, n)
							}
						}
						state.order = newOrder
						log.Printf("removed worktree from serve: %s", name)
					}
					state.mu.Unlock()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("worktrees-dir watcher error: %v", err)
		}
	}
}

// watchSpecsDir starts an fsnotify watcher on specsRoot and all subdirectories.
// Returns a cancel function to stop the watcher.
// Handlers read from disk on every request, so the watcher only needs to track
// new subdirectories as they appear.
func watchSpecsDir(specsRoot string) (cancel func(), err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return func() {}, err
	}

	addDirRecursive(watcher, specsRoot)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create != 0 {
					_ = watcher.Add(event.Name)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return func() { watcher.Close() }, nil
}

// addDirRecursive adds dir and all its subdirectories to watcher (best-effort).
func addDirRecursive(watcher *fsnotify.Watcher, dir string) {
	_ = watcher.Add(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			addDirRecursive(watcher, filepath.Join(dir, e.Name()))
		}
	}
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	exec.Command(cmd, args...).Start() //nolint:errcheck
}
