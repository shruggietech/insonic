# S052 research

- Decision: real platform terminals. Rationale: stderr isatty is the actual product gate; pipes and supplied booleans cannot establish it. Alternatives: visible console/GUI input violates owner constraints; OS signal alone bypasses terminal input.
- Decision: standard-library ConPTY and Linux controlling PTY. Rationale: no new package; fresh Linux interpreter avoids unsafe preexec/fork from worker threads. Windows needs suspended Job assignment, continuous output draining and natural-exit observation before ClosePseudoConsole.
- Decision: two-role acquisition dependency with separate runtime owner. Rationale: verified-files checkpoint provides stable count progress; cancelling the ensure parent would leave its dependency active. Runtime idle exit provides an independently observed release boundary.
- Decision: strict nested platform receipt and maintained coverage scenario. Rationale: a successful top-level package marker alone does not prove the new campaign. No public API contract is added.
- Primary references: [Microsoft ConPTY lifecycle](https://learn.microsoft.com/en-us/windows/console/creating-a-pseudoconsole-session), [process flags](https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags), [Python PTY](https://docs.python.org/3/library/pty.html), [Linux controlling terminal](https://man7.org/linux/man-pages/man2/TIOCSCTTY.2const.html). Existing code references are identified in plan.md.

All research unknowns resolved. No owner decision needed.
