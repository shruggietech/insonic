# S019 research

2026-10-11: Spec Kit Phase 0 dispatched read-only startup, feedback and model researchers. Findings are confirmed against the current implementation.

## Startup

Decision: startup-only generated report through an explicit os.Pipe, one Wait owner and an actual shared deadline. Rationale: detached startup discards failures, and polling cannot distinguish failed bootstrap from missing readiness. Alternatives: raw stderr retention risks secrets; permanent stdout supervision binds runtime output to client lifetime; filesystem handshakes add stale authority. Preserve existing hidden/setsid/breakaway flags. See [Go pipe ownership](https://pkg.go.dev/os/exec#Cmd.StdoutPipe).

## Feedback and errors

Decision: preserve actual generated reasons below a common bounded renderer, with optional terminal wait observation. Rationale: duplicate credentials, damaged backup and source-map reuse all use broad conflict, while durable execution drops messages. Alternatives: guessing from operation alone is false; printing arbitrary child/provider text violates redaction; adding per-item approval gates contradicts owner requirements.

## Models

Decision: successful incompatible inspection, response-only readiness, declared-role files and ready-for-preparation datasets. Rationale: alias resolution already constructs the exact context then discards it through error dispatch; persisted registered/available state is not inference readiness. Alternatives: inventing durable verified/materialized states or weakening strict elections changes authority unnecessarily. Recovery must use models acquire MANIFEST_FILE, or exact existing acquisition work show/wait/retry.

## Diagnostic controls

Decision: private explicitly elected log file, fixed generated fields, 1 MiB rotation and two predecessors; no raw diagnostic sinks. Rationale: useful correlation without content/credential capture. Alternatives: universal default disk logs introduce unrelated filesystem prerequisites; verbose raw payload logging leaks content.
