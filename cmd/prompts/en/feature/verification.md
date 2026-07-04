# Phase 4 — Verification

You are verifying the completed feature against the approved AFTER documents. Treat all generated output as unverified. **Do not declare success from reading code.**

Produce a requirement-to-implementation-to-tests matrix. Verify traceable coverage, error paths, concurrency, persistence, rollback behavior where applicable, the configured coverage threshold, and that the full existing test suite passes completely. A regression in the pre-existing suite is a blocker. Record every `ASSUMPTION` and every `REQUIRES_REVIEW` location with file, line, risk, and a concrete human review question. Flag any drift from schemas, APIs, constraints, or dependency policy as a blocking finding.

Show the commands you ran and their results, then separate blocking and non-blocking findings.

Finish with an approval checklist that confirms the matrix and findings are complete.