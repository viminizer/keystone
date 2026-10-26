package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/viminizer/keystone/internal/config"
)

func TestRoutes(t *testing.T) {
	server := New(&config.Config{}).
		Get("hello", func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "hello from get")
		}).
		Post("keys/{key}/delete", func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, "deleted "+request.PathValue("key"))
		})

	tests := []struct {
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		{"GET", "/hello", 200, "hello from get"},
		{"POST", "/keys/name/delete", 200, "deleted name"},
		{"POST", "/hello", 405, ""},
		{"GET", "/other", 404, ""},
	}
	for _, test := range tests {
		// httptest makes a fake request and records the reply, without a port.
		request := httptest.NewRequest(test.method, test.path, nil)
		reply := httptest.NewRecorder()
		server.routes.ServeHTTP(reply, request)

		if reply.Code != test.wantCode {
			t.Errorf("%s %s: status = %d, want %d", test.method, test.path, reply.Code, test.wantCode)
		}
		if test.wantBody != "" && reply.Body.String() != test.wantBody {
			t.Errorf("%s %s: body = %q, want %q", test.method, test.path, reply.Body.String(), test.wantBody)
		}
	}
}
