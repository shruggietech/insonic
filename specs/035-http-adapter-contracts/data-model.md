# Data model: S035

No persisted schema migration is required. Extraction configuration accepts canonical insonic-http and legacy http, then retains canonical adapter identity. Existing recording revision/document digest and per-chunk cue IDs remain publication authority. Provider responses cannot introduce timestamps or foreign source identity. Operator errors retain category plus an additive locally generated reason; materialization/native/provider bounds retain output_limit. No credentials or provider bodies enter diagnostics.
