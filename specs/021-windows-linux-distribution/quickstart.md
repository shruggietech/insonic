# S021 validation guide

1. Run repository/schema, Python, Go and frontend/site checks.
2. Exercise candidate fixtures with exactly four current qualified packages; reject missing/extra/duplicate targets, corrupt inventory and wrong revisions. Validate historical 1.0.0 records under their historical contract.
3. Assemble/extract Windows CLI and desktop packages into spaced paths and follow version/help/workspace init/doctor examples with a separate disposable library.
4. Exercise helper success, source descendant/alias/existing-target rejection and forced partial-copy/substitution failures only in disposable owned fixture directories.
5. Run remaining Windows/Linux qualification and candidate collection in CI; verify all review threads and exact final head before owner merge handoff.
