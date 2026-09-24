package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResultPrintsLogAndSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/tasks/abc123":
			returnCode := 0
			writeJSON(t, writer, http.StatusOK, map[string]any{
				"task_id":     "abc123",
				"user_id":     "yuxd",
				"command":     "echo ok",
				"status":      "completed",
				"cpu":         2,
				"mem":         "4G",
				"device_num":  1,
				"devices":     []string{"235:0"},
				"finished_at": 1_700_000_000,
				"result": map[string]any{
					"returncode": returnCode,
					"timed_out":  false,
				},
			})
		case "/tasks/abc123/log":
			_, _ = io.WriteString(writer, "hello\n")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	application, out, errOut := testApplication(server.URL)
	code := application.run([]string{"result", "abc123"})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	for _, expected := range []string{"hello", "[✓ completed]", "rc=0", "设备=1"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("missing %q in %s", expected, out.String())
		}
	}
}
