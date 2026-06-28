# Industry Use Cases

How RAG and LLM Wiki knowledge bases are used across industries — what gets stored, who uses it, and which approach fits best.

> **Quick guide:**
> - **RAG** — best for large, frequently updated document corpora requiring precise citation
> - **LLM Wiki** — best for stable, structured knowledge: concepts, SOPs, terminology, decision records
> - **Combined** — Wiki provides stable structure; RAG handles dynamic retrieval depth

---

## Law Firms

**What goes in the KB:** Case law and precedents, contract template libraries, M&A due diligence files, regulatory updates (SEC, GDPR, etc.), internal SOPs and matter management guides.

| Use Case | Who | How |
|---|---|---|
| Legal research | Associates, paralegals | Natural language query returns cited case references |
| Contract review | Senior lawyers | Upload contract, system flags risk clauses vs. historical templates |
| M&A due diligence | Deal teams | Scan thousands of documents, extract key liabilities and risks |
| Compliance monitoring | Compliance officers | Auto-index regulatory updates, push relevant change alerts |
| Client self-service portal | Clients | Common legal questions answered from internal KB |

**Approach:** RAG-primary (huge document volume, frequent regulation updates, citations required). LLM Wiki for legal concept definitions, internal terminology, standard process guides. Tools: Harvey AI, Thomson Reuters CoCounsel.

---

## Accounting & Finance Firms

**What goes in the KB:** GAAP/IFRS/tax code with interpretive guidance, IRS publications, audit standards (PCAOB/ISA), client financial statements and audit workpapers, SEC/FASB regulatory updates, standard engagement letter templates.

| Use Case | Who | How |
|---|---|---|
| Tax research | Tax managers, accountants | Natural language query returns tax code citations with footnotes |
| Audit support | Audit teams | RAG retrieves audit standards, drafts audit memos |
| Financial statement analysis | Analysts, advisors | Upload 10-K/10-Q, LLM summarizes key risks and anomalies |
| Regulatory change tracking | Compliance | Auto-ingest FASB/PCAOB updates, push impact alerts |
| Client advisory chatbot | Wealth management clients | Portfolio-aware Q&A from personalized KB |

**Approach:** RAG-primary (tax/standards volume is large, versions change). LLM Wiki for core GAAP concepts, internal operation manuals, common Q&A. Tools: Microsoft Copilot for Finance, Workiva AI, Bloomberg Tax AI.

---

## Manufacturing Plants

**What goes in the KB:** Equipment operation manuals, maintenance service bulletins, fault history and repair logs, SOPs, quality inspection checklists, OSHA/ISO safety standards, supplier specs and BOMs.

| Use Case | Who | How |
|---|---|---|
| Equipment troubleshooting | Technicians on the floor | Describe fault in natural language, RAG returns manual steps |
| Quality control | QC staff | Query historical defect patterns, retrieve inspection SOPs |
| Safety compliance | Safety managers | Real-time retrieval of OSHA/ISO clauses for on-site issues |
| New employee onboarding | New workers | Conversational learning of plant procedures |
| Supplier & parts lookup | Procurement | Query lead times, part specs, and substitution options |

**Approach:** RAG-primary for PDF-heavy equipment manuals. LLM Wiki for general plant operating norms, terminology dictionaries, onboarding paths. Combined: Wiki as structured backbone, RAG for specific manual details.

---

## Software Development Teams

**What goes in the KB:** Technical docs, API references, Architecture Decision Records (ADRs), Confluence/Notion pages, Jira ticket history, bug reports and resolutions, code comments and READMEs, key Slack/Teams discussions.

| Use Case | Who | How |
|---|---|---|
| Developer doc Q&A | Engineers | Natural language query for API usage, architecture conventions |
| IT helpdesk automation | IT support | Tier 1/2 tickets auto-resolved from internal KB |
| New engineer onboarding | New hires | Ask about system design and code structure, get instant answers |
| Code review assist | Reviewers | AI retrieves similar historical PRs and decisions |
| Incident response | SRE/DevOps | Quickly retrieve runbooks, historical incident handling records |

**Approach:** LLM Wiki-primary for architecture design, concepts, specs (ideal WikiLoop use case). RAG for unstructured docs (Confluence, Jira, Slack archives). Combined: WikiLoop holds stable knowledge; RAG handles latest Issues/PRs. Tools: GitHub Copilot Enterprise, Glean, Guru.

