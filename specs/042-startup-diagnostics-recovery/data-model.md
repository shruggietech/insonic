# Startup failure and private event model

A startup failure retains a generated stage and original wrapped error. Application stages identify workspace selection, playback transport, desktop window startup and desktop window close. A wrapped typed native StartupError refines desktop-window only to one of fourteen generated binding boundaries, including environment-create/completion, controller-request/completion/configuration, webview, bootstrap-script, message-loop, window-destroyed and browser-process. Unknown native strings cannot enter persisted metadata. A numeric system code may be extracted from the error chain without persisting its arbitrary text.

User feedback contains an actionable cause, corrective advice, diagnostic-save status and the saved diagnostic location when available. It is bounded and presented independently of browser initialization. Secondary failures cannot replace the primary cause.

Private events reuse bounded JSONL logging, current-user access, locking and rotation. Only operation desktop.startup may retain the allowlisted startup stage and optional unsigned 32-bit system error code. Invalid stage/operation combinations clear those supplementary fields. Arguments, paths, environment, arbitrary errors and protected input remain absent.

Windows file securing starts with a path-confined regular-file handle and verifies a separately opened handle has the same volume/file identity and no reparse attribute. Existing foreign-owned files reject. A new file already owned by the caller needs only DACL changes; a new file needing owner normalization may request the additional right only after observing that need, with identity revalidation. The resulting DACL remains protected and current-user restrictive.

The second increment's startup state moves pending -> succeeded or pending -> failed exactly once. The first failure stage/cause remains authoritative. Failed/null native results and interrupted initialization cannot publish success. Teardown happens after callback unwinding, closes owned resources once and prevents late completion from restoring state. A fresh process has independent startup state and uses the intended profile.
