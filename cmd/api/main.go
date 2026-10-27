package main

import (
	"log"
	"net/http"
	"proGo/db"
	"proGo/internal/api"
)

func main() {
	cfg := db.LoadConfig()
	s := api.NewServer(cfg)
	router := s.SetupRouters()

	log.Printf("Server starting on port %s", cfg.AppPort)
	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, router))
}
