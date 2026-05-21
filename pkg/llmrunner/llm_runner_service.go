package llmrunner

import (
	"context"
	"fmt"
	"github.com/magomedcoder/gen-runner/pb/llmrunnerpb"
	"github.com/magomedcoder/gen/pkg/document"
	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/logger"
	"github.com/magomedcoder/gen/pkg/rpcmeta"
	"github.com/magomedcoder/gen/pkg/runnerprompt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"strings"
)

func mapResponseFormatToProto(in *domain.ResponseFormat) *llmrunnerpb.ResponseFormat {
	if in == nil {
		return nil
	}
	out := &llmrunnerpb.ResponseFormat{
		Type: in.Type,
	}
	if in.Schema != nil {
		out.Schema = in.Schema
	}
	return out
}

func mapGenerationParamsToProto(in *domain.GenerationParams) *llmrunnerpb.GenerationParams {
	if in == nil {
		return nil
	}

	out := &llmrunnerpb.GenerationParams{
		ResponseFormat: mapResponseFormatToProto(in.ResponseFormat),
	}

	if in.Temperature != nil {
		out.Temperature = in.Temperature
	}

	if in.MaxTokens != nil {
		out.MaxTokens = in.MaxTokens
	}

	if in.TopK != nil {
		out.TopK = in.TopK
	}

	if in.TopP != nil {
		out.TopP = in.TopP
	}

	if in.EnableThinking != nil {
		out.EnableThinking = in.EnableThinking
	}
	return out
}

type LLMRunnerService struct {
	client llmrunnerpb.LLMRunnerServiceClient
	conn   *grpc.ClientConn
	model  string
}

func NewLLMRunnerService(address, model string) (*LLMRunnerService, error) {
	if address == "" {
		address = "localhost:50052"
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("подключение к gen-runner: %w", err)
	}

	return &LLMRunnerService{
		client: llmrunnerpb.NewLLMRunnerServiceClient(conn),
		conn:   conn,
		model:  model,
	}, nil
}

func (s *LLMRunnerService) Close() error {
	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}

func (s *LLMRunnerService) rpcCtx(ctx context.Context) context.Context {
	return rpcmeta.OutgoingContext(ctx)
}

func (s *LLMRunnerService) CheckConnection(ctx context.Context) (bool, error) {
	pr, err := s.RunnerProbe(ctx)
	if err != nil {
		return false, err
	}

	return pr != nil && pr.GetBackendConnected(), nil
}

func (s *LLMRunnerService) RunnerProbe(ctx context.Context) (*llmrunnerpb.RunnerProbeResponse, error) {
	resp, err := s.client.RunnerProbe(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("gen-runner RunnerProbe: %w", err)
	}

	return resp, nil
}

func (s *LLMRunnerService) GetModels(ctx context.Context) ([]string, error) {
	resp, err := s.client.GetModels(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("gen-runner GetModels: %w", err)
	}

	if resp == nil {
		return nil, nil
	}

	return resp.Models, nil
}

func (s *LLMRunnerService) GetGpuInfo(ctx context.Context) (*llmrunnerpb.GetGpuInfoResponse, error) {
	resp, err := s.client.GetGpuInfo(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("gen-runner GetGpuInfo: %w", err)
	}

	return resp, nil
}

func (s *LLMRunnerService) GetServerInfo(ctx context.Context) (*llmrunnerpb.ServerInfo, error) {
	resp, err := s.client.GetServerInfo(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("gen-runner GetServerInfo: %w", err)
	}

	return resp, nil
}

func (s *LLMRunnerService) GetLoadedModel(ctx context.Context) (*llmrunnerpb.GetLoadedModelResponse, error) {
	resp, err := s.client.GetLoadedModel(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("gen-runner GetLoadedModel: %w", err)
	}

	return resp, nil
}

func (s *LLMRunnerService) LoadModel(ctx context.Context, model string) error {
	model = strings.TrimSpace(model)
	if model == "" || model == "default" {
		model = strings.TrimSpace(s.model)
	}
	if model == "" || model == "default" {
		return fmt.Errorf("укажите model для LoadModel")
	}

	_, err := s.client.LoadModel(s.rpcCtx(ctx), &llmrunnerpb.LoadModelRequest{Model: model})
	if err != nil {
		return fmt.Errorf("gen-runner LoadModel: %w", err)
	}

	return nil
}

