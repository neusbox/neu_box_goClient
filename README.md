# Neu Box Go Client (neu-sbox)

Neu Box 的终端沙盒隔离 / 命令任务提交客户端。Go 单文件静态二进制，
**直连 worker**（不经过 WebUI/master）。
2026-08-25 从 `neu_box` 单体仓库拆分独立维护。

## 命令

```
neu-sbox acquire [选项...]       创建沙盒 / 提交任务（--command 时）
neu-sbox release <sandbox_name>  释放沙盒
neu-sbox {list|status|join}      沙盒管理
neu-sbox {tasks|result|log}      任务队列 / 结果 / 日志
neu-sbox check                   检查 worker 可达性与 API 版本兼容性
neu-sbox version
```

`acquire` 选项：

| 选项 | 说明 |
|---|---|
| `--devices 1,3` | 指定卡号（与 --device-num 互斥） |
| `--device-num 2` | 自动分配卡数量 |
| `--cpu 4` / `--mem 8` | 资源上限，0 = 不限 |
| `--priority 1` | 队列优先级：0=普通，1=赶论文 |
| `--pid 12345` | 指定 PID（默认当前 shell 的父进程） |
| `--command "..."` | 提交一次性命令任务（进入命令模式） |
| `--container NAME` | 容器终端身份 / 命令目标容器 |
| `--workdir PATH` | 容器命令工作目录 |
| `--env K=V` | 容器命令环境变量（可重复） |

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
| 0.0.1 | ≥ 0.3.0（`api_version >= 1`） |

`neu-sbox check` 查询 worker `/healthz`：

- `api_version >= 1` → 兼容 ✓
- 有 `api_version` 但 < 1 → 退出码 1（不兼容）
- 无 `api_version` 字段（worker < 0.3.0）→ 退出码 0 + warning（best-effort）

API 契约见 [neu_box](https://github.com/nihaopeng/neu_box) 仓库
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

# 提交赶论文优先级任务（4 卡）
neu-sbox acquire --command "python train.py" --device-num 4 --priority 1

# 队列 / 结果
neu-sbox tasks
neu-sbox result <task_id>
```
