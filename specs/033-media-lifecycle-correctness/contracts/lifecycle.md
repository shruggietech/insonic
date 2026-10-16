# Corrected lifecycle contracts

| Boundary | Observable result | Issues |
| --- | --- | --- |
| Refresh acceptance | Success when accepted; lease/reference retirement remains pending and retryable; real errors stay observable | #143 |
| Playback open | MIME, size and digest identify materialized canonical/preview bytes; source metadata is preserved | #141 |
| Embedded selection | Shipped companion accepts supported text tracks and retains source timing/Unicode | #142 |
| Transcript import | Manifest uses per-item targets; positional import requires --record; conflicting forms fail before effects | #140 |
| HTTP acquisition | Deadline/byte bounds have stable redacted causes and supported corrections through durable work/reload | #145 |

Existing request/response versions and supported backends remain authoritative. Any error-code additions must update registered error schemas, human corrections and tests together. #114/#130/#147 remain open.
