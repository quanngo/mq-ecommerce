---
name: backend-api-docs
description: Generate or refresh API_DOCUMENT.md for every project under workspace/backend/ by scanning each project's actual route/handler code. Creates the file if missing, regenerates it from the current code if it already exists. Use when asked to document backend APIs, create/update API docs, or after backend endpoints have changed.
---

Maintain a standard-Markdown `API_DOCUMENT.md` at the root of every project
under `workspace/backend/`. Source code is the source of truth — never invent
or guess endpoint behavior; mark anything you can't determine from the code
as "undocumented in code".

## Steps

1. **Enumerate backend projects** — list every directory under
   `workspace/backend/`. Each one is a separate project (matching a
   `BACKEND_KEY_*` entry in the root `projects.env`). Skip any that are empty
   or missing and note them in the final summary as "not cloned — run
   ./setup.sh".

2. **For each backend project, independently:**

   a. Detect the backend stack/framework from the project (e.g. Go with
      Gin/Echo/net-http/chi, Node with Express/Fastify/NestJS, Python with
      FastAPI/Flask/Django, etc.) — don't assume one, read `go.mod`/
      `package.json`/`requirements.txt`/`pyproject.toml` or equivalent first.

   b. Scan the actual route/handler/controller registrations to build the
      real API surface. For each endpoint capture, only from what the code
      shows:
      - HTTP method + path
      - Purpose (short, based on handler name/logic/comments)
      - Request: path/query params, body shape and types
      - Response: shape/types, success status code
      - Error cases / status codes it can return
      - Auth requirement (middleware/guard applied, if any)

      Use the project's own `CLAUDE.md` (if present, from `workspace-init`/
      `setup.sh`) as a cross-reference/starting hint only — always verify
      against the actual source before writing anything down.

   c. Check whether `API_DOCUMENT.md` already exists at the project root.

      - **Missing** → create it with this structure:
        ```markdown
        # <Project Name> API Documentation

        Base URL: <if discoverable from code/config, else "not documented in code">

        ## <Resource/Group 1>

        ### METHOD /path
        - **Description**:
        - **Auth**:
        - **Request**: params / body
        - **Response**: shape + status code
        - **Errors**: status codes + meaning
        ```
        Group endpoints by resource/router prefix. Add a table of contents at
        the top if there are more than ~8 endpoints.

      - **Already exists** → re-scan the code and reconcile the file with
        current reality: add newly-found endpoints, remove ones no longer in
        code, correct changed method/path/params/response. Preserve the
        existing file's section structure/ordering and any hand-written prose
        (overview, auth explanation, general notes) that isn't a stale
        endpoint listing — this is an update, not a blind append or full
        rewrite from scratch.

3. **Report a summary** at the end, one line per project: project key,
   whether `API_DOCUMENT.md` was created / updated / left unchanged (already
   current) / skipped (not cloned), and the number of endpoints documented.
