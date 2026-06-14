package service

import (
	"github.com/magomedcoder/coder-server/pkg/idempotency"
	"github.com/magomedcoder/coder-server/pkg/quota"
	"github.com/magomedcoder/coder-server/pkg/requestqueue"
	"github.com/magomedcoder/coder-server/pkg/ssestream"
)

type RequestQueue = requestqueue.Queue
type TokenQuota = quota.Quota
type IdempotencyStore = idempotency.Store
type StreamRegistry = ssestream.Registry
type StreamSession = ssestream.Session
type SSEEvent = ssestream.Event

var (
	NewRequestQueue     = requestqueue.New
	NewTokenQuota       = quota.New
	NewIdempotencyStore = idempotency.New
	ErrQueueTimeout     = requestqueue.ErrQueueTimeout
)
