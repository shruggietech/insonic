# Native runtime research

**2026-10-07 dependency maintenance**: Maintained Cueson package/schema references now identify the required v1.2.0 contract. Historical qualification results describe their original session and do not establish later package execution; S005 records the exact current-version qualification.

**2026-10-07 dependency maintenance**: Maintained Cueson package/schema references now identify the required v1.2.0 contract. Historical qualification results describe their original session and do not establish later package execution; S005 records the exact current-version qualification.

## Decisions

- Decision: Go 1.27.1, stable Wails v2.14.0. Rationale: exact maintained release identities; local Go 1.24.2 cannot build current Wails. Alternative: use Go automatic exact toolchain acquisition rather than change the machine installation.
- Decision: go-winio v0.6.2 and flock v0.13.1. Rationale: maintained current-user pipe/OS locking integrations. Alternatives rejected: TCP listeners and stale PID ownership.
- Decision: jsonschema v6.0.3 embeds existing Draft 2020-12 authority. Preserve exact JSON numbers and disallow network schema loading.
- Decision: Cueson v1.2.0 official executable/schema, with encode/validate/render/restore on real fixture bytes.
- Decision: Ladybug core v0.21.2 and binding revision 42bbf464c74c59088f59691dbd9f204015be3463. Use system_ladybug linking and SHA-256 checked assets rather than mutable upstream download scripts.
- Decision: native secret availability qualification without real user credentials; complete encrypted persistence stays #5.
- Decision: actual three-OS operation receipts, no cross-compile substitution.

## Sources

- [Go release metadata](https://go.dev/dl/?mode=json)
- [Wails release](https://github.com/wailsapp/wails/releases/tag/v2.14.0)
- [Wails platform dependencies](https://v2.wails.io/docs/gettingstarted/installation/)
- [Pinned Ladybug binding](https://github.com/LadybugDB/go-ladybug/tree/42bbf464c74c59088f59691dbd9f204015be3463)
- [Ladybug core release](https://github.com/LadybugDB/ladybug/releases/tag/v0.21.2)
- [Cueson release](https://github.com/shruggietech/cueson/releases/tag/v1.2.0)
- [Pipe release](https://github.com/microsoft/go-winio/releases/tag/v0.6.2)
- [Lock release](https://github.com/gofrs/flock/releases/tag/v0.13.1)
- [Validator release](https://github.com/santhosh-tekuri/jsonschema/releases/tag/v6.0.3)

## Native packaging

Unix Ladybug archives include SONAME aliases; preserve or safely materialize them. Windows links lbug_shared and requires OpenSSL/VC runtimes. macOS core requires deployment target 13.3+. Close database, connections, query results and tuples in binding probes. Fixture identities and upstream asset digests belong in a committed lock; actual execution remains acceptance.
