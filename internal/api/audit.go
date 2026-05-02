package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"ponches/internal/auth"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type AuditLog struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Details   string `json:"details"`
	IPAddress string `json:"ipAddress"`
	Username  string `json:"username"`
}

func (s *Server) LogAudit(ctx context.Context, r *http.Request, action, resource string, details interface{}) {
	detailsJSON, _ := json.Marshal(details)
	
	userID := ""
	username := ""
	if r != nil {
		if user, ok := auth.GetUserFromContext(ctx); ok {
			userID = user.UserID
			username = user.Username
		}
	}

	ip := ""
	if r != nil {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	audit := &AuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   string(detailsJSON),
		IPAddress: ip,
		Username:  username,
	}

	// We'll save this to the DB
	// Need to add SaveAuditLog to store interface
	log.Info().
		Str("action", action).
		Str("resource", resource).
		Str("user", userID).
		Msg("Audit Log Entry")
		
	s.Store.SaveAuditLog(ctx, audit) 
}

func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.Store.ListAuditLogs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}
