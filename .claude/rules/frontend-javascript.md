# Frontend: plain JavaScript, not TypeScript

Frontend projects in this workspace (currently `workspace/frontend/admin`) are
written in **plain JavaScript**: `.js` for modules and `.jsx` for components.
This was the developer's decision (EC-1, 2026-10-14).

- Don't add TypeScript, `tsconfig.json`, `.ts`/`.tsx` files or `@types/*`
  packages, and don't convert existing files.
- Document data shapes with JSDoc `@typedef` / `@param` where it helps.
- Project-specific conventions: `workspace/frontend/admin/CLAUDE.md`.
