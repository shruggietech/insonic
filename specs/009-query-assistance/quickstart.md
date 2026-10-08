# Quickstart validation

1. Build source CLI/desktop using existing development instructions and start an explicit compatible local or hosted query-assistance service.
2. Inspect `insonic query assistance-show`; save elected settings with its expected revision and the contract fields documented in [assistance](contracts/assistance.md).
3. Submit `insonic query assist --input question.json` with a prompt. Confirm an inspectable proposal/explanation and no execution in suggestion mode.
4. Elect mode auto-run and confirm the shown proposal plus current read-only results. Submit malformed/wrong-dialect output from a deterministic service and confirm no execution.
5. Use desktop Explore Query assistance, adopt the complete proposal, edit/run/save it, and verify prompt/editor changes discard a late response.
6. Run repository acceptance and mounted source/relocated package journeys. They inject deterministic assistance replies, never initialize models or run inference.
