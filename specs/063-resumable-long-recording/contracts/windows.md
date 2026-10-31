# Windowed processing contract

Extend existing recording processing JSON with optional `windows`: core duration, context duration, concurrency cap, and track/channel/sample-range selections. Defaults are 30-second cores, 2-second context each side and one active window. Core range is 1 through 300 seconds; explicit context is nonnegative and no more than half the core; omitted context uses min(2 seconds, half the core), while explicit zero means no context; concurrency cap is 1 through 4, with serial execution permitted below the cap. All limits are validated before admission. Explicit per-run settings take precedence over saved defaults; frozen retries retain normalized settings.

New long recordings automatically elect windows; short requests retain their existing behavior unless windows are explicitly elected. Already frozen historical payloads retain their prior path. JSON/schema/help must describe identical names and bounds.

ProcessingPlan is version 1, source-clock-native-samples. Its digest binds canonical source, expected current document, model/adapter byte identities, semantic options, selected intervals and core/context layout. Actual fallback device is excluded only when permitted by the frozen device policy.

Window results retain mapped-relative segments and optional words; timing without recognizer support is never invented. Diarization labels are local to window and track. Final transcript cue provenance identifies contributing windows and original intervals. Verified output changes necessarily produce a new reconciliation digest.

Checkpoint publication uses generation-qualified artifact identity and stable logical plan/window/stage identity. Catalog acceptance is fenced and replay-safe. Artifact availability alone is insufficient; reuse rehashes bytes and validates payload identity. Ready checkpoints remain through cancellation and portable restore, and retire after current acceptance or invalidation.

Resource recovery is finite: normal attempt, at most one compatible CUDA batch reduction where applicable, then at most one policy-permitted CPU fallback. All attempts share the original deadline and preserve model/source/provider configuration.
