//go:build fts5

package mcp

import (
	"errors"

	"github.com/jasen215/wikiloop/internal/kb"
)

func handleKBStatus(kbRoot string) map[string]interface{} {
	result, err := kb.KBStatus(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"documents":     result.Documents,
		"by_layer":      result.ByLayer,
		"by_kind":       result.ByKind,
		"index_path":    result.IndexPath,
		"index_size":    result.IndexSize,
		"distill_queue": result.DistillQueue,
	}
}

func handleKBSearch(kbRoot, query string, layer, kind *string, limit int) map[string]interface{} {
	sourceLimit := limit
	if sourceLimit <= 0 {
		sourceLimit = 5
	}
	synthLimit := min(3, sourceLimit/2)
	if synthLimit < 1 {
		synthLimit = 1
	}

	resp, err := kb.KBSearch(kbRoot, query, layer, kind, sourceLimit, synthLimit)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"results":   resp.Results,
		"conflicts": resp.Conflicts,
	}
}

func handleKBPage(kbRoot string, ids []string, full bool) map[string]interface{} {
	pages, err := kb.KBPage(kbRoot, ids, full)
	if err != nil {
		var kbe *kb.KBError
		if errors.As(err, &kbe) {
			return map[string]interface{}{"error": kbe.Message}
		}
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"pages": pages}
}

func handleKBAdd(kbRoot, filename, content, sourceURL string, overwrite bool) map[string]interface{} {
	result, err := kb.KBAdd(kbRoot, filename, content, sourceURL, overwrite)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	m := map[string]interface{}{"path": result.Path, "indexed": result.Indexed}
	if result.IndexError != "" {
		m["index_error"] = result.IndexError
	}
	return m
}

func handleKBReindex(kbRoot string, full bool) map[string]interface{} {
	result, err := kb.KBReindex(kbRoot, full)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"message": result.Message, "written": result.Written}
}

func handleKBLint(kbRoot string) map[string]interface{} {
	result, err := kb.KBLint(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"warnings":     result.Warnings,
		"count":        result.Count,
		"red_links":    result.RedLinks,
		"broken_links": result.BrokenLinks,
		"placeholders": result.Placeholders,
	}
}
