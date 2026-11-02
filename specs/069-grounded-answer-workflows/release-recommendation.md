# Release recommendation

## Compatibility and proposed release

Recommend **v1.2.0** after the evidence-grounded answer milestone is merged and its final CI/review evidence is accepted. This is additive functionality: existing preparation, queries, source authority, artifact backends and published contracts remain supported. New structured synthesis requires an explicit quote/paraphrase kind; the older standalone answer-envelope validator still accepts its prior shape. Answer retry creates a linked work attempt and preserves prior history.

The current development identity remains **1.1.0** in software, versioned documentation and the master JSON Schema. A separate authorized release-preparation slice must advance all three together to 1.2.0, generate release references and qualify the exact release candidate. This slice neither tags nor publishes a release. The proposed minor version does not require a major-release terminology audit.

## Milestone outcomes

- #220–#223 supplied bounded retrieval, exact current-source references, provenance families and hierarchical context in the preceding merged slice.
- #224 adds configured HTTP/Ollama synthesis, visible-context citation and quote validation, finite repair and explicit insufficient/conflicting answers.
- #225 adds durable CLI/shared-runtime request, phases, wait, inspection, export, cancellation, linked retry and publication recovery, with frozen pipeline elections and atomic source/work acceptance.
- #226 adds the versioned synthetic evaluation corpus, distinct retrieval/provenance/support metrics, an explicit real-model runner, installed CLI/backend campaigns and this recommendation.

The final PR should close the existing three children and parent #210 only after these outcomes have their required evidence. No tracking issue is created.

## Qualification and limitations

The deterministic corpus covers corroboration, duplicates, overlap, contradiction, unanswerable questions, withheld evidence and instructions embedded in source text. Every applicable citation, quote, retrieval, independence, support and uncertainty metric must pass; unsupported supported-claim count must be zero. A zero denominator is explicitly not applicable. Support matches unique frozen propositions and their supporting sources, so repeated copies cannot inflate the numerator. Unrecognized paraphrases fail conservatively.

The Windows source-built CLI passes all seven corpus cases and the failure/retry, cancellation, partial-budget, export, unchanged-source and stale-source journey. The same source-built CLI journey passes against PostgreSQL, S3 and ArcadeDB. Exact packaged Windows/Linux and Linux alternative-backend evidence is recorded in the final verification/handoff receipt. Protocol/provenance regression evidence does not establish semantic truth or acoustic independence.

No real model evaluation was elected for this slice. The maintained elected runner records adapter, exact model, route, configuration digest, corpus identity, measured results and failing examples without sending reference labels to the provider. No claim of general answer accuracy is made. Dedicated desktop answer controls remain unimplemented and are documented as such; CLI and shared runtime expose the capability.

## Proposed release highlights

- Ask questions across recordings with inspectable source citations, faithful quotes and explicit evidence limitations.
- Run configured answers unattended with durable progress, cancellation, linked retries and immutable exports.
- Inspect source freshness and independently measured retrieval, citation and support results.

Full details: [CHANGELOG.md](../../CHANGELOG.md).
