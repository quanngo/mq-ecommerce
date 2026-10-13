---
name: plan-ticket
description: Turn a fetched Jira ticket into an implementation plan for this workspace — which frontend/backend projects are affected, proposed API contract, schema changes, UI placement, and test strategy — written to .workflow/<TICKET>/plan.md and plan.json. Use after /jira-ticket, or as phase 2 of the deliver.sh pipeline.
---

Produce the implementation plan that later phases execute from. The ticket key
is given as the argument (e.g. `PROJ-123`).

This plan is consumed by automated phases with no human in the loop after the
approval gate, so it has to be **decisive and grounded**: every technical claim
must come from a file you actually read, and every gap must be classified as
either a recorded assumption or a blocking open question. Never guess silently.

## Steps

1. **Read the requirement**: `.workflow/<TICKET>/ticket.md`. If it doesn't
   exist, stop and tell the developer to run `/jira-ticket <TICKET>` first.

2. **Read the workspace context**:
   - Root `CLAUDE.md` (overall architecture) and `projects.env` (project keys,
     names, start commands, and any `*_URL_n` / `*_HEALTH_n` entries)
   - For each project that could plausibly be affected:
     `workspace/<type>/<key>/CLAUDE.md`, anything under that project's
     `.claude/rules/`, and `API_DOCUMENT.md` for backends
   - Root `.claude/rules/` for workspace-wide conventions

3. **Decide the targets.** Which side(s) and which specific project keys this
   ticket touches. If the ticket genuinely could belong to more than one
   project and the docs don't settle it, that is a **blocking** open question —
   don't pick arbitrarily.

4. **Write `.workflow/<TICKET>/plan.md`** — human-readable, for the approval
   gate:

   ```markdown
   # Plan: <TICKET> — <summary>

   ## Goal
   <2-3 sentences: what changes for the user when this is done>

   ## Affected projects
   | side | project key | why |

   ## Backend work
   <endpoints to add/change with method + path, handler/service/repo layers,
   validation, auth. Reference the project's existing conventions by file.>

   ## Database / schema changes
   <tables/columns/relations + migration approach, or "none">

   ## Proposed API contract
   <method, path, request shape, response shape, status codes, auth. This
   becomes contract.md in the backend phase.>

   ## Frontend work
   <pages/routes/components, state management, where in the UI this lives>

   ## Test strategy
   <backend unit tests, frontend unit tests, and the end-to-end scenarios
   derived from the acceptance criteria that browser QA will automate>

   ## Acceptance criteria (verifiable)
   <numbered, each phrased so a test can pass/fail it>

   ## Assumptions (proceeding with these)
   <non-blocking gaps filled with a documented, reasonable default>

   ## Open questions (blocking — must be answered before implementation)
   <schema, auth rules, which project, or contract ambiguities that would
   produce wrong code if guessed. Empty is a valid and good answer.>
   ```

5. **Write `.workflow/<TICKET>/plan.json`** — machine-readable for later
   phases:

   ```json
   {
     "ticket": "<TICKET>",
     "targets": [{"side": "backend", "project_key": "api"}],
     "acceptance_criteria": ["..."],
     "assumptions": ["..."],
     "open_questions": ["..."]
   }
   ```

6. **Classify gaps carefully** — this is the most important judgment in the
   phase:
   - **Assumption** (non-blocking): a sensible default exists and a wrong guess
     is cheap to change (e.g. page size of a list, sort order, a label).
   - **Open question** (blocking): a wrong guess produces wrong code or data
     loss — DB schema shape, auth/permission rules, which project owns the
     feature, or a contract the frontend must match exactly.

7. **Report**: the target projects, the acceptance criteria count, and — read
   out explicitly — every assumption and open question. If there are any open
   questions, state clearly that implementation should not proceed until the
   developer answers them at the approval gate.
