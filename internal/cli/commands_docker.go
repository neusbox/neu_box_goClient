package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/neusbox/neu_box_goClient/internal/api"
	"github.com/neusbox/neu_box_goClient/internal/dockerargs"
)

// runDocker 是 `neubox docker run`：把当前 shell 的沙盒身份拼成 annotation，
// 然后 exec docker，让 docker 自己接管终端、信号和退出码。
//
// 客户端不碰 BPF、不碰 cgroup、不做授权判断 —— 它只负责把那行 annotation 拼对，
// 拼错了（或者没拼）Worker 会在登记时拒掉，容器在设备侧被 fail-closed 拦下。
func (a *app) runDocker(args []string) int {
	if len(args) == 0 {
		return a.usageError("用法: " + dockerRunUsage)
	}
	switch args[0] {
	case "run":
	case "help", "-h", "--help":
		a.printDockerHelp()
		return 0
	default:
		return a.usageError(fmt.Sprintf(
			"docker 目前只支持 run，不支持 %s；用法: %s", args[0], dockerRunUsage))
	}

	// `docker run` 之后的参数一个都不解析，原样透传（连同开头的 --）。
	sandboxName, code := a.resolveOwnSandbox()
	if code != 0 {
		return code
	}

	dockerBinary, err := a.lookPath("docker")
	if err != nil {
		a.printError("docker_not_found",
			"PATH 里找不到 docker 命令；请在装了 docker 的主机上运行 neubox docker run")
		return 1
	}

	argv := dockerargs.BuildDockerArgs(sandboxName, args[1:])
	// 成功即进程已被替换，不会再走到这里。
	if err := a.execFn(dockerBinary, argv, os.Environ()); err != nil {
		a.printError("docker_exec_failed", fmt.Sprintf("启动 docker 失败: %v", err))
		return 1
	}
	return 0
}

// resolveOwnSandbox 反查本进程所在的沙盒，失败时自己打印错误并返回非 0 退出码。
//
// neubox 是 acquire 过的那个 shell 的子进程，cgroup 成员身份是继承的，
// 所以报**自己的** PID 就能让 Worker 反查到沙盒名。查不到就报错让用户先
// acquire —— 不猜、也不退化成"不加 annotation 照样起"：没有 annotation 的
// 容器拿不到设备，而且是在容器内第一次初始化时才炸，比在命令行上报错难查得多。
func (a *app) resolveOwnSandbox() (string, int) {
	// 容器里 PID 是 namespace 内视角，报给宿主机 Worker 会撞上别的进程，
	// 反查出一个别人的沙盒名。这条是错误分支，不在主路径上；先报错，
	// 异常处理以后再说。
	if a.insideContainer() {
		a.printError("sandbox_lookup_unsupported",
			"容器内无法按 PID 反查沙盒（PID namespace 与宿主机不同）；"+
				"请改用原生 docker run --annotation sandbox_cgroup=<沙盒名>")
		return "", 1
	}

	selfPID := a.getPID()
	query := url.Values{"pid": []string{strconv.Itoa(selfPID)}}
	status, raw, err := a.worker.Request(http.MethodGet, "/sandbox/status", query, nil)
	if err != nil {
		return "", a.requestError(err)
	}
	if err := api.ResponseError(status, raw); err != nil {
		return "", a.workerFailure(status, raw)
	}
	var response struct {
		SandboxName *string `json:"sandbox_name"`
	}
	if err := api.DecodeJSON(raw, &response); err != nil {
		return "", a.internalError("invalid_worker_response", err)
	}
	if response.SandboxName == nil || strings.TrimSpace(*response.SandboxName) == "" {
		a.printError("not_in_sandbox", fmt.Sprintf(
			"当前进程 (pid %d) 不在任何沙盒中；请先在 shell 里执行 neubox acquire，"+
				"或改用原生 docker run --annotation sandbox_cgroup=<沙盒名>", selfPID))
		return "", 1
	}
	return strings.TrimSpace(*response.SandboxName), 0
}

const dockerRunUsage = "neubox docker run <docker run 参数...>"

func (a *app) printDockerHelp() {
	fmt.Fprint(a.out, `neubox docker run — 在沙盒里启动容器

用法:`+"\n    "+dockerRunUsage+`

说明:
    docker run 的参数一个不改，只在最前面补一行 annotation：
        neubox docker run --rm -it ubuntu bash
      = docker run --annotation sandbox_cgroup=<沙盒名> --rm -it ubuntu bash

    容器必须带这行 annotation：Worker 靠它把容器登记到沙盒名下，没登记的
    容器即使卡空着也一律拿不到设备（fail-closed）。runtime 不用写，
    neu-box-runtime 已经是默认 runtime。

    沙盒按本进程 PID 反查（neubox 是 acquire 过的 shell 的子进程，cgroup
    身份是继承的）；查不到直接报错，不会退化成不加 annotation 启动。

    没有自己的选项：要显式指定沙盒、或者要自己写 annotation，直接用原生
    docker run --annotation sandbox_cgroup=<沙盒名> ... 就好。
`)
}
