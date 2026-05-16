package main

import (
	"github.com/magomedcoder/tce-server/internal/app"
	"github.com/magomedcoder/tce-server/internal/config"
	"log"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
