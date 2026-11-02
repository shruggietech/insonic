# S071 publication evidence model

- AcceptedSource: full 40-character source revision, version 1.2.0, reviewed merge and green main run. Distinct from tooling/PR head.
- Candidate: exactly ten named files, nonnegative byte sizes, SHA-256 digests, four portable package receipts, MSI/setup qualification, release-target public/offline documentation. Complete and immutable after acceptance.
- Campaign: run/attempt, input revision, workflow revision, terminal jobs/steps/artifacts and truthful overlapping lane timings.
- Publication: immutable tag peeled to AcceptedSource, published release, complete original asset identity; draft state is not published.
- Deployment: expected parent, staged/pushed/deployed commit, documentation inventory and completed exact build; concurrent/unrelated commits fail.
- Readback: independent saved downloads and digest comparison, public content/resources, eight render cases, explicit model/signing/security dispositions.
- EvidencePR: separate S071 head, capped review ledger, final checks, project status and owner merge boundary.

State sequence: source accepted -> candidate qualified/retained -> tag -> draft upload verified -> release published -> exact Pages build -> independently verified -> evidence PR ready. Failed/uncertain states retain exact artifacts and receipts; no blind rebuild or silent rollback.
