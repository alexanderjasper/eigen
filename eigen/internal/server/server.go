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

	// Load registered worktrees from registry.
	reg, err := worktree.ReadRegistry(gitRoot)
	if err != nil {
		log.Printf("warning: reading worktree registry: %v", err)
	}
	for _, e := range reg.Entries {
		addWorktreeToState(state, e)
	}

	// Watch .eigen/worktrees.json for hot-reload (AC-011).
	if gitRoot != "" {
		regPath := worktree.RegistryPath(gitRoot)
		go watchRegistryFile(gitRoot, regPath, state)
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

// watchRegistryFile starts a goroutine watching the registry file for changes.
// On a write event it calls reloadRegistry.
func watchRegistryFile(gitRoot, regPath string, state *serveState) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("warning: creating registry watcher: %v", err)
		return
	}
	defer watcher.Close()

	eigenDir := filepath.Join(gitRoot, ".eigen")

	// Try to watch the .eigen dir directly; fall back to the git root.
	if err := watcher.Add(eigenDir); err != nil {
		if err2 := watcher.Add(gitRoot); err2 != nil {
			log.Printf("warning: watching for registry changes: %v", err2)
			return
		}
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Re-add .eigen if it was newly created (first worktree create).
			if event.Op&fsnotify.Create != 0 {
				_ = watcher.Add(eigenDir)
			}
			// Reload on writes to the registry file.
			if event.Name == regPath &&
				(event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0) {
				reloadRegistry(gitRoot, state)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("registry watcher error: %v", err)
		}
	}
}

// reloadRegistry reads the registry from disk and reconciles it with the current state.
// New entries get watchers; removed entries have their watchers cancelled.
// Unchanged entries are left completely undisturbed (AC-014).
func reloadRegistry(gitRoot string, state *serveState) {
	reg, err := worktree.ReadRegistry(gitRoot)
	if err != nil {
		log.Printf("warning: reloading registry: %v", err)
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	// Build a set of new entries by name.
	newByName := make(map[string]worktree.Entry, len(reg.Entries))
	for _, e := range reg.Entries {
		newByName[e.Name] = e
	}

	// Cancel watchers for removed entries and delete from map.
	for _, name := range state.order {
		if name == "main" {
			continue
		}
		if _, stillExists := newByName[name]; !stillExists {
			if aw, ok := state.worktrees[name]; ok {
				if aw.CancelWatch != nil {
					aw.CancelWatch()
				}
				delete(state.worktrees, name)
			}
		}
	}

	// Rebuild order, keeping main first and purging evicted names.
	newOrder := make([]string, 0, len(state.order))
	newOrder = append(newOrder, "main")
	for _, name := range state.order {
		if name == "main" {
			continue
		}
		if _, ok := state.worktrees[name]; ok {
			newOrder = append(newOrder, name)
		}
	}
	state.order = newOrder

	// Open watchers for newly added entries.
	for name, e := range newByName {
		if _, alreadyExists := state.worktrees[name]; alreadyExists {
			continue // unchanged — leave alone
		}
		entrySpecsRoot := filepath.Join(e.Path, "specs")
		cancel, _ := watchSpecsDir(entrySpecsRoot)
		state.order = append(state.order, name)
		state.worktrees[name] = &activeWorktree{
			Entry:       e,
			SpecsRoot:   entrySpecsRoot,
			Branch:      e.Branch,
			CancelWatch: cancel,
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
