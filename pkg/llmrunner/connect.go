package llmrunner

import (
	"context"
	"fmt"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
)

type ConnectEndpoint struct {
	Address       string
	Name          string
	SelectedModel string
}

type ConnectConfig struct {
	Endpoints      []ConnectEndpoint
	ProbeOnConnect bool
}

type ConnectResult struct {
	LLM      domain.LLMRepository
	Pool     *Pool
	Registry *Registry
}

func Connect(ctx context.Context, cfg ConnectConfig) (*ConnectResult, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("llmrunner: нужен хотя бы один ConnectEndpoint")
	}

	states := make([]RunnerState, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		addr := strings.TrimSpace(ep.Address)
		if addr == "" {
			continue
		}

		states = append(states, RunnerState{
			Address:       addr,
			Name:          strings.TrimSpace(ep.Name),
			SelectedModel: strings.TrimSpace(ep.SelectedModel),
			Enabled:       true,
		})
	}

	if len(states) == 0 {
		return nil, fmt.Errorf("llmrunner: все Address пустые")
	}

	reg := NewRegistry(states)
	pool := NewPool(reg)

	if cfg.ProbeOnConnect {
		for _, st := range states {
			ok := pool.ProbeLLMRunner(ctx, st.Address).Connected
			if !ok {
				return nil, fmt.Errorf("llmrunner: runner %q недоступен", st.Address)
			}
		}
	}

	return &ConnectResult{
		LLM:      pool,
		Pool:     pool,
		Registry: reg,
	}, nil
}

func MustConnect(ctx context.Context, cfg ConnectConfig) *ConnectResult {
	r, err := Connect(ctx, cfg)
	if err != nil {
		panic(err)
	}

	return r
}

func ConnectFromAddresses(ctx context.Context, addresses []string, probe bool) (*ConnectResult, error) {
	eps := make([]ConnectEndpoint, 0, len(addresses))
	for _, a := range addresses {
		eps = append(eps, ConnectEndpoint{Address: a})
	}

	return Connect(ctx, ConnectConfig{
		Endpoints:      eps,
		ProbeOnConnect: probe,
	})
}
