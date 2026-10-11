# Research decisions

Three read-only Spec Kit research agents examined current code and live configuration; no external configuration or operator state changed.

| ID | Decision | Rationale | Alternatives considered |
| --- | --- | --- | --- |
| D01 | Reuse native qualification for release and PR. | Release retained core cache, generic CLI and serial both-variant work. | Maintain duplicated workflows (drift), extend timeouts (reject). |
| D02 | Split engine versus desktop/credential runtime lanes and cache identities. | S015 final Windows runtime was 533s versus 488s, native step 177s versus 132s; package paths were shorter. Shared immutable native cache could be first-saved by desktop without runtime variants. | Only package parallelism (insufficient), skip checks (reject). |
| D03 | Shared native CLI build helper prepares desktop sibling explicitly. | macOS desktop currently depends on earlier CLI acceptance output; new UI lane must stand alone. | Repeat CLI acceptance in both lanes (avoidable), ambient CLI (reject). |
| D04 | Explicit reusable documentation target and signing inputs. | Publication requires release-target docs; snapshot CI defaults and signing-before-hashes remain. | Blanket released source metadata (incorrect), mix snapshot packages (reject). |
| D05 | Built-in existing legacy Pages transport plus generic option. | Existing Pages is gh-pages:/ at insonic.io, HTTPS enforced, with github-pages policy allowing gh-pages; repo vars/secrets absent. | New host/PAT, Actions-source migration (unnecessary scope/config changes). |
| D06 | Validate original published archive, atomic nonforced ref and exact build result. | Token pushes do not themselves trigger legacy Pages; upload alone is insufficient. Polling must identify exact staged commit. | Unverified branch upload/force push (reject). |
| D07 | Preserve absent publisher signing identities as explicit unconfigured state. | Variables do not provision runner identities; ad-hoc signing is not publisher notarization. | Invent certificate prerequisite or signing success (reject). |
| D08 | Bundle fixed processing worker under companion inventory. | Packages lacked worker while source journey selected repository bytes. | New manifest field (unnecessary strict-parser expansion), ambient worker (reject). |
| D09 | Verify/extract package then derive all runtime paths. | Existing local packages are baseline, so new reviewed v1 artifact is required. Remove dev loader/tool fallbacks. | Source rebuild or reuse baseline package (invalid evidence). |
| D10 | Keep actual Windows CPU processing elected and stage product launch separately. | Existing model pins and fixture journey are available; acoustic counts/matches are not accuracy ground truth. Squash merge changes source identity. | CI inference/downloads, attended user gate, immediate product publication (reject). |

Primary Pages behavior sources: [publishing source](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site) and [Pages REST builds](https://docs.github.com/en/rest/pages/pages#request-a-github-pages-build). Actual API deployment remains unexecuted until separately elected; isolated tests do not prove hosted publication.
