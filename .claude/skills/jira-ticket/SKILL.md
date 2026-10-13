---
name: jira-ticket
description: Fetch a Jira ticket via the Atlassian MCP connector and write a normalized requirement snapshot to .workflow/<TICKET>/ticket.md. Use when asked to read/pull a Jira ticket, understand what a ticket requires, or as phase 1 of the deliver.sh pipeline.
---

Fetch one Jira ticket and turn it into a clean, self-contained requirement
document. The ticket key is given as the argument (e.g. `PROJ-123`).

Ground everything in what the ticket actually says. **Never invent
requirements, acceptance criteria, or technical decisions that aren't in the
ticket** — a later phase plans and implements from this file, so anything
fabricated here becomes wrong code downstream. Missing information must be
listed as missing, not filled in.

## Steps

1. **Fetch the issue** with the Atlassian MCP tools:
   - `getJiraIssue` with the ticket key. You may need
     `getAccessibleAtlassianResources` first to resolve the `cloudId`.
   - If the key doesn't resolve directly, try `searchJiraIssuesUsingJql`
     (e.g. `key = PROJ-123`).
   - Also fetch the comments and any linked issues if the tool returns them —
     comments frequently change or narrow scope.

   If the Atlassian connector isn't authorised, **stop and report that**: tell
   the developer to connect it (claude.ai connector settings, or `/mcp` in an
   interactive session) and exit without writing a file. Do not proceed from
   guessed ticket content.

2. **Write `.workflow/<TICKET>/ticket.md`** (create the directory if needed)
   with this structure — omit a section entirely if the ticket has nothing for
   it, rather than padding it:

   ```markdown
   # <TICKET>: <summary>

   - **Type**: <story/bug/task>   **Status**: <status>   **Priority**: <priority>
   - **Reporter**: <name>   **Assignee**: <name or unassigned>
   - **URL**: <browse link>

   ## Description
   <the description, converted from Jira markup to plain markdown>

   ## Acceptance criteria
   <verbatim if the ticket has them; if not, say "None stated in the ticket">

   ## Scope notes from comments
   <only comments that add, remove, or clarify scope — skip chatter>

   ## Linked issues / attachments
   <keys + titles, attachment filenames>

   ## Missing / unclear for implementation
   <explicit list: unstated DB schema, unclear UI placement, undefined API
   contract, unstated auth rules, ambiguous which project is affected, etc.>
   ```

3. **Normalize the markup.** Jira descriptions arrive in Jira wiki markup or
   ADF — convert headings, lists, tables, and code blocks to plain markdown so
   later phases read it cleanly. Preserve the content faithfully; don't
   summarize away detail.

4. **Report** to the developer: the ticket key and summary, where the file was
   written, and the "Missing / unclear" list — that list is what the plan phase
   and the human approval gate will act on.
