# Data model: S048

## Optional probe observation

Inspection gains optional `probe`: fixed mode (inspect/snapshot), stage, reason, total native elapsed milliseconds, effective budget milliseconds, child elapsed milliseconds and bounded child observations/counts. Native and child clocks have separate origins. Missing probe means a legacy receipt, not failure.

At most 16 fixed-stage observations; elapsed durations are integral 0..3600000 ms; budget is integral 0..60000 ms; file counts 0..30000; byte counts 0..9007199254740991. Unknown fields, duplicate JSON members, invalid labels, reversed stage/time/count progress, oversized files and redirected checkpoints are rejected. No raw output, exception, environment or path fields.

## States

Inspection starts at interpreter verification, then private home and child stages. Child stages cover package versions, library imports, CPU/CUDA prerequisites and optional dependency inventory/hashing. Terminal outcome is success or a fixed failure reason. Expired enclosing work is not_started/deadline rather than demonstrated engine absence. Cancellation remains cancellation.

## Authority

Only original compatibility plus a complete pinned snapshot can create an environment receipt. Optional diagnostics never override engine, schema, inventory or hash checks. Old receipt bytes are never rewritten. Failed setup retains safe inspection facts in its existing work checkpoint without publishing tools or preserving failed attempt directories.
