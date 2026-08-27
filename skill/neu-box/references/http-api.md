# Worker HTTP API v2

Use this reference for direct Worker access with `curl`. `NEU_BOX_URL` selects the Worker and
defaults to `http://127.0.0.1:59075`. The API has no authentication; connect only over a trusted
network.

Use `--noproxy '*'` for Worker addresses on private networks and use `--fail-with-body -sS` so
HTTP errors remain visible. If the installed curl lacks `--fail-with-body`, use `--fail -sS`.

## Check the Worker

```bash
curl --noproxy '*' --fail-with-body -sS \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/healthz"
```

Require `role` to be `worker` and `api_version` to be at least `2` before using the task
endpoints. An older or missing API version does not support this contract.

## Submit one command job

Fill the JSON with the requested Worker host user, command, and resources. Keep the JSON valid;
do not interpolate unescaped external text into it.

```bash
curl --noproxy '*' --fail-with-body -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  --data-binary @- \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/tasks" <<'JSON'
{
  "user_id": "pengyt",
  "command": "python train.py",
  "device_num": 1,
  "cpu": 4,
  "memory": 8,
  "mem_unit": "GB",
  "priority": 0,
  "target": {"type": "host"}
}
JSON
```

The HTTP 202 JSON response contains `task_id`, `position`, and `priority`. Preserve the exact
`task_id` from that response. Do not use regex to parse JSON in shell automation; use an
available JSON parser or preserve the structured tool output directly.

Resource fields:

- `device_num`: automatically allocate this many devices; zero means none.
- `device_ids`: request exact device minor IDs or `major:minor` values; when nonempty it takes
  precedence over `device_num`.
- `cpu`: CPU core limit; zero means unlimited.
- `memory` and `mem_unit`: memory limit in `GB` or `MB`; zero means unlimited.
- `priority`: `0` for normal or `1` for urgent. Use `1` only when explicitly justified.

For a command inside an existing Docker container, request at least one device and replace the
target with:

```json
{
  "type": "docker_existing",
  "container": "training-01",
  "workdir": "/workspace",
  "user": "1000:1000",
  "env": {"PYTHONUNBUFFERED": "1"}
}
```

The container must already be running and have the requested device nodes mounted.

## Monitor and inspect

Replace `TASK_ID` with the exact returned ID:

```bash
# Queue and recent tasks; use this to investigate an uncertain submission.
curl --noproxy '*' --fail-with-body -sS \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/tasks"

# Status and result metadata.
curl --noproxy '*' --fail-with-body -sS \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/tasks/TASK_ID"

# A bounded current log view while polling.
curl --noproxy '*' --fail-with-body -sS \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/tasks/TASK_ID/log?tail=8192&raw=1"

# Full final log after completed or failed.
curl --noproxy '*' --fail-with-body -sS \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/tasks/TASK_ID/log?raw=1"
```

Task states are `queued`, `running`, `completed`, and `failed`. Continue polling the same ID
for `queued` or `running`. At a terminal state, fetch the final log and inspect
`result.returncode`, `result.timed_out`, and `result.error`. For reliable incremental log
offset handling, use the optional `neu-sbox wait TASK_ID` helper instead of reimplementing its
state machine.

## Cancel or delete

Only send this mutation when cancellation or deletion is authorized:

```bash
curl --noproxy '*' --fail-with-body -sS \
  -X DELETE \
  -H 'Content-Type: application/json' \
  --data-binary '{"task_ids":["TASK_ID"]}' \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/tasks"
```

Queued and completed records are deleted. A running task is canceled asynchronously and
finishes as `failed`.

## Terminal sandbox endpoints

Use these only for a persistent process on the Worker host. The PID must be the real
Worker-host PID owned by `username`, not an Agent-side or unrelated local PID.

```bash
# Acquire resources for an existing persistent process.
curl --noproxy '*' --fail-with-body -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  --data-binary '{"username":"pengyt","pid":12345,"device_num":1,"cpu":4,"memory":8,"mem_unit":"GB"}' \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/sandbox/acquire"

# Inspect sandboxes and preserve sandbox_name from the acquire response.
curl --noproxy '*' --fail-with-body -sS \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/sandbox/list?username=pengyt"

# Release only the sandbox returned by the authorized acquire workflow.
curl --noproxy '*' --fail-with-body -sS \
  -X POST \
  -H 'Content-Type: application/json' \
  --data-binary '{"sandbox_name":"SANDBOX_NAME"}' \
  "${NEU_BOX_URL:-http://127.0.0.1:59075}/sandbox/release"
```

The lower-level endpoints are:

- `POST /sandbox/acquire`: create an allocation for `username` and Worker-host `pid`.
- `POST /sandbox/join`: join another verified Worker-host PID to an existing sandbox.
- `GET /sandbox/list`: inspect current sandbox records.
- `POST /sandbox/release`: release a returned `sandbox_name`.

Prefer `neu-sbox acquire/release` when it is available because it resolves host/container PID
identity and remembers the sandbox metadata needed for reliable release.
