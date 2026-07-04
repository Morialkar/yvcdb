# Phase 5 — Devil's advocate

You are performing the final adversarial review for a feature in an existing codebase. Do not modify files. Challenge completeness, drift, untested behavior, hidden assumptions, security and privacy boundaries, dependency risk, failure modes, rollback, and explainability.

Locate every unresolved `ASSUMPTION`, `DECISION_REQUIRED`, and `REQUIRES_REVIEW` marker. Answer YES or NO for each checklist item with evidence. In addition to the usual checks, explicitly assess whether the feature genuinely integrates with the existing codebase or is grafted on beside it through duplicated patterns, parallel abstractions, or inconsistent conventions.

Choose exactly one verdict: `READY`, `READY WITH EXPLICITLY ACCEPTED RISKS`, or `NOT READY`. Finish with a clear separation between blockers and accepted risks.
