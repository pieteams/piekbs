# WikiLoop in Industry

WikiLoop is a local-first knowledge search engine built for AI agents. Its architecture — a single binary, SQLite FTS5, plain Markdown, MCP protocol — makes it uniquely suited for industries with strict privacy, compliance, and air-gap requirements.

## Why WikiLoop Fits High-Privacy Industries

Most RAG systems require external infrastructure: cloud embedding APIs, vector databases, managed SaaS services. WikiLoop eliminates every one of these dependencies.

| Requirement | How WikiLoop Meets It |
|---|---|
| **No cloud data transmission** | All data stays local. Zero external API calls for search or indexing. |
| **No embedding model required** | Pure SQLite FTS5 + BM25. No vector model, no embedding API, no GPU. |
| **Zero external infrastructure** | Single binary + one SQLite file. No Redis, Kafka, Postgres, or vector DB. |
| **Air-gap compatible** | stdio MCP mode runs as a local subprocess. Works with zero network access. |
| **Auditable knowledge** | All wiki pages are plain Markdown in git. Every change is a diff. |
| **No vendor lock-in** | OKF v0.1 compatible. KB is a directory of files — portable to any system. |
| **Local LLM support** | Distillation works with Ollama or any local model. Never sends documents to OpenAI. |
| **BYOK (Bring Your Own Key)** | If cloud LLM is used for distillation, only the API key is needed — no SaaS subscription. |

---

## Law Firms

**Privacy concern:** Attorney-client privilege. Client documents cannot leave the firm's infrastructure.

**How WikiLoop is used:**

- Compile case research notes, legal memos, and precedent analysis into structured wiki pages
- Build a firm-wide ADR-style decision library: past deal structures, litigation strategies, settlement patterns
- Each matter gets a source-note; concepts (legal theories, regulatory frameworks) synthesized into concept pages
- `wikiloop lint` validates citation integrity — every claim traceable to a source document
- Git history provides full audit trail for privilege review

**Deployment:** Single binary on a firm-managed server or laptop. No internet required after initial setup. LLM distillation can use a local model (Ollama + Llama 3) or an on-premise API.

---

## Accounting & Finance Firms

**Privacy concern:** Client financial data under NDA, regulatory restrictions on data handling (SOX, GDPR).

**How WikiLoop is used:**

- Structured GAAP/IFRS concept library: each standard becomes a wiki page with `related_to` links to relevant interpretations
- Internal engagement methodology Wiki — how the firm approaches specific audit types, documented as decision pages
- Compile client-specific knowledge (anonymized) for cross-engagement learning without exposing raw client data
- Knowledge gap analysis (`wikiloop synthesize --gaps`) surfaces under-documented areas before peak season

**Deployment:** On-premise or private cloud. Distillation with local model keeps client data entirely within firm boundaries.

---

## Healthcare & Hospitals

**Privacy concern:** HIPAA (US), GDPR (EU), local health data laws. Patient data cannot touch public cloud.

**How WikiLoop is used:**

- Internal clinical SOP Wiki: nursing protocols, care pathways, post-operative procedures — structured and searchable
- Drug formulary knowledge base: hospital-specific formulary compiled from standard references, kept up to date by the pharmacy team
- Incident and near-miss knowledge base: de-identified case summaries distilled into source-notes, concept pages for root cause patterns
- New resident onboarding: institution-specific procedures and protocols in a searchable Wiki, reducing reliance on senior staff
- Research team knowledge base: literature synthesis, trial design decisions, IRB-approved protocol documentation

**Deployment:** Air-gapped hospital intranet. `wikiloop stdio` runs as a subprocess of the clinical AI tool. Zero network egress.

---

## Financial Services & Investment Banking

**Privacy concern:** Material non-public information (MNPI), trading confidentiality, regulatory requirements (MiFID II, SEC Rule 10b-5).

**How WikiLoop is used:**

- Internal compliance process Wiki: structured documentation of approved trading practices, escalation procedures, pre-clearance workflows
- Deal knowledge base: for each completed transaction, a source-note captures the deal structure, key decisions, and rationale — builds institutional memory across deal teams
- Regulatory interpretation library: compliance team maintains a Wiki of how the firm interprets specific regulatory rules, with `contradicts` links flagging conflicting interpretations
- Risk model documentation: quantitative analysts document model assumptions, limitations, and validation findings as decision pages

**Deployment:** Air-gapped trading floor infrastructure or private cloud with no public internet path. Distillation uses on-premise LLM.

---

## Manufacturing & Industrial

**Privacy concern:** Trade secrets, proprietary manufacturing processes, IP in equipment design.

**How WikiLoop is used:**