func (s *LLMRunnerService) UnloadModel(ctx context.Context) error {
	_, err := s.client.UnloadModel(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return fmt.Errorf("gen-runner UnloadModel: %w", err)
	}

	return nil
}

func (s *LLMRunnerService) ResetMemory(ctx context.Context) error {
	_, err := s.client.ResetMemory(s.rpcCtx(ctx), &llmrunnerpb.Empty{})
	if err != nil {
		return fmt.Errorf("gen-runner ResetMemory: %w", err)
	}

	return nil
}

func (s *LLMRunnerService) SendMessageStream(ctx context.Context, req *llmrunnerpb.SendMessageRequest) (llmrunnerpb.LLMRunnerService_SendMessageClient, error) {
	stream, err := s.client.SendMessage(s.rpcCtx(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("gen-runner SendMessage: %w", err)
	}

	return stream, nil
}

func (s *LLMRunnerService) resolveRunnerModel(model string) string {
	modelName := strings.TrimSpace(model)
	if modelName == "" || modelName == "default" {
		modelName = strings.TrimSpace(s.model)
	}

	if modelName == "default" {
		modelName = ""
	}

	return modelName
}

func (s *LLMRunnerService) Embed(ctx context.Context, text string) ([]float32, error) {
	lm, err := s.GetLoadedModel(ctx)
	if err != nil {
		return nil, err
	}
	if lm == nil || !lm.GetLoaded() {
		return nil, fmt.Errorf("модель не загружена на раннере: вызовите LoadModel перед Embed")
	}

	resp, err := s.client.Embed(s.rpcCtx(ctx), &llmrunnerpb.EmbedRequest{
		Text: text,
	})
	if err != nil {
		return nil, fmt.Errorf("gen-runner Embed: %w", err)
	}

	if resp == nil {
		return nil, nil
	}

	return resp.Values, nil
}

func (s *LLMRunnerService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	lm, err := s.GetLoadedModel(ctx)
	if err != nil {
		return nil, err
	}
	if lm == nil || !lm.GetLoaded() {
		return nil, fmt.Errorf("модель не загружена на раннере: вызовите LoadModel перед EmbedBatch")
	}

	resp, err := s.client.EmbedBatch(s.rpcCtx(ctx), &llmrunnerpb.EmbedBatchRequest{
		Texts: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("gen-runner EmbedBatch: %w", err)
	}

	if resp == nil {
		return nil, nil
	}

	out := make([][]float32, 0, len(resp.Embeddings))
	for _, e := range resp.Embeddings {
		if e == nil {
			out = append(out, nil)
			continue
		}

		out = append(out, e.Values)
	}

	return out, nil
}

func (s *LLMRunnerService) SendMessage(
	ctx context.Context,
	model string,
	messages []*domain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *domain.GenerationParams,
) (chan domain.LLMStreamChunk, error) {
	nTools := 0
	if genParams != nil {
		nTools = len(genParams.Tools)
	}
	logger.I("Runner gRPC client: phase=SendMessage_stream model=%q tools_in_prompt=%d msgs=%d", s.resolveRunnerModel(model), nTools, len(messages))
	return s.sendMessageStream(ctx, model, messages, stopSequences, timeoutSeconds, genParams)
}

func (s *LLMRunnerService) sendMessageStream(
	ctx context.Context,
	model string,
	messages []*domain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *domain.GenerationParams,
) (chan domain.LLMStreamChunk, error) {
	req := &llmrunnerpb.SendMessageRequest{
		Messages:         domainMessagesToProto(messages),
		StopSequences:    stopSequences,
		GenerationParams: mapGenerationParamsToProto(genParams),
	}
	if timeoutSeconds > 0 {
		req.TimeoutSeconds = &timeoutSeconds
	}

	gpForRunner := genParams
	if gpForRunner != nil {
		if prepared, err := runnerprompt.EnsureRenderedPrompt(gpForRunner, messages); err == nil {
			gpForRunner = prepared
		}
	}

	applyRenderedPromptToRequest(req, gpForRunner)

	runnerModel := s.resolveRunnerModel(model)
	lm, lmErr := s.GetLoadedModel(ctx)
	if lmErr != nil {
		return nil, lmErr
	}
	if lm == nil || !lm.GetLoaded() {
		return nil, fmt.Errorf("модель не загружена на раннере: вызовите LoadModel перед SendMessage")
	}
	if nv := countRunnerVisionAttachments(messages); nv > 0 {
		logger.I("Runner gRPC client: phase=vision_attachments model=%q messages_with_image_payload=%d", runnerModel, nv)
	}

	stream, err := s.client.SendMessage(s.rpcCtx(ctx), req)
	if err != nil {
		logger.W("Runner gRPC client: phase=grpc_send_err model=%q err=%v", runnerModel, err)
		return nil, fmt.Errorf("gen-runner SendMessage: %w", err)
	}

	firstMsg, err := stream.Recv()
	if err != nil {
		logger.W("Runner gRPC client: phase=grpc_recv_first_err model=%q err=%v", runnerModel, err)
		return nil, fmt.Errorf("gen-runner SendMessage: ошибка чтения чанка из потока ответа: %w", err)
	}

	output := make(chan domain.LLMStreamChunk, 100)

	go func() {
		defer close(output)
		current := firstMsg
		for {
			content := current.GetContent()
			rc := current.GetReasoningContent()
			if content != "" || rc != "" {
				select {
				case <-ctx.Done():
					return
				case output <- domain.LLMStreamChunk{
					Content:          content,
					ReasoningContent: rc,
				}:
				}
			}

			if current.Done {
				if u := streamUsageFromResponse(current); u != nil {
					select {
					case <-ctx.Done():
						return
					case output <- domain.LLMStreamChunk{Usage: u}:
					}
				}
				return
			}

			msg, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					logger.W("Runner gRPC client: phase=grpc_recv_err model=%q err=%v", runnerModel, err)
				} else {
					logger.V("Runner gRPC client: phase=grpc_stream_eof model=%q", runnerModel)
				}

				return
			}

			current = msg
		}
	}()

	return output, nil
}

