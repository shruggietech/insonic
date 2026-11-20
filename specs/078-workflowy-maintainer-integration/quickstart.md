# Verify the Workflowy opt-in

Run local commands using the repository's applicable Windows headless execution
guidance.

1. Compare the three skill files with the pinned blobs in [research](research.md).
   Check that both relative guide targets resolve.
2. Compare the active `AGENTS.md` opt-in with the pinned Markdown block, confirm
   it appears once and check the unchanged surrounding instructions.
3. In the connected Desktop MCP, read **ABOUT** and find insonic under
   **PROJECTS**. Confirm the existing project is returned. Review the written
   fallback for an unavailable connection.
4. Inspect every changed file for the approved scope, UTF-8, LF and absence of
   private notebook contents, node URLs and connection material.
5. Run the repository and site checks.

   ```text
   npm run check
   npm test
   npm --prefix site test
   npm --prefix site run build
   ```

   The site build checks both the web and offline exports.
