# insonic v1.1.0

- Process long recordings in bounded windows and resume verified work after interruption.
- Inspect transcript coverage by track and repair selected gaps without retranscribing unrelated windows.
- Prefer NVIDIA for new local processing and speaker work, with bounded CPU fallback and recorded device provenance.
- Improve silence handling, actionable CLI failures and backup/restore capacity recovery.

Existing supported documents remain readable; catalog schema stays at 11. Coverage, repair and advanced window configuration are CLI workflows awaiting dedicated GUI controls. Deterministic package checks do not establish acoustic accuracy or real-model CPU/NVIDIA performance.

[Full changelog](https://github.com/shruggietech/insonic/blob/v1.1.0/CHANGELOG.md)
