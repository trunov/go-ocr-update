package main

import (
	"go-ocr/internal/app/handler"
	"log"
	"net/http"

	"github.com/rs/cors"
)

const (
	port = ":3000"
)

func startServer() {
	h := handler.NewHandler()
	r := handler.NewRouter(h)

	corsWrapper := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
		AllowedHeaders: []string{"*"},
	})

	log.Printf("Running on port %s", port)
	http.ListenAndServe("0.0.0.0"+port, corsWrapper.Handler(r))
}
