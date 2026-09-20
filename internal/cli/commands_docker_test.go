package cli

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// execCall 记录一次假 exec 的调用，测试靠它断言最终交给 docker 的 argv。
type execCall struct {
	path string
	argv []string
	env  []string
}

// recordExec 装一个假 exec：真的 syscall.Exec 成功后不返回，这里假装成功。
func recordExec(application *app) *execCall {
	call := &execCall{}
	application.execFn = func(path string, argv []string, env []string) error {
		call.path = path
		call.argv = argv
		call.env = env
		return nil
	}
	return call
}

func TestDockerRunInjectsAnnotationForOwnSandbox(t *testing.T) {
	var requestedPID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/sandbox/status" {
			t.Errorf("unexpected path: %s", request.URL.Path)
		}
		requestedPID = request.URL.Query().Get("pid")
		writeJSON(t, writer, http.StatusOK, map[string]any{
			"sandbox_name": "sbx_yuxd_42.slice",
			"sandbox":      map[string]any{"name": "sbx_yuxd_42.slice"},
		})
	}))
	defer server.Close()

	application, _, errOut := testApplication(server.URL)
	call := recordExec(application)

	code := application.run([]string{"docker", "run", "--rm", "-it", "ubuntu", "bash"})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if requestedPID != "222" {
		t.Fatalf("客户端应报自己的 PID (222)，实际报的是 %q", requestedPID)
	}
	if call.path != "/usr/bin/docker" {
		t.Fatalf("exec path=%q", call.path)
	}
	want := []string{
		"run",
		"--annotation", "sandbox_cgroup=sbx_yuxd_42.slice",
		"--rm", "-it", "ubuntu", "bash",
	}
	if !reflect.DeepEqual(call.argv, want) {
		t.Fatalf("got  %q\nwant %q", call.argv, want)
	}
	if len(call.env) == 0 {
		t.Fatal("exec 应该带上环境变量")
	}
}

// docker run 之后的参数逐字出现在 docker argv 里，包括开头的 --。
func TestDockerRunPassesArgumentsThroughVerbatim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusOK, map[string]any{
			"sandbox_name": "sbx_yuxd_42.slice",
		})
	}))
	defer server.Close()

	application, _, errOut := testApplication(server.URL)
	call := recordExec(application)

	// 全是 neubox 不认识的参数：连 --sandbox 也一并交给 docker。
	passthrough := []string{"--", "-it", "--rm", "--sandbox", "x", "--annotation", "k=v", "ubuntu", "bash"}
	args := append([]string{"docker", "run"}, passthrough...)
	if code := application.run(args); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	want := append([]string{"run", "--annotation", "sandbox_cgroup=sbx_yuxd_42.slice"}, passthrough...)
	if !reflect.DeepEqual(call.argv, want) {
		t.Fatalf("got  %q\nwant %q", call.argv, want)
	}
}

func TestDockerRunWithoutSandboxFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		// Worker 对"不在沙盒里"就是 200 + null，不是错误码。
		writeJSON(t, writer, http.StatusOK, map[string]any{
			"sandbox_name": nil,
			"sandbox":      nil,
		})
	}))
	defer server.Close()

	application, _, errOut := testApplication(server.URL)
	call := recordExec(application)

	code := application.run([]string{"docker", "run", "--rm", "-it", "ubuntu", "bash"})
	if code == 0 {
		t.Fatal("没有沙盒时必须报错退出")
	}
	if call.argv != nil {
		t.Fatalf("不能在没定到沙盒时 exec docker: %q", call.argv)
	}
	if !strings.Contains(errOut.String(), "acquire") {
		t.Fatalf("错误信息应提示先 acquire: %s", errOut.String())
	}
}

// 容器里 PID 是 namespace 内视角，反查会撞上宿主机上别的进程，只能报错。
func TestDockerRunRefusesPIDLookupInsideContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		t.Error("容器内不该拿 namespace 里的 PID 去查沙盒")
		writeJSON(t, writer, http.StatusOK, map[string]any{})
	}))
	defer server.Close()

	application, _, errOut := testApplication(server.URL)
	application.insideContainer = func() bool { return true }
	call := recordExec(application)

	code := application.run([]string{"docker", "run", "--rm", "ubuntu"})
	if code == 0 {
		t.Fatal("容器内反查沙盒必须报错")
	}
	if call.argv != nil {
		t.Fatalf("不该 exec docker: %q", call.argv)
	}
	if !strings.Contains(errOut.String(), "原生 docker") {
		t.Fatalf("错误信息应指出改用原生 docker: %s", errOut.String())
	}
}

func TestDockerRunReportsMissingDockerBinary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusOK, map[string]any{"sandbox_name": "sbx_a.slice"})
	}))
	defer server.Close()

	application, _, errOut := testApplication(server.URL)
	application.lookPath = func(string) (string, error) {
		return "", errors.New("executable file not found in $PATH")
	}
	call := recordExec(application)

	code := application.run([]string{"docker", "run", "ubuntu"})
	if code == 0 {
		t.Fatal("找不到 docker 必须报错")
	}
	if call.argv != nil {
		t.Fatalf("不该 exec docker: %q", call.argv)
	}
	if !strings.Contains(errOut.String(), "docker") {
		t.Fatalf("错误信息应提到 docker: %s", errOut.String())
	}
}

func TestDockerRejectsUnsupportedSubcommands(t *testing.T) {
	for _, args := range [][]string{
		{"docker"},
		{"docker", "ps"},
		{"docker", "compose", "up"},
	} {
		application, _, errOut := testApplication("http://127.0.0.1:1")
		call := recordExec(application)

		if code := application.run(args); code != 2 {
			t.Fatalf("%v: exit=%d stderr=%s", args, code, errOut.String())
		}
		if call.argv != nil {
			t.Fatalf("%v: 不该 exec docker", args)
		}
		if !strings.Contains(errOut.String(), "run") {
			t.Fatalf("%v: 错误信息应指出只支持 run: %s", args, errOut.String())
		}
	}
}
