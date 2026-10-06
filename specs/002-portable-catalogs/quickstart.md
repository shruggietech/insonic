# Validation guide

1. Run npm check/test, Go test/vet and Python tests through the hidden Windows launcher.
2. Run shared catalog acceptance against SQLite: reopen, CAS/idempotency, scoped FKs, exact values, immutable evidence, recovery/fencing and ordered outbox.
3. Set INSONIC_FIXTURE_POSTGRES to the disposable fixture, then run `go test -tags integration ./internal/catalog`. Exercise the same suite plus SQLite/server round trips. Fixture environment routing is not production profile routing.
4. Initialize a temporary workspace, start/cancel/retry a timer job and inspect history; restart and retain history. Interrupt work and observe fresh recovery without obsolete publication.
5. Use catalog export/restore with an empty destination preserving workspace identity. Compare receipts/records and reject imported active claims.
6. Test CI classification for docs, catalog/runtime, native/dependencies, schema and unknown paths. Retain stable checks and conservative fallback.
