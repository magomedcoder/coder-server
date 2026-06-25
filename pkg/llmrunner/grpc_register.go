package llmrunner

import (
	"github.com/magomedcoder/lm-runner/pb/llmrunnerpb"
	"google.golang.org/grpc"
)

type UnimplementedGRPCServer = llmrunnerpb.UnimplementedLLMRunnerServiceServer

func RegisterGRPCServer(s grpc.ServiceRegistrar, srv llmrunnerpb.LLMRunnerServiceServer) {
	llmrunnerpb.RegisterLLMRunnerServiceServer(s, srv)
}
