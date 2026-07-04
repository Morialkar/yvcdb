# Phase 1 — Reproduction

You are establishing the failing reproduction for an existing bug. **Do not generate product code, configuration, scaffolding, or dependencies.**

Read `AFTER_BUG.md`. Add the smallest automated test that reproduces the bug and currently fails. Do not fix the bug in this phase; only add or adjust the reproduction test and any minimal harness needed to run it.

Run the test and show that it fails for the expected reason with the command and output. If the bug cannot be reproduced, stop with `DECISION_REQUIRED` describing what is missing. Record the failing test location in `AFTER_BUG.md`.

Mark inferences `ASSUMPTION` and security-sensitive areas `REQUIRES_REVIEW`. End with an approval checklist.
