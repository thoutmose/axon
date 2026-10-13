---
type: roadmap
title: Top-5% in 26 Engineering Domains
created: 2026-10-03 13:21
status: active
source: ai
goal: "[[top5-engineer-2029]]"
hours_per_week: 22-26
budget: 200 EUR/month
start: 2026-10-05
target_end: 2030-02-24
---
## Why / end goal

Top 5% in **all** of these: CS, DSA, OS, data engineering, data analysis,
machine learning, maths, stats, probability, web development, DevOps, DataOps,
Linux, C/C++, Java, Rust, Python, software engineering, Bash, network, API
design, containerization (Docker, Podman, Kubernetes), cloud (Google, AWS,
Azure), cybersecurity, databases, database administration.

14 phases, ~3,700 h at 22 h/week, 2026-10-05 → 2030-02-24. Each phase has a
project that builds on the previous one. Each domain has an external proof (see
the matrix below and [[top5-engineer-2029]]).

## Domain coverage matrix

| # | Domain | Phases | Top-5% proof |
| --- | --- | --- | --- |
| 1 | CS (architecture, compilers, theory, distributed) | 2, 3, 6, 13 | nand2tetris 12 projects · CS:APP labs · 6.5840 all labs · both Crafting Interpreters interpreters · 18.404 problem sets |
| 2 | DSA | 1, 2 + 3–5 problems/week in every phase | ≥ 400 LeetCode problems, contest rating top 5% (shown on profile) |
| 3 | OS | 2, 3 | all MIT 6.1810 xv6 labs pass `make grade` |
| 4 | Data engineering | 1, 5, 6, 7 | streamhouse in production on K8s · ≥ 3 merged PRs in DE OSS (Airflow, dbt, Iceberg, DuckDB, Polars…) |
| 5 | Data analysis | 7, 11 | 3 published analyses (EDA + viz + dashboard + A/B) · Kaggle silver |
| 6 | Machine learning | 11, 14 | Kaggle **silver** in a 1000+ team competition (= top 5%) |
| 7 | Maths | 1, 2, 3, 11, 13 | 6.042J, 18.01, 18.02, 18.06, 18.404 finals ≥ 80% |
| 8 | Stats | 5, 6, 7 | 18.650 final ≥ 80% · Bayesian + experimentation projects |
| 9 | Probability | 3, 5 | Stat 110 final ≥ 80% |
| 10 | Web development | 10 | Full Stack Open, **all** parts, certificate |
| 11 | DevOps | 8, 9 | CKA · own pipelines at DORA "elite" level (on-demand deploys, < 1 h lead time) |
| 12 | DataOps | 7 | contracts + data CI/CD + observability in streamhouse · merged PR in GX or OpenLineage |
| 13 | Linux | 1, 4 | **LFCS** |
| 14 | C/C++ | 2, 5, 13 | CS:APP labs · all BusTub projects (C++) · C++ compiler project, clang-tidy clean |
| 15 | Java | 10 | **OCP Java SE 21 (1Z0-830)** |
| 16 | Rust | 5 | rustkv · merged PR in a Rust data project |
| 17 | Python | 1 | pipekit on PyPI · merged PRs in Python OSS |
| 18 | Software engineering | 1, 10 | design docs + reviewed OSS PRs · *SWE at Google* practices applied |
| 19 | Bash | 1, 4 | OverTheWire Bandit 0–33 · ShellCheck-clean automation library with tests |
| 20 | Network | 3, 4, 12 | **CCNA** |
| 21 | API design | 10 | AIP/Zalando-lint-clean REST + gRPC APIs, public design doc |
| 22 | Containerization | 8, 12 | **CKA + CKS** · rootless Podman build pipeline |
| 23 | Cloud (GCP, AWS, Azure) | 8, 9 | **AWS SAA + GCP ACE + AZ-104** |
| 24 | Cybersecurity | 12 | **BSCP** · all PortSwigger labs · 6.566 labs |
| 25 | Databases | 5, 6 | all CMU 15-445 projects · advanced SQL portfolio |
| 26 | Database administration | 5, 6 | PostgreSQL + MySQL + MongoDB + Redis + Cassandra runbooks with recorded failover/restore drills |

## Constraints

- Time: weekdays 2–3 h, weekends ≥ 7 h/day → planned at **22 h/week effective**
  (rest-token days deducted).