- Plant SOP Wiki: standard operating procedures for each production line, searchable by process step, equipment type, or product family
- Equipment troubleshooting knowledge base: distilled from maintenance logs, service bulletins, and technician field notes — agents query it during repairs
- Quality defect pattern library: historical defect data compiled into source-notes; concept pages synthesize recurring root causes
- New employee onboarding: structured knowledge base reduces dependency on experienced workers for process knowledge transfer
- Supplier and materials knowledge: spec sheets, substitution history, and vendor reliability notes compiled as wiki pages

**Deployment:** Factory floor OT network, often physically isolated from the internet. WikiLoop runs on a local server; technicians query via MCP through their AI coding assistant or custom chat interface.

---

## Government & Defense

**Privacy concern:** National security, classified information, data sovereignty requirements.

**How WikiLoop is used:**

- Policy interpretation Wiki: structured documentation of how specific regulations are applied in practice — searchable by policy area
- Citizen service knowledge base: FAQ and procedure guides for government service delivery, maintained by subject-matter experts
- Cross-agency knowledge sharing: common operational procedures compiled into a shared Wiki, reducing duplication across departments
- Procurement and compliance knowledge: past procurement decisions documented as decision pages, with rationale and alternative options considered

**Deployment:** Sovereign cloud or on-premise with no external connectivity. `wikiloop serve` behind an internal network boundary. MCP over stdio for embedded agent use. OKF-compatible KB format enables knowledge transfer across secure systems.

---

## Energy & Utilities (Nuclear, Oil & Gas)

**Privacy concern:** Critical infrastructure protection, proprietary operational data, physical air-gap requirements.

**How WikiLoop is used:**

- Operations and maintenance knowledge base: equipment manuals, incident reports, and procedure updates distilled into structured wiki pages
- Safety and regulatory compliance Wiki: plant-specific interpretations of NRC/HSE regulations, emergency procedures, safety case documentation
- Knowledge retention for aging workforce: experienced engineers' tacit knowledge captured as source-notes and concept pages before retirement
- Incident learning library: post-incident reports distilled and cross-linked — `related_to` edges surface similar past events during future incident response

**Deployment:** Physically air-gapped operational technology (OT) network. WikiLoop runs with zero external dependencies. Distillation batch-processed offline; index serves agent queries in real time.

---

## Research Institutions & Think Tanks

**Privacy concern:** Pre-publication research confidentiality, grant compliance, institutional IP.

**How WikiLoop is used:**

- Literature synthesis: researchers drop papers into `raw/`, WikiLoop distills source-notes and synthesizes concept and comparison pages across the corpus
- Research decision records: methodology choices, hypothesis revisions, data source evaluations documented as decision pages
- Cross-project knowledge sharing: common methodologies and findings compiled into a shared institutional Wiki
- Knowledge gap analysis: `wikiloop synthesize --gaps` identifies under-explored areas before grant proposal writing

**WikiLoop's specific advantage here:** This is the closest to WikiLoop's original design intent. Agents search the knowledge base iteratively, follow `related` links across papers, and synthesize their own conclusions — exactly how a researcher works.

**Deployment:** Researcher's own machine or institutional server. No cloud required. Local Ollama model for distillation keeps pre-publication data entirely offline.

---

## Private Deployment Architecture

For all industries above, a typical WikiLoop deployment looks like:

```
[Local KB directory]
  raw/           ← drop source documents here
  wiki/          ← distilled pages (Markdown + git)
  index/         ← SQLite FTS index (auto-managed)

[WikiLoop binary]
  wikiloop serve ← HTTP MCP + Web UI + file watcher
  wikiloop stdio ← subprocess MCP for embedded agents

[Agent integration]
  Claude Code / Cursor / any MCP client
  → connects via http://localhost:8766/mcp (HTTP)
  → or spawns wikiloop stdio (no network)

[LLM for distillation]
  Ollama (local)  ← fully offline
  or on-premise API endpoint
```

No data leaves the boundary. No external services required. The entire stack runs on a single machine or internal server.

---

## Comparison with Cloud RAG in Regulated Industries

| Factor | Cloud RAG | WikiLoop (local) |
|---|---|---|
| Data leaves premises | Yes (embedding API, LLM API) | No |
| Vendor lock-in | High (Pinecone, OpenAI, etc.) | None (plain files, SQLite) |
| Compliance audit | Hard (black-box vectors) | Easy (git diff, Markdown) |
| Air-gap capable | No | Yes |
| Infrastructure required | Vector DB, embedding service, LLM API | Single binary |
| Knowledge portability | Low | High (OKF-compatible directory) |
| Setup time | Days to weeks | Minutes |
| Cost at query time | Per-API-call | Zero |
