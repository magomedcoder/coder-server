package main

import (
	"log"

	"github.com/magomedcoder/coder-server/internal/app"
	"github.com/magomedcoder/coder-server/internal/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("инициализация: %v", err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Fatalf("сервер: %v", err)
	}
}
