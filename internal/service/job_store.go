package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/magomedcoder/coder-server/internal/domain"
)

const (
	jobStatusPending   = "pending"
	jobStatusRunning   = "running"
	jobStatusCompleted = "completed"
	jobStatusFailed    = "failed"
)

type QueueJob struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	RequestID string          `json:"request_id"`
	Status    string          `json:"status"`
	Body      json.RawMessage `json:"body"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type JobStore struct {
	mu        sync.Mutex
	dir       string
	maxJobs   int
	completed atomic.Int64
}

func NewJobStore(dir string, maxJobs int) (*JobStore, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}

	if maxJobs <= 0 {
		maxJobs = 256
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	return &JobStore{
		dir:     dir,
		maxJobs: maxJobs,
	}, nil
}

func (s *JobStore) Enqueue(kind, requestID string, body any) (string, error) {
	if s == nil {
		return "", errors.New("persistent queue отключена")
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.countLocked(jobStatusPending)+s.countLocked(jobStatusRunning) >= s.maxJobs {
		return "", errors.New("persistent queue переполнена")
	}

	id := uuid.NewString()
	job := QueueJob{
		ID:        id,
		Kind:      kind,
		RequestID: requestID,
		Status:    jobStatusPending,
		Body:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.writeLocked(job); err != nil {
		return "", err
	}

	return id, nil
}

func (s *JobStore) Get(id string) (QueueJob, bool) {
	if s == nil {
		return QueueJob{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked(id)
}

func (s *JobStore) PendingCount() int {
	if s == nil {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.countLocked(jobStatusPending) + s.countLocked(jobStatusRunning)
}

func (s *JobStore) CompletedCount() int64 {
	if s == nil {
		return 0
	}

	return s.completed.Load()
}

func (s *JobStore) Stats(queue *RequestQueue) domain.QueueStatsResponse {
	resp := domain.QueueStatsResponse{CompletedJobs: s.CompletedCount()}
	if queue != nil {
		resp.InFlight = queue.InFlight()
		resp.Capacity = queue.Capacity()
	}

	if s != nil {
		resp.PendingJobs = s.PendingCount()
	}

	return resp
}

func (s *JobStore) ClaimNext() (QueueJob, bool) {
	if s == nil {
		return QueueJob{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return QueueJob{}, false
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		job, ok := s.readLocked(id)
		if !ok || job.Status != jobStatusPending {
			continue
		}

		job.Status = jobStatusRunning
		job.UpdatedAt = time.Now()
		if err := s.writeLocked(job); err != nil {
			continue
		}

		return job, true
	}

	return QueueJob{}, false
}

func (s *JobStore) Complete(id string, result any, runErr error) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.readLocked(id)
	if !ok {
		return fmt.Errorf("job %s не найден", id)
	}

	job.UpdatedAt = time.Now()
	if runErr != nil {
		job.Status = jobStatusFailed
		job.Error = runErr.Error()
	} else {
		job.Status = jobStatusCompleted
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}
		job.Result = raw
		s.completed.Add(1)
	}

	return s.writeLocked(job)
}

func (s *JobStore) jobPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *JobStore) readLocked(id string) (QueueJob, bool) {
	data, err := os.ReadFile(s.jobPath(id))
	if err != nil {
		return QueueJob{}, false
	}

	var job QueueJob
	if err := json.Unmarshal(data, &job); err != nil {
		return QueueJob{}, false
	}

	return job, true
}

func (s *JobStore) writeLocked(job QueueJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return os.WriteFile(s.jobPath(job.ID), data, 0o644)
}

func (s *JobStore) countLocked(status string) int {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0
	}

	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".json")
		job, ok := s.readLocked(id)
		if ok && job.Status == status {
			n++
		}
	}

	return n
}

type JobRunner struct {
	store *JobStore
	queue *RequestQueue
	chat  *ChatService
	agent *AgentService
}

func NewJobRunner(store *JobStore, queue *RequestQueue, chat *ChatService, agent *AgentService) *JobRunner {
	if store == nil {
		return nil
	}

	return &JobRunner{
		store: store,
		queue: queue,
		chat:  chat,
		agent: agent,
	}
}

func (r *JobRunner) RunLoop(ctx context.Context) {
	if r == nil {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.processOne(ctx)
		}
	}
}

func (r *JobRunner) processOne(ctx context.Context) {
	if r.queue != nil {
		if err := r.queue.Acquire(ctx); err != nil {
			return
		}
	}

	job, ok := r.store.ClaimNext()
	if !ok {
		if r.queue != nil {
			r.queue.Release()
		}
		return
	}

	go func(job QueueJob) {
		defer func() {
			if r.queue != nil {
				r.queue.Release()
			}
		}()
		result, err := r.execute(ctx, job)
		_ = r.store.Complete(job.ID, result, err)
	}(job)
}

func (r *JobRunner) execute(ctx context.Context, job QueueJob) (any, error) {
	switch job.Kind {
	case "chat":
		var req domain.ChatRequest
		if err := json.Unmarshal(job.Body, &req); err != nil {
			return nil, err
		}

		if r.chat == nil {
			return nil, errors.New("chat service не инициализирован")
		}
		return r.chat.Complete(ctx, req)
	case "agent_step":
		var req domain.AgentStepRequest
		if err := json.Unmarshal(job.Body, &req); err != nil {
			return nil, err
		}

		if r.agent == nil {
			return nil, errors.New("agent service не инициализирован")
		}
		return r.agent.Step(ctx, req)
	default:
		return nil, fmt.Errorf("неизвестный kind %q", job.Kind)
	}
}

func (s *JobStore) ToStatus(job QueueJob) domain.QueueJobStatus {
	out := domain.QueueJobStatus{
		JobID:     job.ID,
		Status:    job.Status,
		Kind:      job.Kind,
		RequestID: job.RequestID,
		Error:     job.Error,
	}

	if len(job.Result) > 0 {
		var result any
		if err := json.Unmarshal(job.Result, &result); err == nil {
			out.Result = result
		}
	}

	return out
}
