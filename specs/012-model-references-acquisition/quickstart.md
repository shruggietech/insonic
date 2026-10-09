# S012 validation guide

1. Register two synthetic complete bundles with exact hashes and compatibility; create an alias, inspect resolution and reject a stale concurrent retarget.
2. Configure a loopback fixture catalog, discover declared entries and resolve one missing compatible model. Submit more than four deterministic processing consumers and verify acquisition progresses without worker starvation.
3. Cancel one consumer, interrupt shared acquisition, change alias/source selection, restart and retry. Verify exact elected identities survive and remaining consumers finish after verified acquisition.
4. Exercise import attribution and pipeline references, wrong roles/runtime, corrupt cached bytes and dependency failure. No step loads an engine or real weights.
5. Exercise CLI and rendered desktop selection/alias/discovery/work phases, populated schema-7 migration and portable schema-8 snapshot parity. Run root/frontend/site/Go/Python checks and native/backend qualification.
6. Publish the authorized official PR, attach it, handle all reviews within two rounds and verify all required checks on its final head. Stop for owner final review and merge.
