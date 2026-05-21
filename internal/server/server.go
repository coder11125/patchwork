package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder11125/patchwork/internal/pipeline"
)

type Server struct {
	pipeline *pipeline.Pipeline
	logger   *slog.Logger
	addr     string
}

func New(p *pipeline.Pipeline, addr string) *Server {
	return &Server{
		pipeline: p,
		logger:   slog.Default(),
		addr:     addr,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /detect", s.handleDetect)
	mux.HandleFunc("POST /plan", s.handlePlan)
	mux.HandleFunc("POST /run", s.handleRun)
	mux.HandleFunc("GET /health", s.handleHealth)

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		s.logger.Info("shutting down api server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	s.logger.Info("api server starting", "addr", s.addr)
	return srv.ListenAndServe()
}

func (s *Server) handleDetect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Dir == "" {
		req.Dir = "."
	}

	results, err := s.pipeline.Detect(r.Context())
	if err != nil {
		s.logger.Error("detect failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Dir == "" {
		req.Dir = "."
	}

	detections, err := s.pipeline.Detect(r.Context())
	if err != nil {
		s.logger.Error("detect failed during plan", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	changelogs, err := s.pipeline.Analyze(r.Context(), detections)
	if err != nil {
		s.logger.Error("analyze failed during plan", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	plan, err := s.pipeline.Plan(r.Context(), detections, changelogs)
	if err != nil {
		s.logger.Error("plan failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Dir == "" {
		req.Dir = "."
	}

	plan, err := s.pipeline.Run(r.Context())
	if err != nil {
		s.logger.Error("run failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
