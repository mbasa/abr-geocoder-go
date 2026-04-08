// Package server provides the REST API server for abr-geocoder.
// Ported from TypeScript: src/interface/abrg-api-server/index.ts
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	"github.com/mbasa/abr-geocoder-go/internal/interface/format"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
)

// Server provides the REST API for geocoding
type Server struct {
	httpServer *http.Server
	geocoder   *geocode.Geocoder
	port       int
}

// ServerOptions configures the server
type ServerOptions struct {
	Port      int
	DataDir   string
	FuzzyChar string
	Debug     bool
}

// New creates a new Server
func New(opts ServerOptions) (*Server, error) {
	if opts.Port == 0 {
		opts.Port = 8143
	}

	g, err := geocode.New(geocode.GeocoderOptions{
		DataDir:      opts.DataDir,
		FuzzyChar:    opts.FuzzyChar,
		SearchTarget: types.SearchTargetAll,
		Debug:        opts.Debug,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize geocoder: %w", err)
	}

	s := &Server{
		geocoder: g,
		port:     opts.Port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/geocode", s.handleGeocode)
	mux.HandleFunc("/", s.handleNotFound)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting geocoder server on port %d", s.port)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return s.geocoder.Close()
}

// handleGeocode handles geocoding requests
// GET /geocode?address=<address>&format=<format>&fuzzy=<fuzzy>&target=<target>
func (s *Server) handleGeocode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "address parameter is required", http.StatusBadRequest)
		return
	}

	// Parse output format
	formatStr := r.URL.Query().Get("format")
	if formatStr == "" {
		formatStr = "json"
	}
	outputFormat := types.OutputFormat(strings.ToLower(formatStr))
	if !outputFormat.IsValid() {
		http.Error(w, fmt.Sprintf("invalid format: %s", formatStr), http.StatusBadRequest)
		return
	}

	// Parse debug flag
	debugStr := r.URL.Query().Get("debug")
	debug, _ := strconv.ParseBool(debugStr)

	// Geocode the address
	result, err := s.geocoder.Geocode(address)
	if err != nil {
		log.Printf("Geocoding error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", outputFormat.MimeType()+"; charset=utf-8")

	// Write formatted output
	if err := format.WriteResults(w, []*geocodemodels.GeoCodeResult{result}, outputFormat, debug); err != nil {
		log.Printf("Format error: %v", err)
	}
}

// handleNotFound handles unknown routes
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Redirect to geocode endpoint
		http.Redirect(w, r, "/geocode", http.StatusMovedPermanently)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf("endpoint %s not found", r.URL.Path),
	})
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

