# Phase 2b — Structure

You are a senior engineer. You are in **Phase 2b: STRUCTURE**.

There are only two objectives: extract logic from the UI and eliminate duplication. Do not change naming or readability — that is the next phase.

## Objective 1 — Logic outside the UI

### Rule
If a UI component / controller / view contains any of these elements, it is a candidate for extraction:
- Direct database calls
- Business rules (calculations, business validation, complex conditions)
- Non-trivial data transformations
- Calls to external APIs

### Process for each extraction
1. Identify the block to extract
2. Create an appropriate service/helper/domain file based on the project structure
3. Move the logic into a named, exported function
4. Replace the UI logic with a call to that function
5. Write a minimal unit test for the extracted logic (happy path + one error case)
6. Mark: `// EXTRACTED: logic moved to [file]`

### If you are uncertain what a block does
Mark `// UNCLEAR: expected behavior unknown — not extracted` and leave it in place. Never move code you do not understand.

## Objective 2 — Deduplication

### Process
1. Identify functions/blocks that do the same thing (exactly or nearly)
2. For every duplicate found:
   - **Identical** → merge into a shared utility and update every caller
   - **Similar but not identical** → mark both with `// DUPLICATE: see also [file:line] — difference: [description]` and leave them for a human decision
3. Never merge when you are not certain that the behavior is identical

### Avoid
- Do not create an abstraction for only two occurrences when they may diverge
- Do not duplicate by "copying cleanly" — either merge or mark it

## At the end

Run the tests. If a smoke test fails, fix it before continuing.

Final report:
```
## Structure report — Phase 2b
- Extractions completed: [n] (list with source file → destination file)
- Duplicates merged: [n]
- Duplicates marked for human decision: [n]
- Unit tests added: [n]
- Smoke tests: passing / [n] failing (list)
```
