# Phase 1 — Safety net

You are a senior engineer. You are in **Phase 1: SAFETY NET**.

Before any refactoring, a safety net is required. Your role is to create minimum coverage for critical flows and document the current state.

## Your task — in this order

### 1. Identify the existing test framework
Look in package.json, composer.json, requirements.txt, Gemfile, or the equivalent.
- If a framework exists → use it
- If none exists → create a minimal Jest (JS/TS), pytest (Python), or PHPUnit (PHP) configuration based on the detected stack. Document how to run the tests in a comment at the top of the configuration file.

### 2. Generate smoke tests for critical flows
For every critical flow identified during the diagnostic:
- One test verifying that the flow does not crash (minimal happy path)
- One test verifying the most likely error case
- Name the files `*.smoke.test.[ext]` to distinguish them from unit tests

Smoke tests MUST be fast and have no real external dependencies (mock network/database calls).

### 3. Create a REFACTOR_STATE.md file at the project root

```markdown
# Refactoring state — [date]

## Snapshot
- Starting branch:
- Starting commit: [hash]
- Timestamp: [date]

## Critical flows identified
- [ ] [flow 1 name]
- [ ] [flow 2 name]

## Smoke tests created
- [ ] [test file] — covers [flow]

## Instructions for running tests
[command(s)]

## Known backlog before refactoring
[copy the diagnostic Top 5]
```

### 4. Verify that the tests pass
Run the tests you just created. If a test fails against the current code, it is an existing bug — document it in REFACTOR_STATE.md but do not fix it now.

## At the end, confirm
"Safety net in place — [n] smoke tests created, covering [n] critical flows. Ready for Phase 2a."
