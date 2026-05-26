package runnerconnect

import (
	"context"
	"fmt"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/llmrunner"
)

type Endpoint struct {
	Address       string
	Name          string
	SelectedModel string
}

type Config struct {
	Endpoints      []Endpoint
	ProbeOnConnect bool
}

type Result struct {
	LLM      domain.LLMRepository
	Pool     *llmrunner.Pool
	Registry *llmrunner.Registry
}

func Connect(ctx context.Context, cfg Config) (*Result, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("runnerconnect: нужен хотя бы один Endpoint")
	}

	states := make([]llmrunner.RunnerState, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		addr := strings.TrimSpace(ep.Address)
		if addr == "" {
			continue
		}

		states = append(states, llmrunner.RunnerState{
			Address:       addr,
			Name:          strings.TrimSpace(ep.Name),
			SelectedModel: strings.TrimSpace(ep.SelectedModel),
			Enabled:       true,
		})
	}

	if len(states) == 0 {
		return nil, fmt.Errorf("runnerconnect: все Address пустые")
	}

	reg := llmrunner.NewRegistry(states)
	pool := llmrunner.NewPool(reg)

	if cfg.ProbeOnConnect {
		for _, st := range states {
			ok := pool.ProbeLLMRunner(ctx, st.Address).Connected
			if !ok {
				return nil, fmt.Errorf("runnerconnect: runner %q недоступен", st.Address)
			}
		}
	}

	return &Result{
		LLM:      pool,
		Pool:     pool,
		Registry: reg,
	}, nil
}

func MustConnect(ctx context.Context, cfg Config) *Result {
	r, err := Connect(ctx, cfg)
	if err != nil {
		panic(err)
	}

	return r
}

func FromAddresses(ctx context.Context, addresses []string, probe bool) (*Result, error) {
	eps := make([]Endpoint, 0, len(addresses))
	for _, a := range addresses {
		eps = append(eps, Endpoint{
			Address: a,
		})
	}

	return Connect(ctx, Config{
		Endpoints:      eps,
		ProbeOnConnect: probe,
	})
}
