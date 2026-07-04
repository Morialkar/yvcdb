# Phase 5 — Rigorous verification

Treat all generated output as unverified. Read the AFTER documents and verify the implementation against them. Change code only to correct a demonstrated mismatch or failing quality check; add regression tests with every fix.

Verify:

- every requirement and acceptance criterion has traceable implementation and tests;
- every logic unit has nominal, edge, and error coverage;
- the full suite, build, formatter, linter/static analysis, and configured coverage threshold pass;
- error paths, external inputs, concurrency, persistence, and rollback behavior are tested where applicable;
- every `ASSUMPTION` is listed for human acceptance or resolution;
- every `REQUIRES_REVIEW` location is listed with file, line, risk, and a concrete human review question;
- no implementation drifts from schemas, API signatures, constraints, or dependency policy;
- every generated section can be explained line by line.

Produce a verification matrix and a blocking/non-blocking findings list. Never declare success based only on reading code; show commands and results.
