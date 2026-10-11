# S018 CLI contract

Global options may occur before/after command tokens outside literal --. Option values consume one token atomically; equals syntax follows the same validation. Scalar duplicates fail; repeatable selectors retain order. Unknown commands/options fail before access. Help resolution honors value/literal boundaries and precedes required operand validation.

Machine success: one runtime-response on stdout, exit 0. Machine failure: one runtime-response on stderr with error and any useful result/identity, exit 2 invalid_request, 3 incompatible_version, 4 unavailable, 1 other. No logs pollute either JSON envelope. Human version is the exact version string; machine help/version use structured result. Write/encode failure returns nonzero with bounded fallback diagnosis.

Help supports root/group/leaf -h/--help and help COMMAND, aliases, examples/defaults/limits and JSON-file schema/template guidance. Literal help tokens after -- remain data.

Numeric values preserve declared numeric types and exact signed decimal tokens. Work failures retain safe typed causes after restart, while partial accepted items retain existing exit/state semantics.
