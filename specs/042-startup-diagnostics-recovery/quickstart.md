# S042 validation

1. Run root check and tests with `npm run check` and `npm test`.
2. Run focused Go diagnostics and desktop tests under the repository's existing native environment. Use injected storage/presentation and application runner sinks; local tests must not display a window.
3. On Windows, exercise inherited Modify and explicit access, denied storage, existing/rotated logs, foreign ownership relative to selected caller identity, replaced paths/reparse points and concurrent locks. Inspect owner and protected ACLs on fixture-owned files.
4. Verify returned-error wrapping, original-cause feedback despite denied diagnostics, and omission of arbitrary error/path/credential text from persisted generated events.
5. For the second increment, run focused fake-COM/unshown-window startup failure, cleanup, late-callback and fresh-instance regressions, plus pinned-framework loop/cleanup ordering tests. Verify maintained source and notice manifests before packaging.
6. Run affected site check/tests/build and UTF-8/LF/diff checks when current documentation changes.
7. Publish a frozen increment, retain complete CI/gate/terminal evidence, merge with passing required checks, and verify exact main. Existing unattended Windows native/package CI supplies genuine input, natural release and the controlled recovery sequence without an owner-desktop exercise.
8. Update issue criteria individually. Do not close #154 solely because this slice supplies useful improvements; unresolved historical cause evidence remains explicit. Keep S041 campaign evidence on its final S041 revision.
