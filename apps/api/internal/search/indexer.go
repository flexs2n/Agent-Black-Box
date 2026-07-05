package search

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/blackbox-agentdiff/api/internal/model"
	"github.com/blackbox-agentdiff/api/internal/store"
)

type Indexer struct {
	store    store.Store
	embedder Embedder
}

func NewIndexer(st store.Store) *Indexer {
	var embedder Embedder
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		embedder = NewOpenAIEmbedder()
	}
	return &Indexer{
		store:    st,
		embedder: embedder,
	}
}

func (idx *Indexer) Start(ctx context.Context, interval time.Duration) {
	if idx.embedder == nil {
		log.Println("semantic search indexer: no embedder configured (set OPENAI_API_KEY)")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			idx.processBatch(ctx)
		}
	}
}

func (idx *Indexer) processBatch(ctx context.Context) {
	projects, err := idx.store.ProjectList(ctx)
	if err != nil {
		log.Printf("search indexer: project list: %v", err)
		return
	}
	for _, project := range projects {
		if err := idx.indexProject(ctx, project.ID); err != nil {
			log.Printf("search indexer: project %s: %v", project.ID, err)
		}
	}
}

func (idx *Indexer) indexProject(ctx context.Context, projectID string) error {
	traces, err := idx.store.TraceSearch(ctx, projectID, store.TraceSearchFilters{})
	if err != nil {
		return err
	}

	for _, trace := range traces {
		text := BuildTraceSummary(trace)
		if text == "" {
			continue
		}

		if _, err := idx.store.EmbeddingGet(ctx, trace.ID); err == nil {
			continue
		}

		embedding, err := idx.embedder.Embed(ctx, text)
		if err != nil {
			log.Printf("search indexer: embed trace %s: %v", trace.ID, err)
			continue
		}

		if err := idx.store.EmbeddingPut(ctx, trace.ID, trace.ProjectID, embedding); err != nil {
			log.Printf("search indexer: store embedding trace %s: %v", trace.ID, err)
		}
	}
	return nil
}

func BuildTraceSummary(trace model.Trace) string {
	hasContent := trace.Input != nil || trace.Output != nil || trace.Error != nil || trace.AgentName != nil
	if !hasContent {
		return ""
	}
	text := "Trace: "
	if trace.AgentName != nil {
		text += *trace.AgentName + " "
	}
	text += "status=" + trace.Status + ". "
	if trace.Input != nil {
		text += "Input: " + truncate(*trace.Input, 500) + ". "
	}
	if trace.Output != nil {
		text += "Output: " + truncate(*trace.Output, 500) + ". "
	}
	if trace.Error != nil {
		text += "Error: " + truncate(*trace.Error, 500) + ". "
	}
	return text
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
