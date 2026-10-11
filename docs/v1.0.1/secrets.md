# Credential commands

The source-buildable CLI and shared desktop bridge configure credentials independently of catalog access. This lets you provision a PostgreSQL credential before the runtime opens that catalog. Native persistent storage, an encrypted vault and explicit session credentials share the same workspace-scoped UUID references. [Credentials and data access](security.md) defines routing and access boundaries.

## Choose the store

```sh
insonic --workspace WORKSPACE credentials select native --json
insonic --workspace WORKSPACE credentials select vault --json
insonic --workspace WORKSPACE credentials select session --json
```

Choose one backend for the workspace. The selection persists in the private control directory, without credential values. Native storage uses Windows Credential Manager, macOS Keychain or Linux Secret Service. A locked or unavailable service returns a fixed rejected/unavailable result; it does not display an unlock prompt, start a session bus or switch to another backend. Unlock the operating system's service through its own controls, or explicitly select the vault/session alternative.

Credential IDs are UUIDs scoped to the workspace identity. Configuration, catalog snapshots, import/model manifests and jobs retain only those references. Copying encrypted vault bytes into a different workspace does not grant access because the workspace identity is authenticated with the ciphertext.

## Enter and replace values

`add` and `replace` read one bounded JSON object from standard input. There is no credential-value command-line option. `INPUT_PRODUCER` below denotes your protected input source, not an insonic command. The desktop `Credential` bridge passes newly entered input through the same command service.

```sh
INPUT_PRODUCER | insonic --workspace WORKSPACE credentials add CREDENTIAL_UUID --json
INPUT_PRODUCER | insonic --workspace WORKSPACE credentials replace CREDENTIAL_UUID --json
insonic --workspace WORKSPACE credentials status CREDENTIAL_UUID --json
insonic --workspace WORKSPACE credentials delete CREDENTIAL_UUID --json
```

The protected input object contains `value`. A string stores its UTF-8 bytes; an object stores its JSON representation for an adapter that expects structured authentication. These examples describe the input shape and contain placeholders:

```json
{"value":"NEWLY_ENTERED_KEY"}
```

```json
{"value":{"username":"DATABASE_USER","password":"NEWLY_ENTERED_PASSWORD"}}
```

```json
{"value":{"access_key":"ACCESS_KEY","secret_key":"SECRET_KEY","session_token":"OPTIONAL_SESSION_TOKEN"}}
```

`add` rejects an existing registration or a native credential left by an interrupted registration write. `replace` requires an existing registration or native credential and changes it only through the explicit replacement operation. Explicit replacement or deletion can reconcile that interrupted native registration. A credential value is limited to 2,048 bytes, within the Windows native-storage limit; each selected store accepts at most 256 registrations. Unknown fields, duplicate JSON keys, malformed input and oversized values fail with fixed errors that do not repeat the supplied input.

Status returns the credential ID, selected backend and `configured`, `missing` or `rejected`. It checks the managed nonsecret registration information without retrieving a saved value. `configured` is registration status; resolving the credential and using the selected adapter establishes whether the saved value exists and whether the provider accepts it. An external deletion from the operating system's credential store may therefore become visible only during adapter use.

Deleting a credential leaves its references in profiles and other configuration.
Subsequent credential resolution and new authenticated connections report
unavailable. Established provider sessions follow the provider's lifetime and
revocation behavior; deleting a local reference does not revoke an authenticated
PostgreSQL server session. S3 resolves the selected reference for each subsequent
request. Reconfigure or replace the selected credential to restore access.
Native deletion uses no input; vault deletion needs its passphrase through the
protected input object.

## Encrypted vault

Vault input adds a `passphrase` string to the same protected object:

```json
{"value":{"username":"DATABASE_USER","password":"NEWLY_ENTERED_PASSWORD"},"passphrase":"NEWLY_ENTERED_VAULT_PASSPHRASE"}
```

The vault fixes Argon2id version 19 at 64 MiB, three iterations, four lanes and a 32-byte derived key. AES-256-GCM authenticates the encrypted credential map and its format, KDF parameters, workspace ID and nonsecret credential-ID list. Each vault has a random 16-byte salt; each update has a fresh 12-byte nonce. The implementation uses maintained [Go Argon2](https://pkg.go.dev/golang.org/x/crypto/argon2) and [authenticated-encryption](https://pkg.go.dev/crypto/cipher#NewGCM) libraries. Untrusted files cannot select a larger work factor or downgrade the algorithms.

The private control directory stores ciphertext and nonsecret selection/registration data. Updates lock exclusively, reread the current map and atomically replace a synced file, so concurrent owners do not discard each other's changes. The passphrase and derived key remain in process memory and are cleared when the manager closes; neither is saved beside ciphertext. Credential encryption applies to these credentials, while media/artifact encryption follows the selected storage configuration.

```sh
INPUT_PRODUCER | insonic --workspace WORKSPACE credentials unlock --json
```

For `unlock`, send `{"passphrase":"NEWLY_ENTERED_VAULT_PASSPHRASE"}` through standard input. The runtime accepts the passphrase through its protected bootstrap channel and retains the unlocked manager for that owner's lifetime. After the runtime exits, unlock again before an operation requiring vault credentials. Status without an unlocked manager reports rejected. A wrong passphrase, tampered ciphertext or unsupported envelope returns unavailable without exposing its contents.

## Explicit session credentials

Session values remain in the runtime owner's memory and disappear when it exits. Load a UUID-to-value mapping through protected input:

```json
{"session":{"CREDENTIAL_UUID":{"username":"DATABASE_USER","password":"NEWLY_ENTERED_PASSWORD"}}}
```

```sh
INPUT_PRODUCER | insonic --workspace WORKSPACE credentials load --json
```

Once loaded, `add`, `replace`, `delete` and `status` operate on that live manager. A standalone short-lived manager does not claim that an ephemeral change was persisted. The explicitly selected `INSONIC_SESSION_CREDENTIALS` environment mapping remains available for automation, using UUID keys and string/object values. It is consulted only for a workspace that selected session mode; a native/vault failure never falls back to that mapping. Loading a new explicit mapping replaces the current session input rather than persisting it.

The runtime's normal idle shutdown ends vault unlocks and session credentials. Long-running configured work keeps the owner alive for its active operations. CLI and desktop responses contain status information; neither provides a command to display a saved value.
