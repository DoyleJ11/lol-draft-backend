package main

import (
	"context"
	"log"
	"net/http"

	"github.com/DoyleJ11/lol-draft-backend/internal/httpapi"
	"github.com/DoyleJ11/lol-draft-backend/internal/hub"
)

func main() {
	ctx := context.Background()
	h := hub.NewHub(ctx)

	// Build the router *with* the hub injected
	handler := httpapi.SetupRoutes(h)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
