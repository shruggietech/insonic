# S068 data model

QuestionRequest binds question, normalized filters, traversal, continuation, retrieval budgets and elected provider/model/configuration/summary version. Limits are shared and validated before work. No endpoint or secret values are accepted as result metadata.

EvidenceReference binds workspace UUID, recording UUID/revision, document/source/map digests, cue identifier and ordinal, exact nullable source-clock range, mapping/date identity and public source-family facts. Revisions/times that may exceed JavaScript exact integers use decimal strings in the public evidence contract.

EvidenceSet contains immutable ordered items, selection paths, component scores, source-family groups, exclusions, conflicts, incomplete/insufficient state and measured budget usage. Catalog/graph observation is separate from semantic source identity. Unknown lineage is explicit; claims of acoustic verification are absent.

ContextPackage contains source-preserving extracts grouped by cue, section, recording and cross-recording, explicit links and bounded omissions. It records full serialized byte/conservative-token counts, selected evidence digest and condensation contract. A context layer is never an independent source.

AnswerEnvelope defines version, state, claims, citations, uncertainty and conflicts. Supported claims require package-member citations and faithful quotes/exact known timing. This slice validates this envelope but does not generate synthesized answers.

SavedPackage is an immutable verified artifact with a retained reference. CachePointer is a small workspace-scoped CAS setting containing the semantic key and publication/digest/version identity. Freshness is recomputed; historical stale artifacts remain inspectable. A cache miss/corruption produces a new artifact and reconciled pointer, without mutating source data.
