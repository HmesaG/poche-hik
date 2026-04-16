package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"ponches/internal/employees"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func validateDepartment(d *employees.Department) error {
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("department name is required")
	}
	return nil
}

func validatePosition(p *employees.Position) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("position name is required")
	}
	if p.Level < 0 {
		return errors.New("position level cannot be negative")
	}
	return nil
}

func (s *Server) handleCreateDepartment(w http.ResponseWriter, r *http.Request) {
	var d employees.Department
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateDepartment(&d); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if d.ID == "" {
		d.ID = uuid.New().String()
	}

	if err := s.Store.CreateDepartment(r.Context(), &d); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Department already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create department")
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) handleGetDepartment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Department ID is required")
		return
	}

	dept, err := s.Store.GetDepartment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get department")
		return
	}
	if dept == nil {
		writeError(w, http.StatusNotFound, "Department not found")
		return
	}

	writeJSON(w, http.StatusOK, dept)
}

func (s *Server) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListDepartments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list departments")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUpdateDepartment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Department ID is required")
		return
	}

	var d employees.Department
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	d.ID = id

	if err := validateDepartment(&d); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.Store.UpdateDepartment(r.Context(), &d); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Department not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update department")
		return
	}

	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleDeleteDepartment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Department ID is required")
		return
	}

	if err := s.Store.DeleteDepartment(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Department not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete department")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreatePosition(w http.ResponseWriter, r *http.Request) {
	var p employees.Position
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validatePosition(&p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if p.ID == "" {
		p.ID = uuid.New().String()
	}

	if err := s.Store.CreatePosition(r.Context(), &p); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Position already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create position")
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetPosition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Position ID is required")
		return
	}

	pos, err := s.Store.GetPosition(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get position")
		return
	}
	if pos == nil {
		writeError(w, http.StatusNotFound, "Position not found")
		return
	}

	writeJSON(w, http.StatusOK, pos)
}

func (s *Server) handleListPositions(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListPositions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list positions")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUpdatePosition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Position ID is required")
		return
	}

	var p employees.Position
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	p.ID = id

	if err := validatePosition(&p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.Store.UpdatePosition(r.Context(), &p); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Position not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update position")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeletePosition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Position ID is required")
		return
	}

	if err := s.Store.DeletePosition(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Position not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete position")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