- Budget 200 €/month:
  - ~10 € books:
    [ACM + Skills Bundle ($75/yr) O'Reilly access](https://learning.acm.org/e-learning/oreilly)
    or [O'Reilly direct](https://www.oreilly.com/online-learning/pricing)
  - ~20 € cloud sandbox from phase 4, with billing alerts
  - ~100 € **certification fund** (≈ $3,000 of exams over 3.4 years):
    - [LFCS](https://training.linuxfoundation.org/certification/linux-foundation-certified-sysadmin-lfcs/)
      $445
    - [CCNA](https://www.cisco.com/site/us/en/learn/training-certifications/exams/ccna.html)
      $300
    - [CKA](https://training.linuxfoundation.org/certification/certified-kubernetes-administrator-cka/)
      $445
    - [CKS](https://training.linuxfoundation.org/certification/certified-kubernetes-security-specialist/)
      $445
    - [AWS SAA](https://d1.awsstatic.com/training-and-certification/docs-sa-assoc/AWS-Certified-Solutions-Architect-Associate_Exam-Guide_C03.pdf)
      $150
    - [GCP ACE](https://cloud.google.com/learn/certification/cloud-engineer)
      $125
    - [AZ-104](https://learn.microsoft.com/en-us/credentials/certifications/resources/study-guides/az-104)
      (check the price for your region)
    - Oracle 1Z0-830 $245
    - [BSCP](https://portswigger.net/web-security/certification) $99, plus a
      Burp Pro subscription
    - Watch for Linux Foundation discounts.
  - ~20 € reward fund ([[study-block]])
  - the rest: buffer
- Level: working data engineer (Python, SQL, ELT/ETL, Snowflake, Airflow, dbt
  Core, Docker, Linux, Bash, databases).
- Formats: reading + hands-on projects (`1-Projects/`). EN/FR (French maths
  exercises: [Exo7](http://exo7.emath.fr/)).
- **42 school projects:** the
  [42](https://42.fr/en/42-in-paris/computer-science-degree/) projects are
  folded into phases 1–4, 7, 8 and 10–13 (line `- 42 projects:`, shown by
  `<leader>zgp`). They're done self-taught, part-time, alongside the plan.
  - The real 42 cursus is free, but on-site and full-time: a 4-week,
    7-days-a-week
    [Piscine](https://www.codequoi.com/en/how-to-prepare-for-the-42-piscine/),
    then 3–5 years. That doesn't fit 22 h/week.
  - Projects marked *optional* are only for when you're ahead.
- **42 resources** (subjects and testers; never copy solutions, read others'
  code only after yours passes):
  - Subjects: [ALL_SCHOOL_42](https://github.com/evgenkarlson/ALL_SCHOOL_42)
    (subjects of the main program, School 42 + School 21; last updated 2022, so
    check each subject's version)
  - Finding newer subjects and testers by project name: GitHub topics
    [42](https://github.com/topics/42) and
    [42-cursus](https://github.com/topics/42-cursus). On GitHub/GitLab, also
    search "42 subjects", "42cursus", "school21".
  - Exam practice: [42_examshell](https://github.com/terminal-42s/42_examshell)
    (ranks 02–06)
  - 42-like environment on your machine:
    [42-Devcontainer](https://github.com/opsec-infosec/42-Devcontainer)
  - Python projects checker:
    [Germinette](https://github.com/ExceptedPrism3/germinette)
  - A full Common Core done well, to compare against **after** your own version
    passes:
    [mcombeau/42_cursus_projects](https://github.com/mcombeau/42_cursus_projects)
    (archived)
  - Extra DSA with animations (code in C, C++, Python, Java, Rust…):
    [Hello Algo](https://github.com/krahets/hello-algo)
  - Not covered by 42: Java, Rust and DBA. Those come from the phase resources.

## Weekly rhythm (3 interleaved tracks)

`{M}` `{C}` `{L}` `{R}` come from the current phase's `Tracks` line (`{R}`, the
Sunday reading, falls back to `{L}`) and `{P}` is its project. `<leader>zgp`
shows this table filled in for the phase you're in.

| Day | Block | Content |
| --- | --- | --- |
| Mon | 2–3 h | {M} |
| Tue | 2–3 h | {C} |
| Wed | 2–3 h | {L} |
| Thu | 2–3 h | whichever of {M} / {C} is behind |
| Fri | 2 h | spaced tests (`<leader>zrr`) + 3 DSA problems |
| Sat | 7 h | 4 h project {P} · 2 h DSA · 1 h notes |
| Sun | 7 h | 3 h project {P} · 2 h {R} · 1 h Zettel · 1 h [[weekly-reflection]] |

## Phases

### Phase 1: Python, DSA I, discrete maths, Bash & Linux CLI (2026-10-05 → 2026-12-27)

- Milestone: [[p01-python-dsa-linux]] · Project: [[pipekit]]
- Tracks: M = MIT 6.042J discrete maths · C = MIT 6.006 algorithms · L = Fluent Python + Cosmic Python · R = TLCL + BashGuide + Missing Semester
- 42 projects: ready_set_boole (~25 h, boolean algebra + sets, pairs with
  6.042J). Optional: 42 Python piscine modules, checked with Germinette.
- Domains: Python, DSA, maths, SWE, Bash, Linux, DE
- Resources:
  1. [MIT 6.006 Introduction to Algorithms](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/):
     course, free, ~90 h
  2. [MIT 6.042J Mathematics for CS](https://ocw.mit.edu/courses/6-042j-mathematics-for-computer-science-spring-2015/):
     course, free, ~70 h
  3. *[Fluent Python 2e](https://www.oreilly.com/library/view/-/9781492056348)*:
     O'Reilly, ~45 h. [Cosmic Python](https://cosmicpython.com/): free, ~20 h
  4. [The Linux Command Line](https://linuxcommand.org/tlcl.php) (free, ~25 h) +
     [Missing Semester](https://missing.csail.mit.edu/) (free, ~12 h) +
     [BashGuide](https://mywiki.wooledge.org/BashGuide) &
     [BashPitfalls](https://mywiki.wooledge.org/BashPitfalls) (free, ~10 h),
     linted with [ShellCheck](https://www.shellcheck.net/)
- Project: **pipekit**, a mini Airflow (DAG, topo-sort scheduler, retries, CLI,
  plugins), typed, tested and in CI.
- Exit check: pipekit v0.1 with ≥ 90% coverage. 6.006 problem sets done. 60
  LeetCode. Proof of topo sort. ShellCheck-clean bootstrap script.
  [OverTheWire Bandit](https://overthewire.org/wargames/) levels 0–15.

### Phase 2: Computer architecture, C & systems, DSA II, calculus (2026-12-28 → 2027-03-21)

- Milestone: [[p02-systems-c-calculus]] · Project: [[colstore-c]]
- Tracks: M = 18.01SC → 18.02SC calculus (+ Exo7) · C = CS:APP + labs; Nand to Tetris 1–6 · L = Modern C · R = Erickson Algorithms
- 42 projects: libft (~25 h) → ft_printf (~15 h) → get_next_line (~10 h) → BSQ
  (~15 h, largest square = DP) → push_swap (~30 h, sorting with two stacks).
  colstore-c is built on your own libft + get_next_line. Exam practice:
  examshell rank 02. Optional: computorv1 (~10 h, polynomial solver, with
  calculus).
- Domains: CS, C, OS, DSA, maths
- Resources:
  1. [Nand to Tetris](https://www.nand2tetris.org/): course + book, free, ~80 h
     (projects 1–6: gates → CPU → assembler)
  2. *CS:APP 3e* via O'Reilly + [labs](https://csapp.cs.cmu.edu/3e/labs.html) /
     [student site](https://csapp.cs.cmu.edu/3e/students.html): ~110 h
  3. [Modern C](https://gustedt.gitlabpages.inria.fr/modern-c/): free, ~30 h.
     [Erickson Algorithms](https://jeffe.cs.illinois.edu/teaching/algorithms/):
     free, ~40 h
  4. [18.01SC](https://ocw.mit.edu/courses/18-01sc-single-variable-calculus-fall-2010/)
     →
     [18.02SC](https://ocw.mit.edu/courses/18-02sc-multivariable-calculus-fall-2010/):
     free, ~70 h, + Exo7 exercises
- Project: **colstore-c**, CSV → columnar binary (mmap, dictionary/RLE),
  benchmarked against pandas and DuckDB.
- Exit check: nand2tetris 1–6. CS:APP data, bomb, attack, cache, shell and
  malloc labs. 18.01 final ≥ 80%. 110 LeetCode.

### Phase 3: Operating systems, network theory, linear algebra, probability I (2027-03-22 → 2027-06-13)

- Milestone: [[p03-os-networks-linalg]] · Project: [[tinyhttpd]]
- Tracks: M = 18.06SC linear algebra; Stat 110 first half · C = OSTEP + xv6 labs · L = Kurose & Ross + Beej sockets in C
- 42 projects: pipex (~15 h) + minitalk (~10 h, signals) + philosophers (~25 h,
  threads/mutexes = OSTEP concurrency) + minishell (~70 h,
  fork/exec/pipes/redirections) + matrix (~40 h, linear-algebra library, with
  18.06). tinyhttpd follows the 42 webserv spec (config file, multiple ports,
  CGI), in C. Exam practice: examshell rank 03–04. Optional: ft_malloc (mmap
  allocator), ft_strace (ptrace), nm (ELF), Kernel From Scratch.
- Domains: OS, network, C, maths, probability
- Resources:
  1. [OSTEP](https://pages.cs.wisc.edu/~remzi/OSTEP/): free, ~50 h +
     [MIT 6.1810 xv6 labs](https://pdos.csail.mit.edu/6.1810/): free, ~90 h (all
     labs)
  2. [Kurose & Ross](https://gaia.cs.umass.edu/kurose_ross/index.php): book +
     free Wireshark labs, ~40 h. [Beej's Guide](https://beej.us/guide/bgnet/):
     free, ~10 h
  3. [18.06SC Linear Algebra](https://ocw.mit.edu/courses/18-06sc-linear-algebra-fall-2011/):
     free, ~50 h
  4. [Stat 110](https://stat110.hsites.harvard.edu/) +
     [book](http://probabilitybook.net), first half: free, ~35 h
- Project: **tinyhttpd**, an epoll + thread-pool HTTP/1.1 server in C that
  ingests into colstore; load-tested.
- Exit check: **all** xv6 labs pass. 10k concurrent connections. 18.02 + 18.06
  finals ≥ 80%.

### Phase 4: Linux administration, network operations, Bash automation, probability II (2027-06-14 → 2027-09-05)

- Milestone: [[p04-linux-network-ops]] · Project: [[homelab-infra]]
- Tracks: M = Stat 110 second half · C = ULSAH Linux administration + LFCS prep · L = CCNA (Jeremy's IT Lab + Packet Tracer); Bash library
- 42 projects: Born2beroot (~20 h, hardened VM: LVM, sudo policy, UFW, SSH,
  password policy, cron monitoring script) = homelab node 1 · NetPractice (~10
  h, subnetting/routing) · ft_ping (~20 h, ICMP over raw sockets).
- Domains: Linux, Bash, network, probability
- Resources:
  1. *[UNIX and Linux System Administration Handbook 5e](https://www.pearson.com/en-us/subject-catalog/p/Nemeth-UNIX-and-Linux-System-Administration-Handbook-5th-Edition/P200000000513)*:
     via O'Reilly, ~60 h (systemd, storage/LVM/RAID, users/PAM, packages,
     logging, kernel, DNS/DHCP, firewalls)
  2. [Jeremy's IT Lab free CCNA course](https://github.com/dadavidson/CCNA-200-301-Course-Jeremys-IT-Lab)
     (YouTube, with Packet Tracer labs): free, ~80 h.
     [CCNA exam topics](https://learningnetwork.cisco.com/s/ccna-exam-topics)
  3. [LFCS exam domains](https://training.linuxfoundation.org/certification/linux-foundation-certified-sysadmin-lfcs/):
     hands-on prep, ~30 h
  4. Stat 110, second half: ~35 h
- Project: **homelab-infra**, 3+ Linux VMs on a routed/VLAN-segmented virtual
  network:
  - DNS, DHCP, firewall, WireGuard VPN, NFS and LVM
  - monitoring and log shipping
  - everything provisioned by a tested, ShellCheck-clean Bash library (bats
    tests)
- Exit check: **LFCS passed**, **CCNA passed**, Bandit 0–33, Stat 110 final ≥
  80%.

### Phase 5: Database internals, PostgreSQL DBA, Rust, statistics (2027-09-06 → 2027-11-28)

- Milestone: [[p05-databases-rust-stats]] · Project: [[rustkv]]
- Tracks: M = MIT 18.650 statistics · C = CMU 15-445 + BusTub; DDIA · L = Rust Book + Rustlings; PostgreSQL 14 Internals
- Domains: databases, DBA, Rust, C++, stats
- Resources:
  1. [CMU 15-445/645](https://15445.courses.cs.cmu.edu/fall2025/) + BusTub
     (C++): free, ~120 h (all projects)
  2. *DDIA 2e* ([Kleppmann](https://martin.kleppmann.com/)): O'Reilly, ~40 h.
     [PostgreSQL 14 Internals](https://edu.postgrespro.com/postgresql_internals-14_parts1-4_en.pdf):
     free, ~30 h
  3. [Rust Book](https://doc.rust-lang.org/stable/book/) +
     [Rustlings](https://github.com/rust-lang/rustlings): free, ~45 h
  4. [MIT 18.650 Statistics for Applications](https://ocw.mit.edu/courses/18-650-statistics-for-applications-fall-2016/):
     free, ~45 h
- Project: **rustkv**, an LSM KV store in Rust (WAL, SSTables, bloom filters,
  compaction, crash recovery). Plus a PostgreSQL lab: replication, failover,
  PITR, tuning.
- Exit check: all BusTub projects pass the local tests. rustkv recovery test
  passes. PG failover + PITR drill recorded. 18.650 final ≥ 80%.

### Phase 6: Advanced SQL, polyglot DBA (MySQL/NoSQL), distributed systems, Bayesian stats (2027-11-29 → 2028-02-20)

- Milestone: [[p06-sql-nosql-distributed]] · Project: [[raft-polystore]]
- Tracks: M = Think Bayes 2 · C = MIT 6.5840 distributed systems (Go) · L = Use The Index, Luke + MySQL/MongoDB/Redis/Cassandra DBA
- Domains: databases, DBA, CS (distributed), DE, stats
- Resources:
  1. [MIT 6.5840 Distributed Systems](https://pdos.csail.mit.edu/6.824/): free,
     ~110 h (MapReduce, Raft, KV, sharding labs in Go)
  2. [Use The Index, Luke](https://use-the-index-luke.com/): free, ~15 h (+
     window functions, recursive CTEs, query plans across engines)
  3. DBA docs: [MySQL 8.4 manual](https://dev.mysql.com/doc/refman/8.4/en/),
     [MongoDB University](https://learn.mongodb.com/),
     [Redis University](https://redis.io/tutorials/university/),
     [Cassandra docs](https://cassandra.apache.org/doc/latest/): free, ~50 h
  4. [Think Bayes 2](https://allendowney.github.io/ThinkBayes2/): free, ~25 h
- Project: **raft-polystore**: all 6.5840 labs (sharded fault-tolerant KV). Plus
  a DBA lab for MySQL (replication, backup), MongoDB (replica set), Redis
  (sentinel/cluster) and Cassandra (3 nodes, repair). Each has a runbook and a
  recorded failure drill.
- Exit check: all 6.5840 labs pass repeated test runs. 5 DBA runbooks with
  drills. 30 hard SQL problems. Bayesian A/B notebook.

### Phase 7: Advanced data engineering, DataOps, data analysis & experimentation (2028-02-21 → 2028-05-14)

- Milestone: [[p07-data-platform-dataops]] · Project: [[streamhouse]]
- Tracks: M = Think Stats + Kohavi experimentation · C = Fundamentals of DE + Kafka/Spark/Iceberg docs · L = Python for Data Analysis + Wilke dataviz; GX + OpenLineage
- 42 projects: Data Science Piscine (~40 h, PostgreSQL in docker compose, ETL,
  data warehouse, dedup/fusion, analysis + viz) as the warm-up before
  streamhouse.
- Domains: DE, DataOps, data analysis, stats, Python
- Resources:
  1. *[Fundamentals of Data Engineering](https://books.google.com/books/about/Fundamentals_of_Data_Engineering.html?id=Z_TFzgEACAAJ)*:
     O'Reilly, ~25 h
  2. Docs: [Kafka](https://kafka.apache.org/documentation/),
     [Spark Structured Streaming](https://spark.apache.org/docs/latest/streaming/index.html),
     [Iceberg spec](https://iceberg.apache.org/spec/),
     [OpenLineage](https://openlineage.io/docs/),
     [GX Core](https://docs.greatexpectations.io/docs/core/introduction/gx_overview/):
     ~60 h
  3. [Python for Data Analysis 3e](https://wesmckinney.com/book/) +
     [Think Stats 3e](https://allendowney.github.io/ThinkStats/) +
     [Fundamentals of Data Visualization](https://clauswilke.com/dataviz): free,
     ~60 h
  4. *Trustworthy Online Controlled Experiments* (Kohavi, Tang, Xu): book, paid,
     ~20 h
- Project: **streamhouse**:
  - pipeline: Kafka → Spark → Iceberg → dbt → DuckDB/Trino
  - DataOps layer: data contracts, GX, OpenLineage, data CI/CD, freshness/volume
    SLOs
  - BI dashboard
  - 3 published analyses (EDA, viz, A/B with power analysis)
- Exit check: `make up` works. A broken contract fails CI. 3 analyses published.
  First OSS PR opened in a DE/DataOps project.

### Phase 8: Containers, Kubernetes, DevOps, AWS (2028-05-15 → 2028-08-06)

- Milestone: [[p08-k8s-devops-aws]] · Project: [[streamhouse-on-k8s]]
- Tracks: M = SRE books (SLOs, error budgets) · C = Kubernetes docs + Kubernetes The Hard Way · L = Terraform + Docker/Podman; AWS SAA prep
- 42 projects: Inception (~30 h, NGINX + TLS, WordPress, MariaDB in hand-written
  Dockerfiles + docker compose) as the first week of the phase, before
  Kubernetes.
- Domains: containerization, DevOps, cloud (AWS)
- Resources:
  1. [Kubernetes docs](https://kubernetes.io/docs/home/) +
     [Kubernetes The Hard Way](https://github.com/kelseyhightower/kubernetes-the-hard-way):
     ~70 h
  2. [Docker docs](https://docs.docker.com/get-started/) +
     [Podman tutorials](https://docs.podman.io/en/latest/Tutorials.html): ~15 h
  3. [Terraform tutorials](https://developer.hashicorp.com/terraform/tutorials) +
     [Google SRE books](https://sre.google/books/) +
     *[DevOps Handbook 2e](https://itrevolution.com/product/the-devops-handbook-second-edition/)*:
     ~55 h
  4. AWS SAA prep: ~40 h
- Project: **streamhouse-on-k8s**: k3s → EKS via Terraform, GitOps, rootless
  Podman builds, Prometheus/Grafana, SLOs, DORA metrics.
- Exit check: **CKA + AWS SAA passed**. Cluster rebuilt in under 30 min.
  Game-day post-mortem.

### Phase 9: Google Cloud, Azure, multi-cloud & platform engineering (2028-08-07 → 2028-10-29)

- Milestone: [[p09-gcp-azure-multicloud]] · Project: [[multicloud-streamhouse]]
- Tracks: M = DevOps Handbook + cloud cost modeling · C = GCP: Google Skills + ACE guide · L = Azure: AZ-104 learning paths; Terraform GCP/Azure
- Domains: cloud (GCP, Azure), DevOps, DE
- Resources:
  1. [Google Skills (ex-Cloud Skills Boost)](https://www.skills.google/) +
     [ACE exam guide](https://cloud.google.com/learn/certification/cloud-engineer):
     ~70 h
  2. Microsoft Learn
     [AZ-104 study guide](https://learn.microsoft.com/en-us/credentials/certifications/resources/study-guides/az-104) +
     learning paths: ~70 h
  3. Terraform GCP/Azure tracks: ~20 h
- Project: **multicloud-streamhouse**: one Terraform codebase deploying
  streamhouse to GKE (BigQuery/GCS) and AKS (ADLS/Synapse-equivalent) as well as
  EKS. Includes a cost comparison and an IAM/networking comparison write-up.
- Exit check: **GCP ACE + AZ-104 passed**. The same pipeline green on 3 clouds.

### Phase 10: Software engineering, API design, web development, Java (2028-10-30 → 2029-01-21)

- Milestone: [[p10-java-api-web]] · Project: [[datacatalog]]
- Tracks: M = SWE at Google + A Philosophy of Software Design · C = Google AIPs + Zalando API guidelines · L = Java: dev.java + Effective Java + OCP prep · R = Full Stack Open
- 42 projects: ft_transcendence (optional, ~150 h with a team: real-time
  multiplayer web app, auth + 2FA, WebSockets). Do it only if you're ahead of
  schedule.
- Domains: Java, API design, web, SWE
- Resources:
  1. [dev.java](https://dev.java/learn/) +
     *[Effective Java 3e](https://www.oreilly.com/library/view/effective-java-3rd/9780134686097/)*:
     ~55 h. OCP 1Z0-830 prep: ~30 h
  2. [Google AIPs](https://google.aip.dev/1) +
     [Zalando API Guidelines](https://opensource.zalando.com/restful-api-guidelines/):
     ~15 h
  3. [Full Stack Open](https://fullstackopen.com/en/) (**all parts**: React,
     Node, TypeScript, GraphQL, containers, CI/CD): ~150 h.
     [MDN](https://developer.mozilla.org/en-US/docs/Learn_web_development)
  4. [Software Engineering at Google](https://abseil.io/resources/swe-book): ~25
     h. *A Philosophy of Software Design*: ~8 h
- Project: **datacatalog**: Spring Boot REST + gRPC (AIP-style), React/TS front
  end, auth, OpenLineage ingestion, on the clusters.
- Exit check: **OCP Java SE 21 passed**. Full Stack Open certificate (all
  parts). Lint-clean API spec + design doc.

### Phase 11: Machine learning & MLOps, maths for ML (2029-01-22 → 2029-04-15)

- Milestone: [[p11-ml-mlops]] · Project: [[ml-on-streamhouse]]
- Tracks: M = MML book + CS229 notes · C = ISLP + Zero to Hero · L = Hands-On ML (PyTorch) + Designing ML Systems; fast.ai
- 42 projects: ft_linear_regression (~10 h, gradient descent) → DSLR (~30 h,
  logistic regression + statistics + viz) → multilayer-perceptron (~30 h,
  numpy-only backprop), before using libraries.
- Domains: ML, maths, stats, data analysis
- Resources:
  1. [ISLP](https://www.statlearning.com/): ~50 h.
     [MML book](https://mml-book.github.io/): ~30 h.
     [CS229 notes](https://cs229.stanford.edu/): ~25 h
  2. [Zero to Hero](https://karpathy.ai/zero-to-hero.html): ~35 h.
     [fast.ai](https://course.fast.ai/): ~30 h
  3. *[Hands-On ML (Scikit-Learn + PyTorch)](https://www.oreilly.com/library/view/hands-on-machine-learning/9798341607972/colophon01.html)* +
     *Designing ML Systems* ([repo](https://github.com/chiphuyen/dmls-book)):
     ~60 h
- Project: **ml-on-streamhouse**: feature pipeline, experiment tracking, serving
  via datacatalog, drift monitoring. Kaggle competition.
- Exit check: micrograd + GPT from scratch. Model in production. Kaggle
  bronze/top 10%.

### Phase 12: Cybersecurity, network security, container security (2029-04-16 → 2029-07-08)

- Milestone: [[p12-security]] · Project: [[secure-streamhouse]]
- Tracks: M = threat modeling + crypto (6.566 lectures) · C = MIT 6.566 labs · L = PortSwigger Academy → BSCP; CKS prep; OverTheWire
- 42 projects: snow-crash (~30 h, 14-level CTF) → darkly (~25 h, web
  vulnerabilities) → rainfall (~40 h, binary exploitation/reverse engineering).
  Optional: ft_ssl (hash functions from scratch).
- Domains: cybersecurity, network, containerization, Linux
- Resources:
  1. [MIT 6.566 Computer Systems Security](https://css.csail.mit.edu/6.5660/2024/):
     all 5 labs, ~70 h
  2. [PortSwigger Web Security Academy](https://portswigger.net/web-security)
     (all labs) → [BSCP](https://portswigger.net/web-security/certification):
     ~90 h
  3. [OverTheWire](https://overthewire.org/wargames/) Natas/Leviathan/Narnia:
     ~25 h
  4. [CKS](https://training.linuxfoundation.org/certification/certified-kubernetes-security-specialist/)
     prep: ~35 h
- Project: **secure-streamhouse**: threat model, pentest, hardening (SAST/DAST,
  network policies, secrets, IAM least privilege) on all 3 clouds.
- Exit check: **BSCP + CKS passed**. All PortSwigger labs. Pentest report with
  all high findings fixed.

### Phase 13: C++, compilers, theory of computation (2029-07-09 → 2029-09-30)

- Milestone: [[p13-cpp-compilers-theory]] · Project: [[sqlc-cpp]]
- Tracks: M = MIT 18.404J theory of computation · C = Crafting Interpreters; Nand to Tetris 7–12 · L = LearnCpp + A Tour of C++ + C++ Core Guidelines
- 42 projects: CPP modules 00–09 (~80 h, classes, polymorphism, templates, STL),
  before sqlc-cpp. Optional: ft_containers (STL containers from scratch), ft_irc
  (IRC server in C++98).
- Domains: C/C++, CS, maths, databases
- Resources:
  1. [LearnCpp](https://www.learncpp.com/) +
     *[A Tour of C++ 3e](https://www.oreilly.com/library/view/a-tour-of/9780136823575/)* +
     [C++ Core Guidelines](https://isocpp.github.io/CppCoreGuidelines/CppCoreGuidelines):
     ~70 h
  2. [Crafting Interpreters](https://www.craftinginterpreters.com/): both
     interpreters, ~70 h. [Nand to Tetris](https://www.nand2tetris.org/)
     projects 7–12 (VM, compiler, OS): ~50 h
  3. [MIT 18.404J Theory of Computation](https://ocw.mit.edu/courses/18-404j-theory-of-computation-fall-2020/):
     ~60 h
- Project: **sqlc-cpp**: a SQL-subset compiler in modern C++. Lexer, parser,
  planner and bytecode VM that runs on colstore files. Compared with DuckDB.
- Exit check: clox + jlox done. nand2tetris 7–12. 18.404 problem sets ≥ 80%.
  sqlc-cpp clang-tidy/ASan clean.

### Phase 14: Mastery & proof (2029-10-01 → 2030-02-24, 21 wk)

- Milestone: [[p14-mastery-proof]] · Project: [[oss-and-portfolio]]
- Tracks: M = weakest maths/theory domain from monthly tests · C = LeetCode contests + Kaggle · L = OSS PRs + portfolio write-ups
- Domains: all, with weakest first (from monthly/Feynman tests)
- Work: LeetCode contests weekly. Kaggle silver push. ≥ 3 merged OSS PRs (DE,
  Rust/Python, DataOps). Portfolio site with 14 project write-ups. Retake any
  failed exam.
- Exit check: every row of the domain matrix proven in [[top5-engineer-2029]].

## Habits

- [[study-block]] (daily, streak + rest tokens + rewards)
- [[daily-review]] (daily, spaced tests + board)
- [[weekly-project-session]] (weekly, 4 h project)
- [[weekly-reflection]] (weekly + monthly roadmap check)

## Learning method

- Resource → literature notes → permanent notes → spaced tests (5W 1 h · blank 1
  d · cards 3 d · cloze 1 wk · monthly · Feynman 3 mo), on the board at
  `<leader>zrr`.
- Interleave M/C/L tracks weekly. Monthly tests mix the current phase with
  earlier ones.
- One concrete project always running; each one extends the previous.
- Dual coding: a mermaid diagram per big idea, at least one per phase:
  - CPU datapath (P2)
  - page tables (P3)
  - VLAN/routing topology (P4)
  - LSM compaction (P5)
  - Raft (P6)
  - lakehouse flow (P7)
  - K8s control loop (P8)
  - multi-cloud network (P9)
  - API resource model (P10)
  - backprop (P11)
  - threat-model DFD (P12)
  - compiler pipeline (P13)
- Monthly roadmap check in the monthly note: hours vs plan, matrix progress,
  resize the next phase.

## Progress log

- 2026-10-03 roadmap created
- 2026-10-03 expanded to 14 phases so all 26 domains have their own track and
  proof
- 2026-10-03 42 common-core projects added to phases 2, 3, 4, 8, 10, 13
- 2026-10-03 more 42 projects (maths, ML, data science, security, networking) +
  42 resources added
- 2026-10-03 per-phase Tracks added for the current-phase view (`<leader>zgp`);
  start confirmed Monday 2026-10-05
