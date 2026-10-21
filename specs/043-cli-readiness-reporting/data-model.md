# Data model

No catalog migration is required.

- Error.asset: optional fixed role label and logical name (ASCII slash-relative, maximum 128 bytes, no absolute paths, traversal, controls or device names). The allowlisted reason supplies the verification stage/correction. Invalid observations are discarded at SafeError.
- Worker plan budget: download_bytes is the elected network transfer; profile_download_bytes is nominal; local required/verified/missing totals describe only declared assets. Offline download_bytes is exactly zero. Unknown/unreadable assets retain explicit observations rather than fabricated verification.
- Model access: supplied/local resolution state is distinct from upstream not_checked; access_ready concerns only locally necessary references. Reused verified bytes require no credentials.
- Task readiness: configuration, file verification, runtime protocol and observed availability remain distinct. Pipeline usability is the conjunction of its elected tasks, not a fixed transcription/diarization pair.
- Wait outcome: one caller context; last observed work is preserved. Timeout/cancellation does not mutate durable work or assert unobserved state.
