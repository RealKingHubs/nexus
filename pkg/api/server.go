package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server encapsulates the background daemon configuration
type Server struct {
	listenAddr string
}

// NewServer initializes the engine API daemon instance
func NewServer(addr string) *Server {
	return &Server{listenAddr: addr}
}

// Start launches both the HTTP API surface and the continuous background loop
func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// Core API endpoint routings
	mux.HandleFunc("/healthz", s.handleHealthCheck)
	mux.HandleFunc("/api/v1/contracts/apply", s.handleContractApply)

	srv := &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
	}

	// 1. Fire up the continuous drift detection daemon loop asynchronously
	go s.startReconciliationDaemon()

	// 2. Setup graceful shutdown listener mechanics
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("🚀 Nexus Control Plane Daemon listening on %s\n", s.listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("❌ Server error crash: %v\n", err)
		}
	}()

	// Block here until a termination signal hits the process block
	<-shutdownChan
	fmt.Println("\n🛑 Shutting down Nexus control plane daemon cleanly...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy","engine":"nexus-v1alpha1"}`))
}

func (s *Server) handleContractApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Future: Accept target incoming contract bytes and feed them to the reconciler engine
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"message":"Intent contract accepted for background processing"}`))
}

// startReconciliationDaemon models the heart of an intent control plane
func (s *Server) startReconciliationDaemon() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	fmt.Println("🔄 Background Drift Detection Daemon initialized successfully.")

	for {
		select {
		case <-ticker.C:
			// Future: Scan etcd registry contracts and run provider reconciliation sweeps continuously
			fmt.Println("🔍 [Reconciliation Loop] Scanning active cluster contracts for environmental configuration drift...")
		}
	}
}