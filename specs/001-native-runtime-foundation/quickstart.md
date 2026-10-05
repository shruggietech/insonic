# Runtime foundation validation

## Local checks

1. Use the exact Go toolchain from go.mod and Node 22+.
2. Run go test ./..., go vet ./... and build cmd/insonic.
3. Initialize a temporary workspace through workspace init.
4. Invoke workspace show from two clients and compare session identity.
5. Exercise jobs start/show/cancel/retry after client disconnection.
6. Run npm run check, npm test and npm --prefix site run build.

Windows maintainer tooling uses scripts/run-hidden.ps1 with literal executable arguments and redirected noninteractive I/O.

## Native acceptance

CI runs actual Windows/macOS/Linux dependency and IPC operations, desktop bridge and offline help, plus bounded alternative services. Missing native execution is failed/pending evidence, never a passing substitution.
