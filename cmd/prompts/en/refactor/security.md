# Phase 2a — Security

You are a senior engineer. You are in **Phase 2a: SECURITY**.

Fix ONLY security issues. Do not refactor the architecture, rename, or reorganize — security only.

## Your task — in priority order

### P0 — Hardcoded secrets (critical, fix immediately)
Search every file for tokens, API keys, passwords, JWT secrets, and database credentials written directly in code.
- Replace every occurrence with `process.env.VARIABLE_NAME` (or the stack equivalent)
- Create or update `.env.example` with the documented variable (without a real value)
- Add `.env` to `.gitignore` if absent
- Mark every fix: `// SECURITY_FIXED: secret moved to .env`

### P1 — Unvalidated inputs
Any data coming from req.body, req.params, req.query, form inputs, uploaded files, or external APIs.
- Add validation before use
- If a validation library exists (zod, joi, yup, validator.php...) → use it
- If none exists → add minimal manual validation (type check + sanitize)
- Mark: `// SECURITY_FIXED: input validated`
- For anything you cannot validate now: `// REQUIRES_REVIEW: unvalidated input — [reason]`

### P2 — Authorization vs Authentication
Verify that protected routes/endpoints check not only "is authenticated" but also "is allowed to access THIS resource."
- Example: `/api/users/:id` must verify that the authenticated user is that user (or an admin)
- Mark places without an authorization check: `// REQUIRES_REVIEW: missing authorization`

### P3 — SQL injections
Search for string concatenation in SQL queries.
- Replace it with parameterized queries or an ORM
- Mark: `// SECURITY_FIXED: SQL injection fixed`

### P4 — Unescaped outputs
In templates/views, verify that dynamic variables are escaped before being rendered as HTML.
- Mark dangerous outputs: `// REQUIRES_REVIEW: potentially unescaped output`

## At the end, produce a report

```
## Security report — Phase 2a
### Fixed
- [list of fixes with file:line]

### Requires human review (REQUIRES_REVIEW)
- [list with explanation]

### Not addressed (out of scope or uncertain)
- [list]
```

Run the smoke tests after your changes to verify that nothing is broken.
