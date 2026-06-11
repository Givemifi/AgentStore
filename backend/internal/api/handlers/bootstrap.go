package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"agentstore/internal/bootstrap"
	"agentstore/internal/db"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

type BootstrapHandler struct {
	db *db.MongoDB

	mu          sync.RWMutex
	initialized bool
}

func NewBootstrapHandler(database *db.MongoDB) *BootstrapHandler {
	h := &BootstrapHandler{
		db: database,
	}
	// Check initial state
	h.refreshInitialized()
	return h
}

func (h *BootstrapHandler) refreshInitialized() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sys models.SystemConfig
	err := h.db.SystemConfig().FindOne(ctx, bson.M{}).Decode(&sys)
	if err == nil && sys.Initialized {
		h.mu.Lock()
		h.initialized = true
		h.mu.Unlock()
	}
}

func (h *BootstrapHandler) IsInitialized() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.initialized
}

type bootstrapStatusResponse struct {
	Initialized bool `json:"initialized"`
}

func (h *BootstrapHandler) Status(w http.ResponseWriter, r *http.Request) {
	if h.IsInitialized() {
		respondWithJSON(w, http.StatusOK, bootstrapStatusResponse{Initialized: true})
		return
	}
	// Re-check DB in case CLI initialized since startup
	h.refreshInitializedFromContext(r)
	respondWithJSON(w, http.StatusOK, bootstrapStatusResponse{Initialized: h.IsInitialized()})
}

func (h *BootstrapHandler) refreshInitializedFromContext(r *http.Request) {
	var sys models.SystemConfig
	err := h.db.SystemConfig().FindOne(r.Context(), bson.M{}).Decode(&sys)
	if err == nil && sys.Initialized {
		h.mu.Lock()
		h.initialized = true
		h.mu.Unlock()
	}
}

// BootstrapGuard returns middleware that blocks non-bootstrap routes when system is not initialized.
func (h *BootstrapHandler) BootstrapGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.IsInitialized() {
			next.ServeHTTP(w, r)
			return
		}
		// Re-check DB
		h.refreshInitializedFromContext(r)
		if h.IsInitialized() {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "System not initialized",
			"redirect": "/setup",
		})
	})
}

type bootstrapSetupRequest struct {
	OrgName     string `json:"org"`
	DisplayName string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

// Setup handles POST /api/bootstrap/setup. It creates the root tenant and owner
// account on the first run. Returns 409 if the system is already initialized,
// 400 if input is invalid, and 200 on success.
func (h *BootstrapHandler) Setup(w http.ResponseWriter, r *http.Request) {
	// Re-check so a concurrent CLI setup is respected
	h.refreshInitializedFromContext(r)
	if h.IsInitialized() {
		respondWithError(w, http.StatusConflict, "System is already initialized")
		return
	}

	var req bootstrapSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Basic presence check before delegating to bootstrap (which also validates)
	req.OrgName = strings.TrimSpace(req.OrgName)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Email = strings.TrimSpace(req.Email)
	if req.OrgName == "" || req.DisplayName == "" || req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "All fields are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	_, err := bootstrap.InitializeSystem(ctx, h.db, bootstrap.SetupInput{
		OrgName:     req.OrgName,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Password:    req.Password,
	})
	if errors.Is(err, bootstrap.ErrAlreadyInitialized) {
		respondWithError(w, http.StatusConflict, "System is already initialized")
		return
	}
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "password too weak") {
			respondWithError(w, http.StatusBadRequest, msg)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Setup failed: "+msg)
		return
	}

	// Mark in-memory so subsequent requests are served immediately
	h.mu.Lock()
	h.initialized = true
	h.mu.Unlock()

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"initialized": true})
}
