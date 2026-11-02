# S067 Validation

1. Use the existing installed repository and site dependencies. Keep local commands hidden and noninteractive on Windows.
2. Run `python -m unittest discover -s tests -p pages_test.py`, then the release, website and public-verification suites. Delayed/lost readbacks recover; unrelated writers, wrong tree/parent/build, altered archive and receipt-write failures reject.
3. Run `npm run check`, `npm test`, `npm --prefix site test`, `npm --prefix site run check` and `npm --prefix site run build`.
4. Inspect receipts from fixtures for exact identities and sanitized observations. Recovery performs only GET and does not overwrite the original attempt.
5. Review current release-guide timing against policy and executed callers. Preserve original v1.1.0 assets and deploy nothing during qualification.
6. Push/open PR linking #240 and #241, resolve at most two external review rounds and require green CI before owner merge handoff.
