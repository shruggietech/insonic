# S029 release state and identity

No runtime/schema data shape changes.

| Entity | Required fields and validation | State relationship |
| --- | --- | --- |
| Release cut | canonical version 1.0.1, UTC cut date, unique changelog heading, highlights <=180 words ending in exact-tag master changelog | prepared source remains candidate |
| Candidate | full reviewed squash-main revision, one workflow attempt, four archive receipts, one MSI receipt, two help bundles, complete sources/notices and matching hashes | qualified before draft publication |
| Release | immutable v1.0.1 tag resolves exact candidate revision; complete verified nine public assets; explicit unsigned disposition | draft may become public only after remote verification |
| Documentation | release-target marker matches version/source; original published archive bytes; exact Pages commit | public verification follows promotion |
| Checkpoint | S029/#97, PR status, review count <=2, thread resolution and CI evidence | In progress → In review → publication pending → Done after verified public delivery |

Mutable manifest records known historical publication independently of frozen documents. Release-target rendering elects latest candidate; a render-only prose overlay reflects any released entry without mutating its source. Incomplete uploads remain draft; failed promotion can retry original published assets.
