package metricsserver

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/viminizer/keystone/internal/metrics"
)

func TestHandleMetrics(t *testing.T) {
	commandMetrics := metrics.NewCommands()
	commandMetrics.Add("GET")
	commandMetrics.Record("GET", time.Microsecond, false)
	server := New(0, commandMetrics)

	// httptest makes a fake request and records the reply, without a port.
	request := httptest.NewRequest("GET", "/metrics", nil)
	reply := httptest.NewRecorder()
	server.handleMetrics(reply, request)

	if reply.Code != 200 {
		t.Errorf("status = %d, want 200", reply.Code)
	}
	if contentType := reply.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain; version=0.0.4") {
		t.Errorf("Content-Type = %q", contentType)
	}
	want := `keystone_command_duration_seconds_count{command="get"} 1`
	if !strings.Contains(reply.Body.String(), want) {
		t.Errorf("body does not have %s:\n%s", want, reply.Body.String())
	}
}
