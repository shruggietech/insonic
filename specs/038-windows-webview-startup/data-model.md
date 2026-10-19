# Data model: native qualification observations

## Failed native child

Stage/action identify the fixture. stdout and stderr are independent diagnostic strings bounded to 65536 bytes per stream before serialization. Exit code is an integer; -1 means no completed child. Truncated, timed_out and cancelled are separate booleans. A failure never grants native Exit success. Successful JSON stdout stays unchanged; failure evidence is separate stderr/artifact data.

## Focused diagnostic

Tool and payload revisions are separate exact commits. Asset filenames, sizes, hashes and inventory match elected authority. Ordered attempts retain bounded streams and results. Diagnostic-only observations cannot emit scope-proof.json or publication eligibility.

## Owned launch resources

A diagnostic root and exact job/process identities belong to one disposable owner. Historical and derived executable trees are separate. Stream readers retain at most 65536 bytes each while draining output, with total-byte/truncation/incomplete-capture observations. Concurrent CLI workers join and idle before removal. Compiler children receive an exact selected environment. Existing native GUI completion remains authoritative; diagnostic cleanup cannot grant native Exit success. New profile isolation/release design is deferred under #154.

## Join failure observation

Run, attempt, revision, artifact prefix and failed scope/job/status/conclusion are retained before rejection. This evidence cannot authorize delivery.
