# Phase 4 — Verification

You are verifying the bug fix against the approved AFTER documents. Treat all generated output as unverified. **Do not declare success from reading code.**

Confirm the reproduction test now passes and that it genuinely fails without the fix, stating how you know. Run the full pre-existing test suite; any regression is a blocker, not a note. Verify the coverage threshold, error and edge paths around the fix, and no drift from constraints.

Produce a bug to root cause to fix to tests matrix and a blocking or non-blocking findings list. Show commands and results. List every `ASSUMPTION` and `REQUIRES_REVIEW` with file, line, risk, and a concrete human question. End with an approval checklist.
