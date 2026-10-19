# Data model: native qualification observations

## Failed native child

Stage/action identify the fixture. stdout and stderr are independent diagnostic strings bounded to 65536 bytes per stream before serialization. Exit code is an integer; -1 means no completed child. Truncated, timed_out and cancelled are separate booleans. A failure never grants native Exit success. Successful JSON stdout stays unchanged; failure evidence is separate stderr/artifact data.

## Focused diagnostic

Tool and payload revisions are separate exact commits. Asset filenames, sizes, hashes and inventory match elected authority. Ordered attempts retain bounded streams and results. Diagnostic-only observations cannot emit scope-proof.json or publication eligibility.

## Owned launch resources

A fixture root/profile and exact PIDs belong to one owner. Lifecycle advances from created to started/ready/action/root-exited/resources-released/cleanup. Forced cleanup cannot satisfy native Exit. Unrelated paths/PIDs are excluded.

## Join failure observation

Run, attempt, revision, artifact prefix and failed scope/job/status/conclusion are retained before rejection. This evidence cannot authorize delivery.
