package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neusbox/neu_box_goClient/internal/api"
)

type taskQueueResponse struct {
	Queue        []taskResultResponse `json:"queue"`
	TotalPending int                  `json:"total_pending"`
}

func (a *app) runTasks(args []string) int {
	if len(args) != 0 {
		return a.usageError("用法: neubox tasks")
	}
	status, raw, err := a.worker.Request(http.MethodGet, "/tasks", nil, nil)
	if err != nil {
		return a.requestError(err)
	}
	if err := api.ResponseError(status, raw); err != nil {
		return a.workerFailure(status, raw)
	}
	var response taskQueueResponse
	if err := api.DecodeJSON(raw, &response); err != nil {
		return a.internalError("invalid_worker_response", err)
	}
	if a.jsonOutput {
		_ = printJSON(a.out, raw)
		return 0
	}
	fmt.Fprintln(a.out, "[neubox] 任务列表")
	fmt.Fprintf(a.out, "    total: %d\n", len(response.Queue))
	fmt.Fprintf(a.out, "    pending: %d\n", response.TotalPending)
	if len(response.Queue) == 0 {
		fmt.Fprintln(a.out, "    (无)")
		return 0
	}
	for _, task := range response.Queue {
		fmt.Fprintln(a.out)
		fmt.Fprintf(a.out, "    %s\n", entryID(task))
		fmt.Fprintf(a.out, "        kind: %s\n", entryKind(task))
		fmt.Fprintf(a.out, "        status: %s\n", task.Status)
		fmt.Fprintf(a.out, "        user: %s\n", task.UserID)
		if entryKind(task) == "acquire" {
			// acquire 是会话不是命令任务：没有 command / 日志 / 退出码。
			if task.PID != 0 {
				fmt.Fprintf(a.out, "        pid: %d\n", task.PID)
			}
			if task.Sandbox != "" {
				fmt.Fprintf(a.out, "        sandbox: %s\n", task.Sandbox)
			}
		} else {
			fmt.Fprintf(a.out, "        command: %s\n", task.Command)
		}
		if task.Position > 0 {
			fmt.Fprintf(a.out, "        position: #%d\n", task.Position)
		}
		if resources := taskResourceText(task); resources != "" {
			fmt.Fprintf(a.out, "        resources: %s\n", resources)
		}
	}
	return 0
}

// entryKind / entryID 兼容两类条目：任务用 task_id，acquire 会话用 request_id。
// 老 Worker 不带 kind/id 字段时按任务处理。
func entryKind(entry taskResultResponse) string {
	if entry.Kind != "" {
		return entry.Kind
	}
	return "task"
}

func entryID(entry taskResultResponse) string {
	if entry.ID != "" {
		return entry.ID
	}
	return entry.TaskID
}

type taskResult struct {
	ReturnCode *int `json:"returncode"`
	TimedOut   bool `json:"timed_out"`
	Error      any  `json:"error"`
}

type taskResultResponse struct {
	TaskID     string      `json:"task_id"`
	Kind       string      `json:"kind"`
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	Command    string      `json:"command"`
	Status     string      `json:"status"`
	Position   int         `json:"position"`
	PID        int         `json:"pid"`
	Sandbox    string      `json:"sandbox_name"`
	CPU        int         `json:"cpu"`
	Mem        string      `json:"mem"`
	DeviceNum  int         `json:"device_num"`
	Devices    []string    `json:"devices"`
	CreatedAt  *float64    `json:"created_at"`
	FinishedAt *float64    `json:"finished_at"`
	Result     *taskResult `json:"result"`
}

func (a *app) runResult(args []string) int {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return a.usageError("用法: neubox result <task_id>")
	}
	taskID := strings.TrimSpace(args[0])
	pathID := url.PathEscape(taskID)
	status, raw, err := a.worker.Request(http.MethodGet, "/tasks/"+pathID, nil, nil)
	if err != nil {
		return a.requestError(err)
	}
	if err := api.ResponseError(status, raw); err != nil {
		return a.workerFailure(status, raw)
	}
	var response taskResultResponse
	if err := api.DecodeJSON(raw, &response); err != nil {
		return a.internalError("invalid_worker_response", err)
	}

	logStatus, logRaw, logErr := a.worker.Request(
		http.MethodGet,
		"/tasks/"+pathID+"/log",
		url.Values{"raw": []string{"1"}},
		nil,
	)
	if logErr != nil {
		return a.requestError(fmt.Errorf("获取任务日志: %w", logErr))
	}
	if err := api.ResponseError(logStatus, logRaw); err != nil {
		return a.workerFailure(logStatus, logRaw)
	}

	if a.jsonOutput {
		var output map[string]any
		if err := api.DecodeJSON(raw, &output); err != nil {
			return a.internalError("invalid_worker_response", err)
		}
		output["log"] = string(logRaw)
		_ = printJSONValue(a.out, output)
		return 0
	}

	fmt.Fprintln(a.out, "[neubox] 任务日志")
	if len(logRaw) > 0 {
		fmt.Fprintln(a.out, strings.TrimRight(string(logRaw), "\r\n"))
	} else {
		fmt.Fprintln(a.out, "    (暂无日志)")
	}

	icons := map[string]string{
		"completed": "✓",
		"failed":    "✗",
		"running":   "▶",
		"queued":    "○",
	}
	icon := icons[response.Status]
	if icon == "" {
		icon = "?"
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "[neubox] 任务结果")
	fmt.Fprintf(a.out, "    ID=%s\n", response.TaskID)
	fmt.Fprintf(a.out, "    status: [%s %s]\n", icon, response.Status)
	if response.Result != nil && response.Result.ReturnCode != nil {
		fmt.Fprintf(a.out, "    rc=%d\n", *response.Result.ReturnCode)
		if response.Result.TimedOut {
			fmt.Fprintln(a.out, "    timed_out: true")
		}
		if response.Result.Error != nil {
			fmt.Fprintf(a.out, "    error: %v\n", response.Result.Error)
		}
	}
	fmt.Fprintf(a.out, "    user: %s\n", response.UserID)
	fmt.Fprintf(a.out, "    command: %s\n", response.Command)
	resources := taskResourceText(response)
	if resources != "" {
		fmt.Fprintf(a.out, "    resources: %s\n", resources)
	}
	timestamp := response.FinishedAt
	if timestamp == nil {
		timestamp = response.CreatedAt
	}
	if timestamp != nil {
		formatted := time.Unix(int64(*timestamp), 0).Local().Format("01-02 15:04")
		fmt.Fprintf(a.out, "    time: %s\n", formatted)
	}
	return 0
}

func taskResourceText(task taskResultResponse) string {
	resources := make([]string, 0, 3)
	if task.CPU != 0 {
		resources = append(resources, fmt.Sprintf("CPU=%d", task.CPU))
	}
	if task.Mem != "" && task.Mem != "0" {
		resources = append(resources, "mem="+task.Mem)
	}
	if task.DeviceNum != 0 {
		devices := fmt.Sprintf("设备=%d", task.DeviceNum)
		if len(task.Devices) > 0 {
			devices += " (" + strings.Join(task.Devices, ",") + ")"
		}
		resources = append(resources, devices)
	}
	return strings.Join(resources, "  ")
}
