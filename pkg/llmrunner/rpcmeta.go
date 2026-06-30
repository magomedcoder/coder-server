package llmrunner

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const MDTraceID = "x-trace-id"
const maxTraceIDLen = 128

type traceIDKeyType struct{}

var traceIDKey = traceIDKeyType{}

func WithTraceID(ctx context.Context, id string) context.Context {
	id = sanitizeTraceID(id)
	if id == "" {
		return ctx
	}

	return context.WithValue(ctx, traceIDKey, id)
}

func TraceIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(traceIDKey).(string)
	return s
}

func IncomingTraceID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	v := md.Get(MDTraceID)
	if len(v) == 0 {
		return ""
	}

	return sanitizeTraceID(v[0])
}

func EnsureTraceInContext(ctx context.Context) context.Context {
	if TraceIDFromContext(ctx) != "" {
		return ctx
	}

	if t := IncomingTraceID(ctx); t != "" {
		return WithTraceID(ctx, t)
	}

	return WithTraceID(ctx, uuid.New().String())
}

func OutgoingContext(ctx context.Context) context.Context {
	tid := TraceIDFromContext(ctx)
	if tid == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, MDTraceID, tid)
}

func sanitizeTraceID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxTraceIDLen {
		s = s[:maxTraceIDLen]
	}

	return s
}
