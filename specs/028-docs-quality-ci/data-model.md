# S028 data model

No product record/schema migration is added.

- DocumentationManifest.latest selects exactly one existing version. Entries retain version, status and ordered pages. Candidate status is not publication.
- CurrentAlias has a stable path, manifest-selected version/topic, canonical fixed-version URL and rendered article or explicit missing-current-topic fallback. Fragments resolve against actual rendered headings. Archive entries keep explicit version authority.
- DocumentationFrame has version/status, selected topic, previous/next, section links and disclosure state. Native closed state is useful without JavaScript; viewport enhancement does not hide document content.
- CIBudget is660seconds. Current workflow/qualification/policy checks use this bound, preserving unrelated operation deadlines.
- TestLeaseFixture retains exact workspace/role/owner/generation and guarded durable deadline. Fixture transitions live->expired are explicit; production Claim/Renew implement observed conflict/takeover/fencing semantics.
- StandardsEvidence records source revision, templates/content pages, viewport/theme/keyboard/noJS/file/overflow observations, isolated screenshot paths and known limitations. Evidence is internal and never rendered as public planning.
