package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleaseOnlySendsSandboxName(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decodeRequest(t, r, &received)
		writeJSON(t, w, 200, map[string]any{})
	}))
	defer server.Close()
	application, _, errOut := testApplication(server.URL)
	application.insideContainer = func() bool { return true }
	if rc := application.run([]string{"release", "sbx_yuxd_43210.slice"}); rc != 0 {
		t.Fatalf("rc=%d %s", rc, errOut.String())
	}
	if len(received) != 1 || received["sandbox_name"] != "sbx_yuxd_43210.slice" {
		t.Fatalf("payload=%v", received)
	}
}

func TestHostStatusReadsProcWithoutExternalCommands(t *testing.T) {
	const sandboxName = "sbx_yuxd_43210.slice"
	application, out, errOut := testApplication("http://127.0.0.1:1")
	application.readFile = func(path string) ([]byte, error) {
		if path != "/proc/111/cgroup" {
			t.Fatalf("unexpected path: %s", path)
		}
		return []byte("0::/sandbox_" + sandboxName + "\n"), nil
	}
	code := application.run([]string{"status"})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "sandbox: "+sandboxName) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
