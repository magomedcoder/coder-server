package llmrunner

import (
	"context"
	"testing"
)

func TestConnect_requiresEndpoint(t *testing.T) {
	_, err := Connect(context.Background(), ConnectConfig{})
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
}

func TestConnect_buildsRegistry(t *testing.T) {
	r, err := Connect(context.Background(), ConnectConfig{
		Endpoints: []ConnectEndpoint{
			{
				Address:       "127.0.0.1:50052",
				SelectedModel: "m1",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if r.LLM == nil || r.Pool == nil || r.Registry == nil {
		t.Fatal("компоненты nil")
	}
}
