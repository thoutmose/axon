# 0012 — Delivery agents read repositories; they never approve or merge

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-26
- **Builds on:** [0008](0008-agents-propose-people-approve.md)

## Context

The Side-Projects agent analyses a repository and drafts issues; the Code Review agent reviews pull requests. Repository content is untrusted: a README or a diff can carry instructions aimed at an AI. The backlog requires read-only access by default, approval for drafted issues, and a reviewer that can comment but never approve or merge.

## Decision

- **Connector:** a per-person GitHub connection (owner/repo, optional fine-grained token sealed with `AXON_ENCRYPTION_KEY`, an "allow comments" switch that is off by default, and the project board for drafted issues). It is managed from browser sessions only.
- **Client surface:** the client can read the default branch's file tree, read files (100 KB), list open pull requests, and read a diff (200 KB). Its one write is a plain issue comment (`POST /repos/{o}/{r}/issues/{n}/comments`), refused unless comments are allowed. The code contains no review, approval, merge, push, or contents-write call. That is how the guarantee holds, not by prompting.
- **Tools:** five read tools, plus `draft_issue` (a card on the person's own board) and `post_review_comment`. Both writes are proposals.
- **External writes are never dry-run.** Proposals are normally validated by applying them in a rolled-back transaction, but a GitHub comment cannot be rolled back. Tools with external effects declare `Check`, which validates without acting, so the comment is posted exactly once, after approval. Posted comments are signed as drafted by the agent and approved by the owner.
- Repository content reaches the model framed as data, and the skills tell the agents to report injected instructions rather than follow them.

## Alternatives

- **A GitHub App with webhooks:** automatic reviews, but it needs public reachability (Axon is tailnet-only) and broader permissions.
- **Letting the agent submit a GitHub review:** the review API's `event` field includes `APPROVE`, so that surface was left out entirely.

## Consequences

Agents work on public repositories without any token. Private ones need a read token, and comments need a narrowly scoped write permission the person adds on purpose. GitLab and other forges would each need a client with the same small surface.
