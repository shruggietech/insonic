# S066 evidence model

No product schema or persistence migration is introduced.

| Entity | Required fields and relationships | Validation |
| --- | --- | --- |
| AcceptedSource | version 1.1.0; revision 82a1bfd7e8f4e389d451789ec7840d874127f39c; exact-main CI 41270520010 | Clean complete source; reviewed protected-main history; successful source checks |
| ReleaseCampaign | run ID; attempt; workflow head; selected input revision; original jobs/steps/artifact inventory; terminal conclusion | Trusted workflow identity; complete pagination; actual terminal timestamps; workflow head kept separate from product source |
| RetainedCandidate | original ZIP ID/name/size/SHA256; ten asset entries; original package and installer receipts; documentation targets | Unique unexpired attempt artifact; confined regular paths; exact ZIP digest; complete loader checks; release-target docs |
| PublishedRelease | immutable tag; release ID/URL; source revision; all ten public assets; concise highlights | No tag retargeting or asset replacement; public bytes equal accepted candidate |
| PagesDeployment | expected parent; archive identity; promoted commit; completed build; configured destination | Non-force history; exact commit completion; unchanged domain/HTTPS/history contracts |
| PublicVerification | fresh public download directory; ten-asset comparison; exact-site byte receipt; Pages fences; eight-case render report | Original receipts unchanged; actual binary versus API metadata versus rendered-byte evidence distinguished |
| DeliveryLedger | S066 issue/milestone/project; evidence PR/head; review triggers/results/threads; final checks; missing evidence | At most two triggers; every actionable finding closed; green final applicable CI; PR remains open for owner |

Candidate lifecycle: prepared -> qualified -> retained/validated -> tagged -> published -> independently retrieved. Documentation lifecycle: published original archive -> staged parented tree -> promoted exact commit -> completed build -> independently verified. Failed states retain diagnostics and do not silently advance or roll back.
