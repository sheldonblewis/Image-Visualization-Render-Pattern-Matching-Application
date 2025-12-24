package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/worldlabs/image-grid-viewer/backend/config"
	"github.com/worldlabs/image-grid-viewer/backend/service"
	"github.com/worldlabs/image-grid-viewer/backend/storage"
)

func main() {
	cfg := config.Load()
	httpClient := &http.Client{Timeout: cfg.RequestTimeout}
	storageClient := storage.NewHTTPClient(httpClient)
	querySvc := service.NewQueryService(cfg, storageClient)

	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", healthCheckHandler).Methods("GET")
	api.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		queryHandler(querySvc, w, r)
	}).Methods("POST")

	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	srv := &http.Server{
		Handler: c.Handler(router),
		Addr:    ":" + cfg.Port,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on :%s\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// Health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func queryHandler(svc *service.QueryService, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req service.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	resp, err := svc.Query(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if service.IsClientError(err) {
			status = http.StatusBadRequest
		}
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), status)
		return
	}

	json.NewEncoder(w).Encode(resp)
}
