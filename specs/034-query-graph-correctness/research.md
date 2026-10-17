# Research: S034

## 2026-10-17

- Decision: separate row/edge offsets inside the existing opaque cursor. Rationale: current same-page endpoint filtering permanently drops edges; later-endpoint assignment still concentrates an arbitrarily dense hub into one bounded response. Alternatives: unbounded response rejected; independent public edge API unnecessary. Consumers retain stable IDs and may receive edges before both endpoints.
- Decision: move pinned zone loading and platform-local mapping beneath library/contracts. Rationale: contracts cannot import library because library already depends on contracts. Preserve pinned ZIP checksum and CLDR version. Alternative: host time.LoadLocation rejected because Windows packages lack consistent IANA lookup and fixed/local semantics differ.
- Decision: save-specific title constraint. Rationale: run/explain inputs may legitimately omit title. Contextual examples derive from schema and must also execute in runtime.
- Decision: existing catalog.ValidateJSON before native parameter decoding. Rationale: it already rejects duplicate members without float-round-tripping numbers.
- Decision: one trailing semicolon and existing pure functions. Rationale: improves conventional query input while retaining statement/effect guards. Diagnostics use stable allowlisted reasons rather than unbounded query echo.

- Decision: preserve the detached runtime idle lease and check Windows write/delete-access release for at most ten additional seconds. Hosted desktop archive qualification passed both GUI probes but cleanup encountered a mapped insonic.exe after the fixed 31-second sleep. Mapped image sharing blocks write-data access; requesting write-data and delete access without disposition observes release without modifying the image, terminating a process or ignoring cleanup failure. The overall 660-second qualification deadline remains unchanged.
