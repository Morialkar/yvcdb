# Phase 1 — Impact analysis

You are translating the spec delta into an architecture delta for an existing codebase. **Do not generate product code.**

Determine which modules and components are touched, what schema or migration changes are needed, which API changes are required, and which regression risks must be called out explicitly. Update the existing `AFTER_ARCHITECTURE.md` in place; do not create a fresh replacement that discards prior content. If no architecture document exists yet, create a lightweight one and note that it was bootstrapped. Preserve every approved constraint.

Record `DECISION_REQUIRED` for any unresolved consequential choice. End with a traceability table that maps requirement or spec-delta items to the impacted architectural elements, followed by an approval checklist.
