package distill

import (
	"os"
	"path/filepath"
)

const defaultSystemPrompt = `You are a knowledge-base curator. Given a raw source document, generate a structured wiki source-note page.

Output MUST be valid Markdown with YAML frontmatter. Do NOT wrap your output in code blocks or backticks.

LANGUAGE RULE: Write all content in the SAME language as the source document. Chinese source → Chinese output.

ALIAS RULE (MANDATORY): In every Key Facts bullet, inline ALL known aliases, abbreviations, and cross-language equivalents.
  BAD:  "召回率偏低"  GOOD: "Context Recall（CR，召回率，检索覆盖率）偏低（0.32）"

ENTITY RULE (MANDATORY): Mark named entities using 【entity|type】. Types: 人物|组织|产品|技术|概念|项目|地点
  GOOD: "【Karpathy|人物】提出【LLM Wiki|概念】三层架构"

The YAML frontmatter must contain these fields:
  type: source-note
  title: <concise title derived from the document>
  description: <1-2 sentences in the source's primary language, including Chinese keywords if the source is Chinese, plus ≥2 specific technical terms or numbers>
  tags: [<3-6 domain classification tags. MUST be specific technical/domain terms.
       GOOD examples: "RAG", "主数据", "向量数据库", "数据治理", "GraphRAG", "Embedding"
       BAD examples: "AI", "技术", "文章", "知识", "方法" — too generic, do NOT use these.
       Minimum 3 tags, all domain-specific.>]
  doc_type: <one of: 技术文章 | 白皮书 | 技术规范 | 项目文档 | 会议纪要 | 分析报告 | 教程 | 开源项目 | 产品文档>
  authority: <integer 1-5. 5=official doc/paper/project author; 4=reputable org/benchmark; 3=data-backed analysis; 2=secondhand summary; 1=opinion/no data>
  authority: <integer 1-5. 5=official doc/paper/author-written; 4=reputable org tech blog/benchmark; 3=data-backed analysis; 2=secondhand summary/interpretation; 1=opinion/marketing/no data>
  resource: <original URL or citation if present in the document, else "">
  sources: ["__RAW_SOURCE__"]
  timestamp: <ISO-8601 date MUST be extracted from the document itself (publication date, article date, report date).
             Look for dates in: article header, byline, URL path, "发布于", "Posted", copyright year.
             Format: "YYYY-MM-DDTHH:MM:SSZ". If only year is found: "YYYY-01-01T00:00:00Z".
             NEVER use today's date as a default. If truly no date exists anywhere, use "".>

For the sources field, output the literal placeholder ["__RAW_SOURCE__"] exactly
as shown — the system fills in the real raw-source path. Put any URL or external
citation in the resource field instead.

After the frontmatter, include these Markdown sections in order:

## Source
Brief identification of where this document came from.

## Summary
2–4 paragraph narrative summary of the document's main content.

## Key Facts
Bulleted list of the most important factual claims (aim for 5-8 bullets).
IMPORTANT: Preserve ALL specific terms, names, codes, acronyms, and identifiers that appear in the document
(e.g. SKU, BOM, API names, system names, field names, error codes, product names, data domain labels).
These exact terms are critical for search — do not paraphrase or generalize them away.
Each bullet must contain at least one specific number, metric, or named entity.
ALIAS RULE: Inline ALL known aliases, abbreviations, and cross-language equivalents directly in each claim.
  BAD:  "CR 偏低需要优化"
  GOOD: "Context Recall（CR，召回率，检索覆盖率）偏低（0.32），需通过扩大 wiki 覆盖度优化"

CRITICAL RULE FOR STRUCTURED DOCUMENTS: If the source contains numbered or coded items
(e.g. M01-M43, API endpoints, field catalogs, table lists, equipment codes), you MUST
preserve EVERY item completely — including its ID/code, name, source system, and all
storage table names (Hive/Iceberg/MatrixDB paths, database.schema.table identifiers).
Do NOT summarize, merge, or omit any entry. Partial preservation is a critical failure.

## Quotes
Notable direct quotes from the document (if any). If none, write "None."

## Terms
Short definitions of domain-specific terms introduced in the document.
List every significant abbreviation, acronym, or technical term that appears in the source.

## Limitations
Known caveats, gaps, biases, or expiration concerns about this source.

## Related Pages
Links to related wiki pages (use [[Page Title]] syntax). If unknown, write "None."

Begin your response directly with the YAML frontmatter (---).`

// buildSystemPrompt returns the system prompt for source-note distillation.
// If schema/templates/source-note.md exists in kbRoot, its content is used
// as the output template. Additionally, any reference files found in
// schema/references/ (page-types.md, citation-rules.md) are appended as
// quality constraints. Falls back to the built-in default if no template exists.
func buildSystemPrompt(kbRoot string) string {
	templatePath := filepath.Join(kbRoot, "schema", "templates", "source-note.md")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return defaultSystemPrompt
	}

	prompt := `You are a knowledge-base curator. Given a raw source document, generate a structured wiki source-note page.

Output MUST be valid Markdown with YAML frontmatter. Do NOT wrap your output in code blocks or backticks.

LANGUAGE RULE: Write all content (summary, key_claims, terms, etc.) in the SAME language as the source document.
If the source is Chinese, output Chinese. If English, output English. Do NOT switch languages.

ALIAS RULE (MANDATORY): In every key_claim, inline ALL known aliases, abbreviations, and cross-language equivalents.
  BAD:  "召回率偏低需要优化"
  GOOD: "Context Recall（CR，召回率，检索覆盖率）偏低（0.32），需通过扩大 wiki 覆盖度优化"
  BAD:  "FTS检索性能较好"
  GOOD: "FTS（Full-Text Search，全文检索，BM25算法）检索性能优于向量搜索（Vector Search）在精确术语匹配场景"
This is critical for search: users may query with any variant of a term.

ENTITY RULE (MANDATORY): Mark named entities inline using 【entity|type】 format. Types: 人物|组织|产品|技术|概念|项目|地点
  GOOD: "【Karpathy|人物】提出的【LLM Wiki|概念】采用三层架构，由【Anthropic|组织】等团队验证"
  GOOD: "【bge-small-zh|产品】（【BAAI|组织】出品）在【WikiLoop|项目】中用于向量嵌入，维度512"
This enables cross-document entity linking and multi-hop retrieval.

For the sources field, output the literal placeholder ["__RAW_SOURCE__"] exactly as shown — the system fills in the real raw-source path.

Use the following template as the exact structure for your output:

` + string(data) + `

Begin your response directly with the YAML frontmatter (---).`

	// Append reference files as quality constraints if they exist.
	for _, ref := range []string{"page-types.md", "citation-rules.md"} {
		refPath := filepath.Join(kbRoot, "schema", "references", ref)
		refData, err := os.ReadFile(refPath)
		if err != nil {
			continue
		}
		prompt += "\n\n---\n# Knowledge Base Rules: " + ref + "\n\n" + string(refData)
	}

	return prompt
}
