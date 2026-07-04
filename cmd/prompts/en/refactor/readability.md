# Phase 2c — Readability

You are a senior engineer. You are in **Phase 2c: READABILITY**.

The code now does what it should, securely and with a sound structure. It must now be understandable to someone who has never seen it.

## Your task — in this order

### 1. Split functions and files that are too long

**Functions over 40 lines**
- Identify internal logical blocks
- Extract every block into a named private function whose name describes its intent
- The original function becomes a readable sequence of named calls

**Files over 300 lines with multiple responsibilities**
- Propose and apply a split into files by responsibility
- Update all imports

**Rule**: every split must preserve behavior. When in doubt, write a test first, then split.

### 2. Naming

Rename in these cases:
- A variable named `data`, `result`, `temp`, `item`, `x`, or `val` → a name describing its contents
- A function named `handle`, `process`, `do`, or `run` without a qualifier → `handleUserLogin`, `processPaymentRefund`, etc.
- A boolean without an `is/has/can/should` prefix → `isLoading`, `hasPermission`, etc.
- Magic number → named UPPER_CASE constant with a comment when not obvious
- Repeated magic string → shared constant

Document every rename in a block at the end of your response.

### 3. Intent comments

Add comments explaining **why** (not what) for:
- Workarounds and hacks (`// HACK: works around a [lib] v[x] bug — see issue #[n]`)
- Non-obvious decisions (`// Intentional: prefer X over Y because Z`)
- Complex conditions (explain the business rule, not the syntax)

Do not comment on what is obvious. `// Increment the counter` before `count++` is noise.

For anything you do not understand: `// UNCLEAR: expected behavior unknown — do not modify`

### 4. Documentation for public exports

For every exported public function/class/method, add minimal documentation following the project standard (JSDoc, Python docstring, PHPDoc, etc.) with:
- What it does (one line)
- Non-obvious parameters
- What it returns when non-obvious
- Possible exceptions/errors

### 5. Backlog

For anything you identify but do not fix now:
`// REFACTOR_BACKLOG: [concise description of the issue]`

## At the end

```
## Readability report — Phase 2c
### Renames completed
- [old] → [new] in [file] (reason)

### Functions split
- [original name] → [list of new functions] in [file]

### Files reorganized
- [old] → [new files]

### REFACTOR_BACKLOG items identified
- [file:line]: [description]
```

Run the tests. Confirm that they still pass.
