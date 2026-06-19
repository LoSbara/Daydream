package rag

import (
	"context"
	"fmt"
	"daydream/internal/db"
	"daydream/internal/embedding"
	"sort"
)

// KBEntry è un documento della knowledge base recuperato dal retriever.
type KBEntry struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Category string  `json:"category"`
	Score    float64 `json:"score"`
}

// Retriever esegue hybrid search sulla knowledge_base:
// HNSW (cosine similarity) + BM25 fulltext, risultati mergiati con RRF.
type Retriever struct {
	db    db.DBClient
	embed embedding.Provider
}

func NewRetriever(database db.DBClient, embedder embedding.Provider) *Retriever {
	return &Retriever{db: database, embed: embedder}
}

// Retrieve trova i documenti più rilevanti per la query.
// Ritorna al massimo `limit` risultati ordinati per relevance score.
// Se l'embedding o il DB falliscono, restituisce slice vuota (non errore)
// per non bloccare il turno di gioco.
func (r *Retriever) Retrieve(ctx context.Context, query string, limit int) []KBEntry {
	if query == "" || limit <= 0 {
		return nil
	}

	vecResults := r.vectorSearch(ctx, query, limit)
	ftResults := r.fulltextSearch(ctx, query, limit)

	return rrfMerge(vecResults, ftResults, limit)
}

// vectorSearch esegue ANN search con l'indice HNSW.
func (r *Retriever) vectorSearch(ctx context.Context, query string, limit int) []KBEntry {
	vecs, err := r.embed.Embed(ctx, []string{query})
	if err != nil || len(vecs) == 0 {
		return nil
	}

	// Converti []float32 in []any per SurrealDB
	vec := make([]any, len(vecs[0]))
	for i, v := range vecs[0] {
		vec[i] = v
	}

	qr, err := r.db.QueryOne(
		fmt.Sprintf(`SELECT id, title, content, category,
			vector::similarity::cosine(embedding, $vec) AS score
			FROM knowledge_base
			WHERE embedding <|%d,COSINE|> $vec
			ORDER BY score DESC`, limit),
		map[string]any{"vec": vec},
	)
	if err != nil {
		return nil
	}

	var entries []KBEntry
	_ = qr.All(&entries)
	return entries
}

// fulltextSearch esegue BM25 search sull'indice testuale.
func (r *Retriever) fulltextSearch(ctx context.Context, query string, limit int) []KBEntry {
	ftQR, err := r.db.QueryOne(
		fmt.Sprintf(`SELECT id, title, content, category,
			search::score(0) AS score
			FROM knowledge_base
			WHERE content @0@ $q
			ORDER BY score DESC
			LIMIT %d`, limit),
		map[string]any{"q": query},
	)
	if err != nil {
		return nil
	}

	var entries []KBEntry
	_ = ftQR.All(&entries)
	return entries
}

// rrfMerge implementa Reciprocal Rank Fusion tra due liste di risultati.
// RRF(d) = Σ 1/(k + rank(d)) con k=60 (parametro standard).
func rrfMerge(a, b []KBEntry, limit int) []KBEntry {
	const k = 60.0
	scores := map[string]float64{}
	byID := map[string]KBEntry{}

	for _, list := range [][]KBEntry{a, b} {
		for rank, entry := range list {
			scores[entry.ID] += 1.0 / (k + float64(rank+1))
			if _, seen := byID[entry.ID]; !seen {
				byID[entry.ID] = entry
			}
		}
	}

	merged := make([]KBEntry, 0, len(byID))
	for id, entry := range byID {
		entry.Score = scores[id]
		merged = append(merged, entry)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}