---

## Customer Service Centers

**What goes in the KB:** Product/service FAQs, refund/return policies, service agreements, historical tickets and resolved cases, agent scripts and escalation SOPs, CRM customer history.

| Use Case | Who | How |
|---|---|---|
| Real-time agent assist | Support reps | RAG surfaces suggested answers during live calls |
| Self-service chatbot | End customers | Common questions answered by AI, reducing ticket volume |
| Ticket classification & routing | System | AI understands query content, routes to correct expert team |
| New agent training | New hires | Simulated Q&A to learn scripts and product knowledge |
| First-contact resolution | Quality team | Analyze knowledge gaps, continuously update KB |

**Approach:** RAG-primary (policies update frequently with promotions and new products). LLM Wiki for stable standard scripts, return policies, escalation flows. Tools: Zendesk AI, Salesforce Einstein, Amazon Q.

---

## Healthcare & Hospitals

**What goes in the KB:** EHR summaries and patient history, clinical guidelines, drug interaction databases, ICD-10/CPT coding standards, hospital SOPs, nursing procedures, medical literature (PubMed).

| Use Case | Who | How |
|---|---|---|
| Clinical decision support | Physicians | Query drug contraindications and treatment guidelines |
| Patient handoff summaries | Nurses, residents | AI auto-summarizes EHR for shift handoffs |
| Medical coding assist | Coders, billing | RAG matches historical cases to correct ICD/CPT codes |
| Patient FAQ bot | Patients | Self-service for appointments, fees, medication instructions |
| Radiology/pathology assist | Specialists | AI retrieves similar historical reports to assist draft |
| Drug information lookup | Pharmacists, nurses | Real-time formulary and drug database retrieval |

**Approach:** RAG-primary (vast medical literature, fast-updating guidelines, evidence citations required). LLM Wiki for internal SOPs, nursing protocols, common care pathways. **Note:** HIPAA compliance requires private/on-premise deployment. Tools: Epic AI, Microsoft Azure Health Bot, Nuance DAX.

---

## Financial Services & Investment Banking

**What goes in the KB:** Regulatory docs (Basel III, MiFID II, Dodd-Frank), research reports, earnings call transcripts, trade history, risk model docs, KYC/AML compliance procedures, internal investment memos.

| Use Case | Who | How |
|---|---|---|
| Investment research synthesis | Research analysts | RAG integrates multi-source reports, generates insight summaries |
| Compliance Q&A | Compliance / risk officers | Query latest regulatory text, assess business compliance |
| Client advisor support | Wealth advisors | Real-time product info and policy retrieval during calls |
| Anti-fraud knowledge base | Risk teams | Cross-team sharing of emerging fraud pattern knowledge |
| M&A due diligence | Investment bankers | Rapidly scan target company financial and legal documents |
| Trader compliance assist | Traders | Pre-trade compliance rule check to avoid violations |

**Approach:** RAG-primary (high document volume, frequent regulatory updates). LLM Wiki for internal product knowledge, standardized compliance workflows. **Note:** Highly sensitive data requires air-gapped or private cloud RAG.

---

## Education & Training

**What goes in the KB:** Course content, teaching materials, enrollment policies, scholarship FAQs, exam syllabi, historical Q&A, faculty research outputs, accreditation compliance docs.

| Use Case | Who | How |
|---|---|---|
| Student Q&A bot | Students | Self-service for course schedules, enrollment rules, scholarships |
| Personalized learning assistant | Students | AI tutoring based on course KB |
| Staff knowledge base | Faculty, admin | Query departmental policies, forms, compliance requirements |
| Corporate training retrieval | Employees | Search training materials, key points, historical case studies |
| Thesis research assist | Graduate students | Semantic search of related literature and existing findings |

**Approach:** Combined — course structure and learning paths as Wiki; specific exercises and research literature via RAG. LLM Wiki-primary for standardized K12 content and fixed corporate training. Tools: Khanmigo, Microsoft Copilot for Education.

---

## Government & Public Agencies

**What goes in the KB:** Laws and regulations, policy documents, government service FAQs (tax, social security, visas), cross-department approval guides, historical policy interpretations, internal compliance regulations.

