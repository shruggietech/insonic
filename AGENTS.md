# Agent instructions

## Owner-directed collaboration

Treat the owner as a capable collaborator who defines intended product behavior. Exercise engineering judgment and explain deviations. Do not silently narrow the product or introduce eligibility checks, approvals, moderation systems or unrelated prerequisites based on hypothetical misuse. Separate approved requirements, technical necessities, recommendations and limitations on assistance. Unapproved reports are not binding requirements. Identify any operation you cannot perform before editing, and obtain agreement before materially reducing scope.

Finish authorized work autonomously. Report changed behavior, performed checks, unverified behavior and remaining deviations with concrete references. Plans, partial implementations and passing unrelated tests do not constitute completion.

## Project authority

Read `.specify/memory/constitution.md` and the current documentation under `docs/`. v0.0.0 is a specification baseline; do not claim product commands or installable packages exist. Use Spec Kit for iterative slices and GitHub Issues, milestones and Projects as the primary tracking system once published.

The application is CLI-first. Implement core operations in shared runtime contracts and keep the GUI wrapper aligned; document advanced or experimental CLI features that arrive before GUI controls. Preserve early raw metadata/date provenance, cross-library speaker segments and optional speaker-associated training/model retrieval. Defaults are filesystem, SQLite and LadybugDB; generic S3, PostgreSQL and ArcadeDB are fully supported alternatives, not unofficial future aspirations.

Run configured import, transforms, attribution, jobs and elected training without adding mandatory per-item reviews. Existing route and publication authorization persists. Ask for concrete unresolved date interpretation, unauthorized destructive effects, new external routing or material scope changes; do not repeat permission for routine configured work.

Keep public documentation and planning self-contained. Do not cite private external design material or carry source-specific names, corpora, paths, prompts or eligibility rules into the product. Upstream owns the brand; read `brand/kit/enforcement/consumer-contract.json`, `brand/kit/enforcement/IMPLEMENTATION.md` and relevant upstream guidance before applying it. Update governed kit bytes only through `scripts/sync-brand-kit.mjs`.

Official documentation states authoritative system contracts. Keep slice codes and implementation evidence in internal `specs/` planning, and historical/version-change prose in `CHANGELOG.md`. Do not render or link numbered planning and tracking artifacts from the public documentation. Keep one concise specification-status statement in the documentation index. Render the master changelog and JSON contracts within the documentation site.

The software, versioned documentation and master JSON Schema MUST share the release version. Prefer backward and forward compatibility where practical without treating either as an absolute promise. Explain breaking changes in the changelog and concise release notes. For every major release, rescan all documentation for technical terms and refresh the glossary with precise definitions, external primary learning references and local usage links.

## Workflowy

This project opts into the `workflowy` skill and its private agent workspace.
Load the skill when available. Use the operator's project link or find this
repo or folder under **AI Agent Workspace > PROJECTS**. Reuse its branch or
insert it alphabetically. Search **STANDARDS** by title or topic tag. Read the
applicable rules, defaults, and scopes before governed work.

Use normal bullets for candid discussion, drafts, decisions, and evidence.
Never store work in Workflowy comments or comment drafts. Give `info` exactly
three direct children: `tags`, `links`, and `details`. Put purpose, current state,
next actions, blockers, and supporting evidence under `details`.
Use lowercase letters and underscores for reusable field names. Use inline
code for exact technical literals. Link to the exact canonical finding or evidence.
Git and the established tracker govern official progress where used.
Otherwise use the project artifacts. The operator decides what is published.

In `notes`, keep major sessions and substantive updates as a flat, oldest-first
list. Use only the ISO 8601 date or timestamp as the title. Retain the finest
supported precision and offset. Put the description, scope tags, and prose
under `title`, `tags`, and `details`. Use label-only field bullets with each
value or link nested in its own bullet. Require the Workflowy bullet format
for fields and values. Only the contents of `details` may use other formats.
Keep tags out of names and other fields. Use one hashtag per `tags` value.
Put any project-level tags in `info > tags`. Link related updates. The operator
owns timeline construction. Agents record dates without creating timeline
metadata, views, saved searches, or exports.

For Spec Kit, use one direct `spec-kit` bullet per formal slice in code order.
Use only its code and short title. Nest the verified date under `date` and
the documented epic under `epic` when applicable. Use the skill for shared
reference records, migration, and precise field rules.

Desktop MCP requires a running, signed-in Workflowy app with Local MCP
enabled and a connected client. A cloud or other nonlocal agent uses a
separately configured, operator-authorized connector when available. Confirm
the destination and read back writes. If no connector is available, continue
under the available instructions and provide an appropriately private,
ready-to-paste nested outline with its exact destination. Mark it unsynced.

## Quality and file integrity

Identify poor logic and fix or propose a proportional improvement. Do not replicate bad code during porting or refactoring. Save UTF-8 without BOM, LF line endings, and check for mojibake before delivering files. Keep Markdown soft-wrapped. Use numbered steps for a purely linear procedure; use diagrams for meaningful branches, relationships, dependencies, feedback or retry paths, keeping Mermaid flowcharts top-down. Sequence plans and update logs chronologically. Avoid unnecessary em-dashes and canned rhetorical contrasts.

## Windows execution

Foreground, flashing or focus-stealing console windows are prohibited. Direct Git and GitHub operations through the headless command runner are allowed. Batch compatible read-only checks. Before unverified non-Git console tooling, use repository-aware APIs, direct file tools or a launcher guaranteeing `CREATE_NO_WINDOW` and redirected noninteractive I/O. Stop a launcher that produces visible windows and continue unrelated verified paths. Project child launchers must hide console windows and disable prompts.

## Verification and publication

Run `npm run check`, `npm test` and the affected site/product checks. Target thirty-minute (1,800-second) qualification after verified prerequisite readiness, with a sixty-minute (3,600-second) hard limit. Expand maintained test execution allowances consistently through nested callers, native/frontend waits and local/CI/release/diagnostic wrappers; retain ordinary product-operation deadlines and intentionally short injected failure clocks. Target-only overruns are advisory and do not block otherwise valid acceptance or merge. Preserve the sixty-second provisional finalization reserve, terminal timestamp authority, separate preparation/qualification/combined accounting, and mandatory functional, integrity, provenance and elected checks. Retain compact audited receipts outside the workflow run before a full rerun; attempt-qualified artifact names prevent collisions but do not establish platform retention. User observations normally follow releases as issues; do not invent an attended pre-release verification gate. Follow the owner-approved squash merge and branch-deletion model. Release notes are short highlights ending with a link to `CHANGELOG.md`. No first push or GitHub publication is authorized by the local foundation request.
