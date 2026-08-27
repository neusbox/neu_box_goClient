---
name: neu-box
description: Submit, monitor, and inspect accelerator-backed jobs and terminal sandboxes through neu-sbox. Use when work should run on a Neu Box Worker, needs GPU/NPU/CPU/memory allocation, or requires following Neu Box task status and logs. Do not use for ordinary local commands that need no Neu Box resources.
---

# Neu Box

Use the `neu-sbox` CLI. It talks directly to the Worker selected by `NEU_BOX_URL` and does
not require the WebUI.

## Command jobs

1. Run `neu-sbox check` before the first operation against a Worker.
2. Submit with `neu-sbox submit --json ... -- <command>`. Preserve the returned `task_id`
   before doing more work. A successful submission only means the task entered the queue.
3. Run `neu-sbox wait <task_id>` to follow incremental logs until the task reaches
   `completed` or `failed`. Keep waiting on that same local process when the command is
   long-running; do not submit a duplicate task.
4. Use `neu-sbox result --json <task_id>` when a point-in-time metadata and full-log
   snapshot is needed.

`wait` writes task output incrementally to stdout and status transitions to stderr. It exits
zero only for `completed`; a failed task or monitoring error exits nonzero. Interrupting
`wait` stops only the local follower, not the remote task. Resume with the same `task_id`.

If submission returned an uncertain response, inspect `neu-sbox tasks --json` before
retrying. The Worker has no idempotency key, so blind retries can create duplicate jobs.

Use the resource values the user requested. Do not raise priority, request devices, or pick
specific device IDs without a reason grounded in the request. Prefer `submit` for Agent-run
commands; `acquire` changes a persistent parent shell and is unsuitable for unrelated,
short-lived command invocations.

Read [references/cli.md](references/cli.md) when choosing resource flags, running inside an
existing Docker container, configuring the Worker URL, or diagnosing task states.
