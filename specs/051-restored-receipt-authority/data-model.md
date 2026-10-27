# S051 internal authority rules

- Source recording: media_id is a canonical UUID string, unique across all six source cases. revision has exact integer type and is >=1; no default or coercion.
- Restored recording: media_id equals its source; explicit integer revisions satisfy source revision <= prior_revision < revision. Distinct fixture/source identities imply distinct destination recordings.
- Malformed evidence: absent/null/noncanonical/unhashable identifiers and missing/boolean/noninteger/nonpositive source revisions receive ValueError through the maintained validator.
- Historical receipt: source-only v1 is validated without restored-only requirements; restored v2 retains its existing shape.
- Revalidation evidence: raw receipt digest and original source/package/harness/reference bindings, current validator input digest and elapsed validation time; original campaign phase times remain separate and immutable.
