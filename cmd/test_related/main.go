//go:build fts5

package main

import (
	"fmt"
	"github.com/pieteams/piekbs/internal/kb"
)

func main() {
	kbRoot := "/Users/jasen/.hermes/piekbs-kb"
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Println("=== Full reindex ===")
	n, _ := kb.IndexFilesFull(db, kbRoot)
	fmt.Printf("indexed %d files\n\n", n)

	var total int
	db.QueryRow("SELECT COUNT(*) FROM document_tags").Scan(&total)
	fmt.Printf("document_tags total: %d\n", total)

	fmt.Println("\n=== Noise check (should be 0) ===")
	for _, noise := range []string{"nbsp", "sources", "pattern", "mmbiz", "karpathy"} {
		var cnt int
		db.QueryRow("SELECT COUNT(DISTINCT doc_id) FROM document_tags WHERE tag=?", noise).Scan(&cnt)
		fmt.Printf("  %q: %d docs\n", noise, cnt)
	}

	fmt.Println("\n=== Normalized tags ===")
	for _, t := range []string{"Karpathy", "LLM Wiki", "Obsidian"} {
		var cnt int
		db.QueryRow("SELECT COUNT(DISTINCT doc_id) FROM document_tags WHERE tag=?", t).Scan(&cnt)
		fmt.Printf("  %q: %d docs\n", t, cnt)
	}

	fmt.Println("\n=== hops=1 related for 'Karpathy 知识管理' ===")
	results1, _, _ := kb.SearchLayered(db, kbRoot, "Karpathy 知识管理", nil, nil, 3, 2)
	for _, r := range results1 {
		fmt.Printf("[%s] %s\n", r.Kind, r.Title)
		tagNeighbors := kb.TagExpand(db, []string{r.ID}, 1, 5)
		fmt.Printf("  hop1(%d):", len(tagNeighbors))
		for _, n := range tagNeighbors {
			title := n.Title
			if len(title) > 30 { title = title[:30] }
			fmt.Printf(" | %s", title)
		}
		fmt.Println()
	}

	fmt.Println("\n=== hops=2 related for 'Karpathy 知识管理' ===")
	results2, _, _ := kb.SearchLayered(db, kbRoot, "Karpathy 知识管理", nil, nil, 3, 2)
	for _, r := range results2 {
		fmt.Printf("[%s] %s\n", r.Kind, r.Title)
		tagNeighbors := kb.TagExpand(db, []string{r.ID}, 2, 5)
		fmt.Printf("  hop2(%d):", len(tagNeighbors))
		for _, n := range tagNeighbors {
			title := n.Title
			if len(title) > 30 { title = title[:30] }
			fmt.Printf(" | %s", title)
		}
		fmt.Println()
	}
}
