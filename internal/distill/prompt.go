package distill

import (
	"os"
	"path/filepath"
)

const defaultSystemPrompt = `You are a knowledge-base curator. Given a raw source document, generate a structured wiki source-note page.

Output MUST be valid Markdown with YAML frontmatter. Do NOT wrap your output in code blocks or backticks.

The YAML frontmatter must contain these fields:
  type: source-note
  title: <concise title derived from the document>
  description: <one-sentence summary>
  tags: [<tag1>, <tag2>, ...]
  resource: <original URL or citation if present in the document, else "">
  sources: ["__RAW_SOURCE__"]
  timestamp: <ISO-8601 date, e.g. 2024-01-15>

For the sources field, output the literal placeholder ["__RAW_SOURCE__"] exactly
as shown — the system fills in the real raw-source path. Put any URL or external
citation in the resource field instead.

After the frontmatter, include these Markdown sections in order:

## Source
Brief identification of where this document came from.

## Summary
2–4 paragraph narrative summary of the document's main content.

## Key Facts
Bulleted list of the most important factual claims.
IMPORTANT: Preserve ALL specific terms, names, codes, acronyms, and identifiers that appear in the document
(e.g. SKU, BOM, API names, system names, field names, error codes, product names, data domain labels).
These exact terms are critical for search — do not paraphrase or generalize them away.

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
// as the template section; otherwise the built-in default is returned.
func buildSystemPrompt(kbRoot string) string {
	templatePath := filepath.Join(kbRoot, "schema", "templates", "source-note.md")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return defaultSystemPrompt
	}
	return `You are a knowledge-base curator. Given a raw source document, generate a structured wiki source-note page.

Output MUST be valid Markdown with YAML frontmatter. Do NOT wrap your output in code blocks or backticks.

For the sources field, output the literal placeholder ["__RAW_SOURCE__"] exactly as shown — the system fills in the real raw-source path.

Use the following template as the exact structure for your output:

` + string(data) + `

Begin your response directly with the YAML frontmatter (---).`
}
