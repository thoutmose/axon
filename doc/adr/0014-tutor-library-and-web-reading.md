# 0014 — Tutor and Resource agents: a curated library, allowlisted reading, and conversations

- **Status:** Accepted
- **Date:** 2026-10-11
- **Epic:** EP-27
- **Builds on:** [0008](0008-agents-propose-people-approve.md), [0013](0013-roadmaps-licences-and-graph.md)

## Context

The Tutor agent should interview the person, then draft a roadmap grounded in recognised sources. The Resource agent should find the best material for a skill. Both need sources the model cannot invent, and some way to read course pages. A tool that fetches any URL would let an injected instruction send data out in a query string. Agents had no way to ask a question and wait for the answer.

## Decision

- **Curated library.** Axon ships `roadmaps/library.json`: 24 domains (CS, algorithms, maths, OS, Linux and Bash, C/C++, Java, Python, Rust, networking, DevOps and GitOps, Docker/Podman/Kubernetes, web and full stack, API design, system design and data-intensive systems, databases, data engineering and big data, data analysis, data science, machine learning, MLOps, security and DevSecOps, cloud, soft skills). It holds 258 widely recommended books, papers, university courses, videos, channels, and practice sites. Only titles, authors, kinds, a free flag, and links are stored, so nothing is redistributed. Every link was checked when the library was written. Each domain imports as an ordinary roadmap the person can edit or delete.
- **Citations are required.** `draft_roadmap` refuses a skill without an http(s) source link, and prerequisites must name earlier skills. The skills tell the model to take links only from the library, a page it read, or the person.
- **Allowlisted reading.** `fetch_page` (package `internal/web`):
  - reads only an allowlist of educational hosts and their subdomains (Wikipedia, MDN, MIT OCW, Coursera, edX, Khan Academy, arXiv, language docs, freeCodeCamp, OpenStax, Gutenberg, OSSU), and checks every redirect too;
  - uses https only, and its dialer refuses private, loopback, and link-local addresses;
  - honours robots.txt (cached for an hour), sends an identifying User-Agent, and allows 15 pages per run;
  - returns at most 12,000 characters of text with scripts, styles, navigation, and markup removed. The text reaches the model framed as data, like every other tool result.
- **Conversations.** `POST /v1/agent-runs/{id}/continue` starts a new run that carries the earlier messages. Like replies, the transcript lives in API memory for an hour and is never stored or logged. Runs record only `parent_run_id`. Budgets, consent, and step limits apply to each run.
- **Proposals as before.** `draft_roadmap` and `suggest_resources` are write tools: dry-run validated, then applied only when the person approves (ADR 0008).

## Consequences

- The library is a snapshot. Links can rot, so it is reviewed each release (EP-21), and people can edit their own copies.
- The allowlist keeps exfiltration narrow, but not impossible: an allowlisted site with open redirects or user content could still receive a crafted URL. Redirects are re-checked against the allowlist, the run's content stays the person's own, and 15 fetches bound the channel.
- A server restart drops held conversations. The person then starts a new run.