func countRunnerVisionAttachments(messages []*domain.Message) int {
	n := 0
	for _, m := range messages {
		if m == nil || len(m.AttachmentContent) == 0 {
			continue
		}

		mt := strings.ToLower(strings.TrimSpace(m.AttachmentMime))
		if document.IsAllowedChatImageMIME(mt) || strings.HasPrefix(mt, "image/") {
			n++
			continue
		}

		if mt == "" && document.IsImageAttachment(m.AttachmentName) {
			n++
		}
	}

	return n
}

func applyRenderedPromptToRequest(req *llmrunnerpb.SendMessageRequest, genParams *domain.GenerationParams) {
	if req == nil || genParams == nil {
		return
	}

	if rp := strings.TrimSpace(genParams.RenderedPrompt); rp != "" {
		req.RenderedPrompt = &rp
	} else {
		req.RenderedPrompt = nil
	}
}

func domainMessagesToProto(messages []*domain.Message) []*llmrunnerpb.ChatMessage {
	prepared := runnerprompt.PrepareMessagesForRunner(messages)
	out := make([]*llmrunnerpb.ChatMessage, 0, len(prepared))
	for _, m := range prepared {
		if m == nil {
			continue
		}

		cm := &llmrunnerpb.ChatMessage{
			Id:        int64(len(out) + 1),
			Content:   m.Content,
			Role:      string(m.Role),
			CreatedAt: m.CreatedAt.Unix(),
		}

		if n := strings.TrimSpace(m.AttachmentName); n != "" {
			cm.AttachmentName = &n
		}
		if len(m.AttachmentContent) > 0 {
			cm.AttachmentContent = m.AttachmentContent
		}

		if mime := strings.TrimSpace(m.AttachmentMime); mime != "" {
			cm.AttachmentMime = &mime
		}

		out = append(out, cm)
	}

	return out
}

func streamUsageFromResponse(msg *llmrunnerpb.ChatResponse) *domain.StreamTokenUsage {
	if msg == nil {
		return nil
	}
	total := msg.GetTotalTokens()
	if total <= 0 {
		return nil
	}
	return &domain.StreamTokenUsage{
		PromptTokens:     msg.GetPromptTokens(),
		CompletionTokens: msg.GetCompletionTokens(),
		TotalTokens:      total,
	}
}
