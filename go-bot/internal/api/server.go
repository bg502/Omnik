package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// QueryRequest represents an API query request
type QueryRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

// QueryResponse represents an API query response
type QueryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// MessageHandler processes incoming API messages
type MessageHandler func(ctx context.Context, message string, sessionID string) error

// Server represents the HTTP API server
type Server struct {
	port           int
	messageHandler MessageHandler
	server         *http.Server
	mu             sync.Mutex
}

// New creates a new API server
func New(port int, handler MessageHandler) *Server {
	return &Server{
		port:           port,
		messageHandler: handler,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("[API] Starting HTTP server on port %d", s.port)

	go func() {
		<-ctx.Done()
		log.Printf("[API] Shutting down HTTP server...")
		s.server.Shutdown(context.Background())
	}()

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// handleQuery handles POST /api/query
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, QueryResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid JSON: %v", err),
		})
		return
	}

	if req.Message == "" {
		respondJSON(w, http.StatusBadRequest, QueryResponse{
			Success: false,
			Error:   "Message field is required",
		})
		return
	}

	log.Printf("[API] Received query: %s (session: %s)", req.Message, req.SessionID)

	// Process the message
	if err := s.messageHandler(r.Context(), req.Message, req.SessionID); err != nil {
		respondJSON(w, http.StatusInternalServerError, QueryResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to process message: %v", err),
		})
		return
	}

	respondJSON(w, http.StatusOK, QueryResponse{
		Success: true,
		Message: "Query accepted and being processed",
	})
}

// handleHealth handles GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
