# S051 validation

1. Run the maintained focused receipt suites: `python -m unittest discover -s tests -p "journey*_test.py"`. Missing/malformed source authority must be refused, valid restored and legacy source-only controls must pass.
2. Recover the gzip/base64 receipt from [retained actual CPU evidence](https://github.com/shruggietech/insonic/pull/179#issuecomment-6434152258). Decode and gunzip into owned temporary storage, then verify SHA-256 `e45118f36954a8486cb65289fdd3a68bfb9cb3f7906dc9127c93d667889c3e29`.
3. Load the frozen acoustic corpus through the maintained reference loader and call `journey_cpu.validate_acceptance` on the unchanged receipt. Record the original package/harness revision and current validator hash; report revalidation time separately.
4. Run root `npm run check`, `npm test`, full Python tests and affected site checks. Review final-head CI and individual bot findings, retain compact audit evidence and remove the owned temporary inputs before owner handoff.

See [data-model.md](data-model.md) for authority requirements and [plan.md](plan.md) for execution scope. These checks establish corrected evidence validation; the original retained receipt owns the runtime observations.
