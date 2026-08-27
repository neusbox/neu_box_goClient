# Neu Box Go Client (neu-sbox)

Neu Box 的终端沙盒隔离 / 命令任务提交客户端。Go 单文件静态二进制，
**直连 worker**（不经过 WebUI/master）。
2026-08-25 从 `neu_box` 单体仓库拆分独立维护。

## 命令

```
neu-sbox acquire [选项...]       为当前终端同步申请沙盒
neu-sbox submit [选项...] -- CMD 异步提交命令任务
neu-sbox release <sandbox_name>  释放沙盒
neu-sbox {list|status|join}      沙盒管理
neu-sbox {tasks|result|log}      任务队列 / 结果快照 / 完整日志
neu-sbox wait TASK_ID            增量跟踪日志并等待任务结束
neu-sbox skill install DIR       安装内置 Agent skill 到 DIR/neu-box
neu-sbox check                   检查 worker 可达性与 API 版本兼容性
neu-sbox [--json] version
```

资源选项可用于 `acquire` 和 `submit`：

| 选项 | 说明 |
|---|---|
| `--device 1` | 指定一张卡，可重复 |
| `--devices 1,3` | 指定卡号（与 --device-num 互斥） |
| `--device-num 2` | 自动分配卡数量 |
| `--cpu 4` / `--mem 8` | 资源上限，0 = 不限 |

`acquire` 专属选项：

| 选项 | 说明 |
|---|---|
| `--pid 12345` | 指定 PID（默认当前 shell 的父进程） |
| `--container NAME` | 容器终端身份 |

命令任务统一使用 `submit`，其专属选项为：

| 选项 | 说明 |
|---|---|
| `--priority 1` | 队列优先级：0=普通，数值越大越先执行 |
| `--container NAME` | 在已有容器中执行；不指定则在 Host 执行 |
| `--workdir PATH` | 容器命令工作目录 |
| `--container-user USER` | 容器命令用户 |
| `--env K=V` | 容器命令环境变量（可重复） |
| `--command "..."` | 命令字符串；也可把命令及参数放在 `--` 后 |

默认输出为适合终端阅读的摘要，不混入 Worker 原始 JSON。自动化调用可将
`--json` 放在子命令前或紧跟子命令；成功结果写入 stdout，失败对象写入
stderr。

## Agent skill

`neu-sbox` 内嵌符合标准目录规范的 `neu-box` skill，可以安装到任意技能根目录；
目标目录不存在时自动创建：

```bash
neu-sbox skill install ~/.codex/skills
# → ~/.codex/skills/neu-box/SKILL.md
```

skill 默认引导 Agent 通过标准 `curl` 直接使用 Worker API v2，并把 `neu-sbox`
作为可选的确定性 helper：长任务可用 `wait` 稳定处理日志 offset，终端沙盒可用
`acquire/release` 处理 Host 与容器 PID 身份。安装操作不访问 Worker，也不需要
root。

## 任务日志

`result` 返回调用时的状态与完整日志快照。长任务应使用：

```bash
neu-sbox wait TASK_ID
neu-sbox wait TASK_ID --interval 5s --timeout 2h
```

`wait` 使用日志接口的字节 offset 只读取新增部分，同时轮询任务状态；进入终态后
再拉取一次剩余日志。日志写到 stdout，状态变化写到 stderr，任务 `completed`
退出 0，`failed` 退出非 0。中断或本地超时只停止跟踪，不会取消远端任务。

## 环境变量

| 变量 | 说明 |
|---|---|
| `NEU_BOX_URL` | worker 地址，默认 `http://127.0.0.1:59075` |
| `NEU_BOX_USER` | 用户名（默认取 $USER） |
| `NEU_BOX_CONTAINER` | 容器名（容器内运行时自动探测也可） |
| `NEU_BOX_STATE_DIR` | 沙盒状态文件目录，默认 `/tmp/neu-box-<uid>` |

## 与 worker 的兼容

| 客户端 | 验证过的 worker |
|---|---|
| 0.2.0 | ≥ 0.4.0（`api_version >= 2`，使用 `/tasks`） |

`neu-sbox check` 查询 worker `/healthz`：

- `api_version >= 2` → 兼容 ✓
- 有 `api_version` 但 < 2 → 退出码 1（不兼容）
- 无 `api_version` 字段（旧版 worker）→ 退出码 1（不支持 `/tasks`）

API 契约见 [neu_box](https://github.com/neusbox/neu_box) 仓库
`docs/worker-api.md`。

## 构建

```bash
make build        # → neu-sbox + neu-sbox.sha256（本机架构）
make test vet
# 交叉编译示例:
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false \
  -tags netgo,osusergo -ldflags "-s -w -X main.version=$(cat VERSION)" \
  -o neu-sbox .
```

## 安装

```bash
./install.sh /path/to/release-dir   # 校验 sha256 → /usr/local/bin/neu-sbox
```

## 示例

```bash
# 终端独占 2 张卡
neu-sbox acquire --device-num 2
# 退出前释放
neu-sbox release "$(neu-sbox list | grep current)"

# 提交高优先级任务（4 卡）
neu-sbox submit --device-num 4 --priority 1 -- python train.py

# 队列 / 结果
neu-sbox tasks
neu-sbox wait <task_id>
neu-sbox result --json <task_id>
```
