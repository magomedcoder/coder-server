package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type QdrantClient struct {
	baseURL    string
	apiKey     string
	collection string
	http       *http.Client
}

func NewQdrantClient(url, apiKey, collectionPrefix string) *QdrantClient {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if url == "" {
		return nil
	}

	if strings.TrimSpace(collectionPrefix) == "" {
		collectionPrefix = "coder"
	}

	return &QdrantClient{
		baseURL:    url,
		apiKey:     strings.TrimSpace(apiKey),
		collection: collectionPrefix,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (q *QdrantClient) collectionName(workspaceID string) string {
	ws := sanitizeCollectionPart(workspaceID)
	if ws == "" {
		ws = "default"
	}

	return fmt.Sprintf("%s_%s", q.collection, ws)
}

func sanitizeCollectionPart(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func (q *QdrantClient) Upsert(ctx context.Context, workspaceID, pointID string, vector []float32, payload map[string]any) error {
	if q == nil || len(vector) == 0 {
		return nil
	}

	body := map[string]any{
		"points": []map[string]any{{
			"id":      pointID,
			"vector":  vector,
			"payload": payload,
		}},
	}

	return q.do(ctx, http.MethodPut, "/collections/"+q.collectionName(workspaceID)+"/points", body, nil)
}

func (q *QdrantClient) Delete(ctx context.Context, workspaceID string, ids []string) error {
	if q == nil || len(ids) == 0 {
		return nil
	}

	body := map[string]any{"points": ids}
	return q.do(ctx, http.MethodPost, "/collections/"+q.collectionName(workspaceID)+"/points/delete", body, nil)
}

func (q *QdrantClient) Search(ctx context.Context, workspaceID string, vector []float32, limit int) ([]string, []float64, error) {
	if q == nil || len(vector) == 0 {
		return nil, nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	var resp struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}

	body := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}

	if err := q.do(ctx, http.MethodPost, "/collections/"+q.collectionName(workspaceID)+"/points/search", body, &resp); err != nil {
		return nil, nil, err
	}

	ids := make([]string, 0, len(resp.Result))
	scores := make([]float64, 0, len(resp.Result))
	for _, hit := range resp.Result {
		id := fmt.Sprint(hit.ID)
		if id == "" {
			continue
		}

		ids = append(ids, id)
		scores = append(scores, hit.Score)
	}

	return ids, scores, nil
}

func (q *QdrantClient) do(ctx context.Context, method, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, q.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	res, err := q.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("qdrant %s %s: status %d", method, path, res.StatusCode)
	}

	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}

	return nil
}
