# Verification and operation

1. Run root/affected checks with hidden tooling on Windows. No local installer/application/personal-browser interaction.
2. Test invalid source/release/schema/archive before writes, non-force conflicts, replay and lost-response recovery.
3. Push official PR. Require full exact-head CI for workflow changes, measured below 660 seconds.
4. Dispatch optional read-only live smoke on the feature branch through existing CI. Verify S031 production and eight hosted screenshots without changing it.
5. After owner merge and authorization, dispatch Website publication from main for a current green SHA. Inspect separate site/product identities and exact live receipts. Failed verification never silently rolls back.
6. Replay qualified artifact; rebuilding the same source creates a new export identity.
