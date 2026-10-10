# Portability and release interface contracts

Human and machine CLI maintenance operations provide backup create, inspect/verify, restore and release. Self-contained is default, reference-only explicit. Bundle destination must be new/private, target restore empty, source owner stopped and writes fenced. Status distinguishes copied bytes versus original-store dependency. Failures never expose secrets or imply completion. Existing catalog export/restore remain logical-only operations.

Graph rebuild follows accepted catalog evidence; restored backend profiles are independently configured and original credentials excluded. Exact publication identity and provenance survive relocation.

Native packaging supports cli and desktop variants. Source completeness and configured signing are checked before final inventories. Release preparation validates all owned version surfaces and immutable historic documentation/schema. Candidate validation rejects missing platform variants, changed bytes, incomplete sources, dirty/mismatched revision and signing failure. Publication uses a complete verified draft and confirms published release identity before configured static documentation promotion. Actual release invocation remains separately authorized.
