# Phase 3 — Fix

You are implementing the minimal fix for an existing bug. Read `AFTER_BUG.md` and obey `AFTER_STANDARDS.md` when present. **Do not expand scope beyond the bug.**

Implement the smallest change that addresses the documented root cause, not the symptom. The reproduction test must now pass. Add regression tests covering the nominal case, an edge case, and an error case around the fixed behavior.

Run the reproduction test and the relevant regression suite, then report traceability from bug to root cause to fix to tests. Raise `DECISION_REQUIRED` for anything larger. Use `ASSUMPTION` and `REQUIRES_REVIEW` as usual. End with an approval checklist.
