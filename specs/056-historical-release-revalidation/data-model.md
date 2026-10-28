# Data model

- HistoricalProfile: immutable version, revision, tree, manifest SHA-256, MSI name/size/SHA-256 and canonical original receipt identities. Input data cannot create a profile.
- SourceProof: every pinned tracked member, mode, Git blob identity and matching selected-source bytes. Missing/changed/nonregular members fail; before/after proof encloses candidate validation.
- Candidate: original immutable manifest and nine matching published assets. The manifest binds package/installer receipt identities and all seven payload/archive hashes; checksum bytes remain checked.
- Acceptance evidence: selected profile/source, original asset sizes/digests and successful maintained checks, distinct from fresh runtime qualification or installed product behavior.
