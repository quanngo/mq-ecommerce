---
name: write-code-be
description: Implement a feature on the backend only — understand the relevant backend project, ask the developer to clarify anything ambiguous (DB schema, API contract, auth), implement the endpoint(s)/schema changes, write backend unit tests, update the backend API docs, and add backend-side integration tests. Does not modify frontend code. Use when asked to build/implement a feature on the backend only, or to expose a new API for the frontend to consume later.
---

Implement a feature on the **backend only**, grounded strictly in what the
existing project's code and docs actually show. Never fabricate framework
conventions, schema, or API shapes that aren't present — prefer asking the
developer over guessing. This skill never writes or modifies frontend code.

## Steps

1. **Capture the requirement.** Restate the developer's feature request in
   your own words as a first check you understood it correctly.

2. **Understand the frontend (read-only, optional).** If it's useful to know
   what a consumer expects, check the relevant `workspace/frontend/<key>/CLAUDE.md`
   for context on how the frontend is likely to call this API. This is
   optional context gathering only — this skill never modifies frontend code.

3. **Understand the backend.** Read the root `CLAUDE.md` and `projects.env`
   for the `BACKEND_KEY_*` entries. For each backend project the feature
   could touch, read `workspace/backend/<key>/CLAUDE.md` and its
   `API_DOCUMENT.md` if present (produced by the `backend-api-docs` skill) to
   learn the framework, DB schema, existing endpoint/handler conventions, and
   test setup. If multiple backend projects exist and it isn't obvious which
   one the feature belongs to, ask the developer.

4. **Ask clarifying questions before writing any code** whenever the request
   leaves any of the following unclear — do not guess or invent defaults:
   - Technical approach/design, if there's more than one reasonable way to
     build it given the existing backend architecture
   - Data model / DB schema changes (new tables/columns, relations)
   - API contract specifics (request/response shape, auth/permissions,
     validation rules, edge cases/error handling)
   - Which backend project is affected, if ambiguous from step 3

   Once these are answered, proceed straight through implementation — there
   is no separate design sign-off step.

5. **Implement the backend.** Build schema/migration changes if needed, then
   handler/service/repository code for the new or changed endpoint(s) in the
   relevant `workspace/backend/<key>/` project, following that project's
   existing framework and conventions exactly. Don't introduce a new
   library, pattern, or folder structure it doesn't already use.
   In the Go backend (EC-7), anything a user reads is translated: a new error
   code needs `error.<CODE>` in both `i18n/locales/vi.json` and `en.json`
   (it becomes `error_text`; the code in `message` never changes), and names or
   labels go through `i18n.T(i18n.Locale(c), key)`. Per-user settings are
   columns of `public.user_config`, not new tables.

6. **Implement the frontend — not applicable.** This skill never writes
   frontend code. If a frontend also needs updating to use this API, tell
   the developer to run `/write-code-fe` or `/write-code-full` next.

7. **Integrate.** Verify the implemented endpoint matches the contract
   clarified in step 4 exactly (and any known frontend expectations
   gathered in step 2), by re-reading your own new handler code — don't just
   assume it matches what was discussed.

8. **Write unit tests (backend only)**, using the project's existing test
   framework/conventions: handler/service/repository-level tests for the new
   logic. Don't introduce a new test tool if one is already configured; if
   none is set up, ask the developer before adding one. Frontend unit tests
   are not applicable here.

9. **Update backend API docs.** For every backend project whose API surface
   changed, follow the same approach as the `backend-api-docs` skill
   (see `.claude/skills/backend-api-docs/SKILL.md`): re-scan the project's
   route code and reconcile its `API_DOCUMENT.md` — create it if missing,
   update it in place (preserving existing structure and hand-written prose)
   if present. Scope this to only the backend project(s) touched by this
   feature.

10. **Integration test the feature (backend side).** If the backend project
    already has an integration/API test setup (e.g. supertest-style tests
    hitting a running server, or equivalent for its framework), add a test
    exercising the new/changed endpoint(s) end-to-end at the API layer,
    following that project's existing conventions. No frontend is involved.
    If no such setup exists, say so explicitly and ask the developer whether
    to add one rather than silently introducing new tooling.

11. **Final report.** Summarize: files changed (backend only), tests added,
    whether `API_DOCUMENT.md` was created/updated, and whether a frontend
    counterpart (via `/write-code-fe` or `/write-code-full`) is still needed.
