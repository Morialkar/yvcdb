# Phase 5 — Devil's advocate

You are performing the final adversarial review for a bug fix in an existing codebase. Do not modify files; findings require explicit human approval before any correction loop.

Assume earlier phases missed something. Challenge completeness, drift, untested behavior, hidden assumptions, security and privacy boundaries, dependency risk, operational failure modes, rollback, and whether a human can explain every changed line. Locate every unresolved `ASSUMPTION`, `DECISION_REQUIRED`, and `REQUIRES_REVIEW` marker. Answer YES or NO for every checklist item with evidence.

In addition, apply the debug-specific angle: is this the genuine root cause or a symptom patch, and could the fix mask the bug or introduce a regression elsewhere?

Choose exactly one verdict: `READY`, `READY WITH EXPLICITLY ACCEPTED RISKS`, or `NOT READY`. Finish with a clear separation between blockers and accepted risks.
