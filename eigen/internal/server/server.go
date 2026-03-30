package server

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

//go:embed ui
var uiFS embed.FS

// Start wires routes and blocks serving on the given port.
func Start(specsRoot string, port int, open bool) error {
	mux := http.NewServeMux()

	// JSON API
	mux.HandleFunc("/api/modules", modulesHandler(specsRoot))
	mux.HandleFunc("/api/modules/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			changeApproveHandler(specsRoot)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/reject") {
			changeRejectHandler(specsRoot)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/changes") {
			moduleChangesHandler(specsRoot)(w, r)
		} else {
			moduleDetailHandler(specsRoot)(w, r)
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
