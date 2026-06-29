# Schema & Templates

`piekbs init` populates the KB's `schema/` directory with bundled authoring rules and page templates.

## Directory Structure

```text
schema/
  templates/     Markdown templates for each page type
  references/    Authoring rules — page types, citation rules, conflict rules
```

## Page Types

| Type | Location | Description |
|---|---|---|
| source-note | `wiki/source-notes/` | One distilled note per raw document |
| concept | `wiki/concepts/` | Cross-document synthesis of a concept |
| comparison | `wiki/comparisons/` | Side-by-side comparison of approaches |
| decision | `wiki/decisions/` | Technical decision record with rationale |

## Customization

The distill/synthesize prompts read these templates, so editing them customizes the generated wiki format per-KB.

Edit `schema/templates/` to change the structure of distilled pages. Edit `schema/references/` to change authoring rules like citation requirements, conflict handling, and naming conventions.

Changes take effect on the next distillation run.
