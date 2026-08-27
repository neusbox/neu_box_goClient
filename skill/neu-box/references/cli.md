# Optional neu-sbox helper

Use this reference only when `neu-sbox` is available and a deterministic helper is preferable
to direct `curl`. The Worker HTTP API remains the normative interface; do not invent CLI flags
or require installation of the helper.

## Connection

`NEU_BOX_URL` selects the Worker and defaults to `http://127.0.0.1:59075`.
`NEU_BOX_USER` selects the Worker host user and otherwise follows the current user.
The Worker API has no authentication; connect only over a trusted network.

Check connectivity and API compatibility when selecting the helper:

```bash
neu-sbox check
```

## Submit a Host job

```bash
neu-sbox submit --json \
  --device-num 1 --cpu 4 --mem 8 -- \
  python train.py
```

Relevant options:

- `--device-num N`: automatically allocate N accelerator devices.
- `--device ID` or `--devices 1,3`: request exact device IDs; mutually exclusive with
  `--device-num`.
- `--cpu N`, `--mem N`: CPU cores and memory in GB; zero or omission means unlimited.
- `--priority 1`: urgent queue priority. Use only when explicitly justified.
- Arguments after `--` form the remote command.

The JSON response contains `task_id`, `position`, and `priority`. Persist `task_id`.

## Follow or inspect a job

```bash
neu-sbox wait TASK_ID
neu-sbox wait TASK_ID --interval 5s --timeout 2h
neu-sbox result --json TASK_ID
neu-sbox tasks --json
```

`wait` uses byte offsets to fetch only new log segments while separately polling task state.
At a terminal state it drains the final available segment before exiting. `--timeout 0`
means no local wait timeout; reaching a local timeout does not cancel the Worker task.

This is the main reason to choose the helper for a long-running job. Keep waiting on the same
task ID; never submit another task merely because local monitoring stopped.

Task states:

- `queued`: persisted and waiting for resources.
- `running`: sandbox allocated and command executing.
- `completed`: exit code zero.
- `failed`: nonzero exit, timeout, cancellation, execution failure, or cleanup failure.

`result` is a snapshot and returns the current full log. Prefer `wait` for long-running jobs
so repeated polling does not download the entire log each time.

## Existing Docker container

```bash
neu-sbox submit --json \
  --device-num 1 \
  --container training-01 \
  --workdir /workspace \
  --container-user 1000:1000 \
  --env PYTHONUNBUFFERED=1 -- \
  python train.py
```

The container must already be running and have the requested device nodes mounted. Worker
cannot add devices to an existing container. Container jobs must request at least one device.

## Terminal sandbox

Use `neu-sbox acquire` only when the caller has a persistent shell whose child processes
should inherit the sandbox:

```bash
neu-sbox acquire --device-num 1 --cpu 4 --mem 8
neu-sbox status
neu-sbox release SANDBOX_NAME
```

For an Agent executing independent commands, prefer queued `submit` jobs because a sandbox
attached to a short-lived launcher shell will not provide a stable execution context.
