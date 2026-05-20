package runnerconnect

import (
	"context"
	"testing"
)

func TestConnect_requiresEndpoint(t *testing.T) {
	_, err := Connect(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConnect_buildsRegistry(t *testing.T) {
	r, err := Connect(context.Background(), Config{
		Endpoints: []Endpoint{
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
		t.Fatal("nil components")
	}
}
