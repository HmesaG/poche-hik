package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"ponches/internal/employees"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// validateEmployee validates required fields and data integrity
func validateEmployee(e *employees.Employee) error {
	if strings.TrimSpace(e.FirstName) == "" {
		return errors.New("firstName is required")
	}
	if strings.TrimSpace(e.LastName) == "" {
		return errors.New("lastName is required")
	}
	if strings.TrimSpace(e.EmployeeNo) == "" {
		return errors.New("employeeNo is required")
	}
	if e.BirthDate.IsZero() {
		return errors.New("birthDate is required")
	}
	if e.BirthDate.After(time.Now()) {
		return errors.New("birthDate cannot be in the future")
	}
	if e.HireDate.IsZero() {
		return errors.New("hireDate is required")
	}
	if e.Status == "" {
		e.Status = "Active"
	}
	validStatuses := map[string]bool{"Active": true, "Inactive": true, "Suspended": true, "Terminated": true}
	if !validStatuses[e.Status] {
		return errors.New("invalid status. Valid values: Active, Inactive, Suspended, Terminated")
	}
	if e.BaseSalary < 0 {
		return errors.New("baseSalary cannot be negative")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	var emp employees.Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateEmployee(&emp); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if emp.ID == "" {
		emp.ID = uuid.New().String()
	}

	if err := s.Store.CreateEmployee(r.Context(), &emp); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Employee number already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create employee")
		return
	}

	writeJSON(w, http.StatusCreated, emp)
}

func (s *Server) handleGetEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Employee ID is required")
		return
	}

	emp, err := s.Store.GetEmployee(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get employee")
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	writeJSON(w, http.StatusOK, emp)
}

func (s *Server) handleUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Employee ID is required")
		return
	}

	var emp employees.Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	emp.ID = id

	if err := validateEmployee(&emp); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.Store.UpdateEmployee(r.Context(), &emp); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Employee not found")
			return
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Employee number already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update employee")
		return
	}

	writeJSON(w, http.StatusOK, emp)
}

func (s *Server) handleDeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Employee ID is required")
		return
	}

	if err := s.Store.DeleteEmployee(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Employee not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete employee")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
