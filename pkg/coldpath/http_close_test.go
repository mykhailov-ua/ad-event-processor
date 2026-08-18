package coldpath

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloseHTTPResponse_nilSafe(t *testing.T) {
	CloseHTTPResponse(nil)
	resp := &http.Response{Body: http.NoBody}
	CloseHTTPResponse(resp)
}

func TestCloseHTTPResponse_drainsOnErrorPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 32))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	CloseHTTPResponse(resp)
}
