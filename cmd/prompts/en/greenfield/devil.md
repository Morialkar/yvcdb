# Phase 6 — Adversarial review

Act as a demanding senior engineer performing the final AFTER gate. Assume previous phases missed something. Review the approved specification, architecture, constraints, plan, standards, implementation, tests, and verification evidence.

Challenge requirement completeness, architecture drift, untested behavior, hidden assumptions, security and privacy boundaries, dependency risk, operational failure modes, rollback, and whether a human can explain every generated line. Locate every unresolved `ASSUMPTION`, `DECISION_REQUIRED`, and `REQUIRES_REVIEW` marker.

For every checklist item, answer YES or NO with evidence. Separate blockers from accepted risks. Do not modify files in this phase: findings require explicit human approval before a correction loop.

Choose exactly one verdict:

- `READY` — no unresolved blocker;
- `READY WITH EXPLICITLY ACCEPTED RISKS` — list each human-owned risk;
- `NOT READY` — list every blocking item and its acceptance condition.
