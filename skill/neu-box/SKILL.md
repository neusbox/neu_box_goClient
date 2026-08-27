---
name: neu-box
description: Submit, monitor, and inspect accelerator-backed jobs and terminal sandboxes through the Neu Box Worker HTTP API, with neu-sbox as an optional reliable helper. Use when work should run on a Neu Box Worker, needs GPU/NPU/CPU/memory allocation, or requires following Neu Box task status and logs. Do not use for ordinary local commands that need no Neu Box resources.
---

# Neu Box

Talk directly to the Worker selected by `NEU_BOX_URL`; the default is
`http://127.0.0.1:59075`. Do not route through the WebUI.

## Choose the interface

- Use the Worker API v2 with `curl` by default. It is the transparent, normative interface
  and does not require learning a private CLI grammar.
- Use `neu-sbox` when it is already available and its deterministic behavior is useful,
  especially for incremental log following or terminal sandbox identity handling. It is an
  optional helper, not a prerequisite for this skill.

Do not install software merely to switch interfaces unless the user asks. Read
[references/http-api.md](references/http-api.md) before making direct API calls. Read
[references/cli.md](references/cli.md) only when choosing or using the optional helper.

## Command jobs

1. Check `/healthz` before the first operation and require `api_version >= 2`.
2. Submit exactly once with `POST /tasks`. Preserve the returned `task_id` before doing more
   work. HTTP 202 means only that the task entered the queue.
3. Poll `GET /tasks/<task_id>` until `completed` or `failed`. Inspect
   `GET /tasks/<task_id>/log` while it runs and once more after it reaches a terminal state.
4. For long-running incremental output, `neu-sbox wait <task_id>` may replace manual status
   and log polling when the helper is available.

The Worker has no idempotency key. If submission returned an uncertain response, inspect
`GET /tasks` for the original user, command, and creation time before considering another
submission. Never blindly retry or create a duplicate task.

Use the resource values the user requested. Do not raise priority, request devices, or pick
specific device IDs without a reason grounded in the request. Prefer queued command jobs for
Agent-run commands; terminal acquire changes a persistent process hierarchy and is unsuitable
for unrelated, short-lived invocations.

Only cancel or delete a task when the user requested it or the active workflow clearly
requires it. Stopping local polling does not stop the remote task.

## Terminal sandboxes

Use a terminal sandbox only when a persistent Worker-host process and its descendants should
inherit the allocation. Prefer the optional `neu-sbox acquire/release` helper because it
handles shell and container process identity. With direct HTTP, verify the Worker-host PID
and ownership before calling `/sandbox/acquire`; always release the returned sandbox when the
authorized terminal workflow ends.
