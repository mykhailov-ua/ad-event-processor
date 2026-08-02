package controlplane

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type AdminBootJSON struct {
	User        UserDTO  `json:"user"`
	Permissions []string `json:"permissions"`
}

type AdminUIGate struct {
	auth *AuthMiddleware
}

func NewAdminUIGate(auth *AuthMiddleware) *AdminUIGate {
	if auth == nil {
		return nil
	}
	return &AdminUIGate{auth: auth}
}

func (g *AdminUIGate) bootFromRequest(r *http.Request) (AdminBootJSON, bool) {
	if g == nil || g.auth == nil {
		return AdminBootJSON{}, false
	}
	user, ok := g.auth.SessionFromRequest(r)
	if !ok {
		return AdminBootJSON{}, false
	}
	perms := GetPermissionsForRole(user.Role)
	dto := UserDTO{
		ID:          user.UserID.String(),
		Role:        user.Role,
		CustomerID:  user.CustomerID.String(),
		Permissions: perms,
	}
	return AdminBootJSON{User: dto, Permissions: perms}, true
}

func injectAdminBoot(indexHTML []byte, boot AdminBootJSON) ([]byte, error) {
	raw, err := json.Marshal(boot)
	if err != nil {
		return nil, err
	}
	snippet := append([]byte(`<script id="__BOOT__" type="application/json">`), raw...)
	snippet = append(snippet, []byte(`</script>`)...)
	marker := []byte("<div id=\"root\"></div>")
	idx := bytes.Index(indexHTML, marker)
	if idx < 0 {
		return nil, io.ErrUnexpectedEOF
	}
	out := make([]byte, 0, len(indexHTML)+len(snippet))
	out = append(out, indexHTML[:idx]...)
	out = append(out, snippet...)
	out = append(out, indexHTML[idx:]...)
	return out, nil
}

func serveLoginHTML(w http.ResponseWriter, staticFS http.FileSystem) {
	f, err := staticFS.Open("login.html")
	if err != nil {
		http.Error(w, "login page missing", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "login page unreadable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(data)
}

func serveIndexHTML(w http.ResponseWriter, staticFS http.FileSystem, boot *AdminBootJSON) {
	f, err := staticFS.Open("index.html")
	if err != nil {
		http.Error(w, "index missing", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "index unreadable", http.StatusInternalServerError)
		return
	}
	if boot != nil {
		injected, errInject := injectAdminBoot(data, *boot)
		if errInject != nil {
			http.Error(w, "boot inject failed", http.StatusInternalServerError)
			return
		}
		data = injected
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(data)
}

func isAdminSPAPath(path string) bool {
	if path == "/" {
		return true
	}
	if strings.HasPrefix(path, "/api/") {
		return false
	}
	if strings.Contains(path, ".") {
		return false
	}
	return true
}
