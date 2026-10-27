# S052 internal evidence model

- Terminal observation: scenario ID, actual platform/transport, work ID, request ID, executable SHA256, exit code, natural exit, observed progress/count, classified result and elapsed/effective bound. Raw output and host paths are not retained in the receipt.
- Terminal campaign: format version, executable SHA256, platform, transport, required scenario map and cleanup state. Missing/duplicate/unknown scenarios, noncanonical work IDs, nonfinite timing, mismatched identities, overflow, forced exit or incomplete cleanup are refused.
- Lifetime: create private workspace and explicit owner -> verified ready -> held work -> terminal observation -> natural client exit -> independently verified durable outcome -> released server/runtime/channels -> fixture removal -> accepted receipt.
- Ctrl+C changes client wait lifetime only; explicit cancellation changes durable work. Timeout records a last observation and does not invent current terminal authority.

Case durable_state records the independently checked final durable outcome. It does not relabel an interrupted or timed-out client snapshot as success. Human request IDs are elected argv identities; JSON echoes are validated, and request-correlated running diagnostic events gate the JSON/quiet/redirected suppression controls. One absolute 600-second campaign allowance bounds nested launches, terminal observations, barriers and ordinary idle exit; cleanup retains its separate bounded reserve and cannot promote expiry to acceptance.