| Use Case | Who | How |
|---|---|---|
| Citizen service portal | Public | 24×7 self-service for tax, social security, permit procedures |
| Policy document retrieval | Civil servants | Natural language search of policy text to support decisions |
| Cross-department knowledge sharing | All departments | Unified KB breaks information silos |
| Compliance review | Auditors | Retrieve regulations for side-by-side compliance audits |
| Parliamentary Q&A prep | Officials | Quickly retrieve historical documents to prepare responses |

**Approach:** Combined — stable policies as structured Wiki for citizen understanding; full regulatory text indexed via RAG. **Note:** Data sovereignty requires on-premise deployment. Tools: Palantir AI Platform, Microsoft Azure Government.

---

## E-commerce & Retail

**What goes in the KB:** Product details, specifications, usage instructions, return/exchange policies, shipping FAQ, historical tickets, inventory and supplier data, promotion rules and membership benefits.

| Use Case | Who | How |
|---|---|---|
| Smart product Q&A | Customers, agents | Natural language query for specs, stock, shipping time |
| Return/exchange automation | Customers | AI self-service for return requests, reducing manual handling |
| Personalized recommendations | Shoppers | KB combined with user profiles for precise product matching |
| Supply chain Q&A | Procurement, ops | Query inventory, supplier lead times, restocking suggestions |
| Agent assist | Support reps | Real-time policy answers during calls, reducing handle time |

**Approach:** RAG-primary (massive SKU count, frequent product changes). LLM Wiki for stable return policies, membership tier rules, general FAQs. Tools: Salesforce Einstein, Zendesk AI, Shopify Sidekick.

---

## More Industries

| Industry | KB Contents | Key Use Cases | Approach |
|---|---|---|---|
| **Telecom** | Network fault guides, plan comparisons, churn intervention playbooks | Technician troubleshooting, customer plan recommendations, churn intervention scripts | RAG for faults, Wiki for scripts |
| **Insurance** | Policy terms, claims SOPs, underwriting standards, regulatory rules | Claim handler queries, underwriting assist, compliance Q&A | RAG-primary, Wiki for stable processes |
| **Life Sciences / Pharma** | Clinical trial data, FDA/EMA regulatory files, drug development docs | Researcher literature search, regulatory submission drafting | RAG + Wiki combined |
| **Real Estate** | Property listings, market reports, contract templates, local regulations | Agent Q&A, market analysis, contract review | RAG-primary |
| **Media & Publishing** | Archive articles, editorial guidelines, style guides, rights databases | Journalist research, editorial consistency checks, rights clearance | RAG + Wiki combined |

---

## WikiLoop Specifically

WikiLoop's local-first, MCP-native, FTS-based design fits a particular profile in each industry:

| Industry | WikiLoop Fit | Specific Use |
|---|---|---|
| **Law firms** | High | Compile case research notes, decision records, client matter summaries into auditable Wiki |
| **Accounting firms** | High | Structured GAAP concept library, internal engagement methodology Wiki |
| **Manufacturing** | Medium | Plant SOPs, equipment glossary, onboarding knowledge base — stable content only |
| **Software dev** | Very High | ADR library, architecture knowledge base, team onboarding Wiki — WikiLoop's sweet spot |
| **Customer service** | Medium | Standard script Wiki, stable policy reference — combine with RAG for dynamic product data |
| **Healthcare** | High | Internal clinical SOPs, nursing pathway Wiki — private deployment satisfies HIPAA |
| **Financial services** | High | Internal compliance process Wiki, product knowledge base — air-gapped deployment |
| **Education** | High | Course structure Wiki, institutional knowledge base, research synthesis |
| **Government** | High | Policy interpretation Wiki, citizen FAQ base — sovereign deployment requirement met |
| **Research teams** | Very High | Literature synthesis, concept graphs, decision records — core WikiLoop use case |

---

## Selection Guide

| Factor | Choose RAG | Choose LLM Wiki | Choose Both |
|---|---|---|---|
| Document volume | Tens of thousands+ | Hundreds of pages | Mixed |
| Update frequency | Daily / weekly | Monthly / quarterly | Mixed cadence |
| Citation required | Yes (legal, medical) | Generally no | Sensitive domains |
| Content structure | Low (PDF, email) | High (process, concept) | Mixed formats |
| Query type | Open-ended retrieval | Fixed-pattern Q&A | Both types |
