package api

import (
	"encoding/json"
	"net/http"
	"ponches/internal/employees"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) handleListHolidays(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListHolidays(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list holidays")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateHoliday(w http.ResponseWriter, r *http.Request) {
	var h employees.Holiday
	if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	if h.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if h.Date.IsZero() {
		writeError(w, http.StatusBadRequest, "Date is required")
		return
	}

	if err := s.Store.CreateHoliday(r.Context(), &h); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create holiday")
		return
	}

	writeJSON(w, http.StatusCreated, h)
}

func (s *Server) handleUpdateHoliday(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var h employees.Holiday
	if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	h.ID = id
	if err := s.Store.UpdateHoliday(r.Context(), &h); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update holiday")
		return
	}

	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleDeleteHoliday(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.Store.DeleteHoliday(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete holiday")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetHoliday(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h, err := s.Store.GetHoliday(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get holiday")
		return
	}
	if h == nil {
		writeError(w, http.StatusNotFound, "Holiday not found")
		return
	}
	writeJSON(w, http.StatusOK, h)
}
