package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neusbox/neu_box_goClient/internal/api"
)

type terminalAcquireRequest struct {
	Username  string   `json:"username"`
	PID       int      `json:"pid"`
	DeviceNum int      `json:"device_num"`
	DeviceIDs []string `json:"device_ids"`
	CPU       int      `json:"cpu"`
	Memory    int      `json:"memory"`
	MemUnit   string   `json:"mem_unit"`
}

type acquireResponse struct {
	SandboxName string   `json:"sandbox_name"`
	Devices     []string `json:"devices"`
	Error       string   `json:"error"`
}

func (a *app) runAcquire(args []string) int {
	options, err := parseAcquireOptions(args)
	if err != nil {
		return a.usageError(err.Error())
	}
	return a.runTerminalAcquire(options)
}

func parseAcquireOptions(args []string) (acquireOptions, error) {
	options := acquireOptions{}
	positionals := make([]string, 0, 3)

	for index := 0; index < len(args); index++ {
		argument := args[index]
		if handled, err := consumeResourceOption(args, &index, &options.resourceOptions); handled || err != nil {
			if err != nil {
				return options, err
			}
			continue
		}

		switch argument {
		case "--pid":
			raw, err := optionValue(args, &index)
			if err != nil {
				return options, err
			}
			options.pid, err = positiveInteger("--pid", raw)
			if err != nil {
				return options, err
			}
			options.pidSet = true
		case "--container":
			return options, errors.New("acquire 仅支持宿主 PID；请在宿主 shell 申请沙盒后启动容器")
		case "--command", "--workdir", "--container-user", "--env", "--":
			return options, errors.New("命令任务已移至 submit；请使用 neubox submit [选项] -- <command>")
		default:
			if strings.HasPrefix(argument, "-") {
				return options, fmt.Errorf("未知 acquire 选项: %s", argument)
			}
			positionals = append(positionals, argument)
		}
	}

	if len(positionals) > 3 {
		return options, errors.New("acquire 最多接受 3 个位置参数: device_num cpu mem；命令任务请使用 submit")
	}
	if err := applyPositionalResources(&options.resourceOptions, positionals); err != nil {
		return options, err
	}
	if err := validateResourceOptions(&options.resourceOptions); err != nil {
		return options, err
	}
	return options, nil
}

func (a *app) runTerminalAcquire(options acquireOptions) int {
	shellPID := a.getPPID()
	if options.pidSet {
		shellPID = options.pid
	}
	if a.insideContainer() {
		return a.usageError("acquire 仅支持宿主 PID，不能从容器内申请；请在宿主 shell 申请沙盒")
	}
	payload := terminalAcquireRequest{
		Username:  a.config.username,
		PID:       shellPID,
		DeviceNum: options.deviceNum,
		DeviceIDs: options.deviceIDs,
		CPU:       options.cpu,
		Memory:    options.memory,
		MemUnit:   "GB",
	}

	status, raw, err := a.worker.Request(http.MethodPost, "/sandbox/acquire", nil, payload)
	if err != nil {
		return a.requestError(err)
	}
	if err := api.ResponseError(status, raw); err != nil {
		return a.workerFailure(status, raw)
	}
	if status == http.StatusAccepted {
		var queued struct {
			AcquireID string `json:"acquire_id"`
		}
		if err := api.DecodeJSON(raw, &queued); err != nil || queued.AcquireID == "" {
			return a.internalError("invalid_worker_response", errors.New("Worker 排队响应缺少 acquire_id"))
		}
		for status == http.StatusAccepted {
			time.Sleep(100 * time.Millisecond)
			status, raw, err = a.worker.Request(http.MethodGet, "/sandbox/acquire/"+url.PathEscape(queued.AcquireID), nil, nil)
			if err != nil {
				return a.requestError(err)
			}
			if err := api.ResponseError(status, raw); err != nil {
				return a.workerFailure(status, raw)
			}
		}
	}
	var response acquireResponse
	if err := api.DecodeJSON(raw, &response); err != nil || response.SandboxName == "" {
		if err == nil {
			err = errors.New("Worker 响应缺少 sandbox_name")
		}
		return a.internalError("invalid_worker_response", err)
	}
	if a.jsonOutput {
		_ = printJSON(a.out, raw)
		return 0
	}
	devices := "无"
	if len(response.Devices) > 0 {
		devices = strings.Join(response.Devices, ",")
	}
	fmt.Fprintln(a.out, "[neubox] 沙盒已创建")
	fmt.Fprintf(a.out, "    sandbox: %s\n", response.SandboxName)
	fmt.Fprintf(a.out, "    pid: %d\n", shellPID)
	fmt.Fprintf(a.out, "    devices: %s\n", devices)
	fmt.Fprintf(a.out, "    release: neubox release %s\n", response.SandboxName)
	fmt.Fprintln(a.out, "    docker: neubox docker run -- <docker run 参数>")
	return 0
}
