package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/magomedcoder/gen/pkg/domain"
)

type LLMChat interface {
	SendMessage(
		ctx context.Context,
		messages []*domain.Message,
		stopSequences []string,
		timeoutSeconds int32,
		genParams *domain.GenerationParams,
	) (chan domain.LLMStreamChunk, error)
}

type GenParamsConfig struct {
	Temperature float32
	MaxTokens   int32
}

func RewriteQueryForRetrieval(
	ctx context.Context,
	llm LLMChat,
	cfg GenParamsConfig,
	timeoutSeconds int32,
	sessionID int64,
	userQuery, model string,
) (string, error) {
	msgs := []*domain.Message{
		domain.NewMessage(sessionID, QueryRewriteSystem, domain.MessageRoleSystem),
		domain.NewMessage(sessionID, userQuery, domain.MessageRoleUser),
	}
	gp := &domain.GenerationParams{Temperature: &cfg.Temperature, MaxTokens: &cfg.MaxTokens}
	ch, err := llm.SendMessage(ctx, msgs, nil, timeoutSeconds, gp)
	if err != nil {
		return "", err
	}
	raw, err := DrainStreamContent(ctx, ch)
	if err != nil {
		return "", err
	}
	return SanitizeRewrittenQuery(raw), nil
}

func GenerateHyDEPseudoDocument(
	ctx context.Context,
	llm LLMChat,
	cfg GenParamsConfig,
	timeoutSeconds int32,
	sessionID int64,
	query, model string,
) (string, error) {
	msgs := []*domain.Message{
		domain.NewMessage(sessionID, HyDESystem, domain.MessageRoleSystem),
		domain.NewMessage(sessionID, query, domain.MessageRoleUser),
	}
	gp := &domain.GenerationParams{Temperature: &cfg.Temperature, MaxTokens: &cfg.MaxTokens}
	ch, err := llm.SendMessage(ctx, msgs, nil, timeoutSeconds, gp)
	if err != nil {
		return "", err
	}
	raw, err := DrainStreamContent(ctx, ch)
	if err != nil {
		return "", err
	}
	return SanitizeHyDEPseudoDocument(raw), nil
}

func BuildRerankUserPrompt(userQuery string, pool []domain.ScoredDocumentRAGChunk, passageMaxRunes int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Вопрос:\n%s\n\n", strings.TrimSpace(userQuery))
	for i := range pool {
		excerpt := TrimPassageForRerank(pool[i].DocumentRAGChunk.Text, passageMaxRunes)
		fmt.Fprintf(&b, "--- Фрагмент %d ---\n%s\n\n", i+1, excerpt)
	}
	return b.String()
}

func RerankSearchHits(
	ctx context.Context,
	llm LLMChat,
	cfg GenParamsConfig,
	timeoutSeconds int32,
	sessionID int64,
	userQuery string,
	hits []domain.ScoredDocumentRAGChunk,
	maxCandidates int,
	passageMaxRunes int,
	model string,
) (reordered []domain.ScoredDocumentRAGChunk, elapsedMs int64, err error) {
	t0 := time.Now()
	defer func() { elapsedMs = time.Since(t0).Milliseconds() }()

	if len(hits) < 2 {
		return hits, elapsedMs, nil
	}
	if maxCandidates < 2 {
		maxCandidates = 16
	}

	pool := hits
	var tail []domain.ScoredDocumentRAGChunk
	if len(pool) > maxCandidates {
		pool = hits[:maxCandidates]
		tail = append([]domain.ScoredDocumentRAGChunk(nil), hits[maxCandidates:]...)
	}

	msgs := []*domain.Message{
		domain.NewMessage(sessionID, RerankSystem, domain.MessageRoleSystem),
		domain.NewMessage(sessionID, BuildRerankUserPrompt(userQuery, pool, passageMaxRunes), domain.MessageRoleUser),
	}
	gp := &domain.GenerationParams{Temperature: &cfg.Temperature, MaxTokens: &cfg.MaxTokens}
	ch, err := llm.SendMessage(ctx, msgs, nil, timeoutSeconds, gp)
	if err != nil {
		return hits, elapsedMs, err
	}

	reply, err := DrainStreamContent(ctx, ch)
	if err != nil {
		return hits, elapsedMs, err
	}

	order := ParseRerankOrder(reply, len(pool))
	if len(order) != len(pool) {
		return hits, elapsedMs, nil
	}

	out := make([]domain.ScoredDocumentRAGChunk, 0, len(pool))
	for _, idx := range order {
		if idx < 0 || idx >= len(pool) {
			return hits, elapsedMs, nil
		}
		out = append(out, pool[idx])
	}
	return append(out, tail...), elapsedMs, nil
}

func RunDeepMapCall(
	ctx context.Context,
	llm LLMChat,
	cfg GenParamsConfig,
	timeoutSeconds int32,
	sessionID int64,
	userQuery, excerptBlock, model string,
) (string, error) {
	u := fmt.Sprintf("Вопрос пользователя:\n%s\n\nФрагменты документа:\n%s", strings.TrimSpace(userQuery), strings.TrimSpace(excerptBlock))
	msgs := []*domain.Message{
		domain.NewMessage(sessionID, DeepMapSystem, domain.MessageRoleSystem),
		domain.NewMessage(sessionID, u, domain.MessageRoleUser),
	}
	gp := &domain.GenerationParams{Temperature: &cfg.Temperature, MaxTokens: &cfg.MaxTokens}
	ch, err := llm.SendMessage(ctx, msgs, nil, timeoutSeconds, gp)
	if err != nil {
		return "", err
	}
	raw, err := DrainStreamContent(ctx, ch)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(raw), nil
}

type DeepMapConfig struct {
	ChunksPerMap   int
	MaxMapCalls    int
	MaxOutputRunes int
	Gen            GenParamsConfig
	TimeoutSeconds int32
}

func DeepMapSummaries(
	ctx context.Context,
	llm LLMChat,
	cfg DeepMapConfig,
	sessionID int64,
	userQuery string,
	scored []domain.ScoredDocumentRAGChunk,
	model string,
) (summary string, mapCalls int, elapsedMs int64, err error) {
	if len(scored) == 0 {
		return "", 0, 0, nil
	}
	chunksPer := cfg.ChunksPerMap
	if chunksPer <= 0 {
		chunksPer = 8
	}
	maxCalls := cfg.MaxMapCalls
	if maxCalls <= 0 {
		maxCalls = 4
	}

	t0 := time.Now()
	defer func() { elapsedMs = time.Since(t0).Milliseconds() }()

	var parts []string
	for i := 0; i < len(scored); i += chunksPer {
		if mapCalls >= maxCalls {
			break
		}
		end := min(i+chunksPer, len(scored))
		block := FormatChunksForDeepMap(scored[i:end])
		if strings.TrimSpace(block) == "" {
			mapCalls++
			continue
		}
		part, callErr := RunDeepMapCall(ctx, llm, cfg.Gen, cfg.TimeoutSeconds, sessionID, userQuery, block, model)
		mapCalls++
		if callErr != nil {
			if errors.Is(callErr, context.Canceled) || errors.Is(callErr, context.DeadlineExceeded) {
				return "", mapCalls, elapsedMs, callErr
			}
			continue
		}
		if p := strings.TrimSpace(part); p != "" {
			parts = append(parts, p)
		}
	}

	joined := strings.Join(parts, "\n\n---\n\n")
	if cfg.MaxOutputRunes > 0 {
		joined = TruncateRunes(joined, cfg.MaxOutputRunes)
	}
	return joined, mapCalls, elapsedMs, nil
}
