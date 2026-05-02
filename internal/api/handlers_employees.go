package api

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"ponches/internal/employees"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type employeePayload struct {
	ID           string  `json:"id"`
	EmployeeNo   string  `json:"employeeNo"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	IDNumber     string  `json:"idNumber"`
	Gender       string  `json:"gender"`
	BirthDate    string  `json:"birthDate"`
	PhotoURL     string  `json:"photoUrl"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Address      string  `json:"address"`
	DepartmentID string  `json:"departmentId"`
	PositionID   string  `json:"positionId"`
	HireDate     string  `json:"hireDate"`
	Status       string  `json:"status"`
	BaseSalary   float64 `json:"baseSalary"`
	FaceID       string  `json:"faceId"`
	FleetNo      string  `json:"fleetNo"`
	PersonalNo   string  `json:"personalNo"`
	PhotoRemoved bool    `json:"photoRemoved"`
}

func (s *Server) markEmployeeAdminStatus(ctx context.Context, emp *employees.Employee) error {
	if emp == nil {
		return nil
	}

	isAdmin, err := s.Store.HasAdminUserByEmail(ctx, emp.Email)
	if err != nil {
		return err
	}
	emp.IsSystemAdmin = isAdmin
	return nil
}

func (s *Server) markEmployeesAdminStatus(ctx context.Context, list []*employees.Employee) error {
	for _, emp := range list {
		if err := s.markEmployeeAdminStatus(ctx, emp); err != nil {
			return err
		}
	}
	return nil
}

func parseEmployeePayload(r io.Reader) (*employees.Employee, *employeePayload, error) {
	var payload employeePayload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, nil, err
	}

	parseDate := func(value string) (time.Time, error) {
		value = strings.TrimSpace(value)
		if value == "" {
			return time.Time{}, nil
		}

		layouts := []string{
			"2006-01-02",
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, value); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid date: %s", value)
	}

	birthDate, err := parseDate(payload.BirthDate)
	if err != nil {
		return nil, nil, err
	}
	hireDate, err := parseDate(payload.HireDate)
	if err != nil {
		return nil, nil, err
	}

	return &employees.Employee{
		ID:           payload.ID,
		EmployeeNo:   payload.EmployeeNo,
		FirstName:    payload.FirstName,
		LastName:     payload.LastName,
		IDNumber:     payload.IDNumber,
		Gender:       payload.Gender,
		BirthDate:    birthDate,
		PhotoURL:     payload.PhotoURL,
		Phone:        payload.Phone,
		Email:        payload.Email,
		Address:      payload.Address,
		DepartmentID: payload.DepartmentID,
		PositionID:   payload.PositionID,
		HireDate:     hireDate,
		Status:       payload.Status,
		BaseSalary:   payload.BaseSalary,
		FaceID:       payload.FaceID,
		FleetNo:      payload.FleetNo,
		PersonalNo:   payload.PersonalNo,
	}, &payload, nil
}

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
	emp, _, err := parseEmployeePayload(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateEmployee(emp); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if emp.ID == "" {
		emp.ID = uuid.New().String()
	}

	if err := s.Store.CreateEmployee(r.Context(), emp); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Employee number already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create employee")
		return
	}

	s.queueEmployeeAutoSync(emp, "create", false)
	s.LogAudit(r.Context(), r, "CREATE_EMPLOYEE", emp.EmployeeNo, emp)

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
	if err := s.markEmployeeAdminStatus(r.Context(), emp); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to resolve employee role")
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

	emp, payload, err := parseEmployeePayload(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	emp.ID = id

	if err := validateEmployee(emp); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.Store.UpdateEmployee(r.Context(), emp); err != nil {
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

	if payload.PhotoRemoved {
		if err := s.markEmployeeAdminStatus(r.Context(), emp); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to resolve employee role")
			return
		}
		if emp.IsSystemAdmin {
			writeError(w, http.StatusConflict, "No se puede quitar la foto de un empleado administrador del sistema")
			return
		}
		if err := s.Store.ClearEmployeePhoto(r.Context(), emp.EmployeeNo); err != nil {
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "Employee not found")
				return
			}
			log.Error().Err(err).Str("employeeNo", emp.EmployeeNo).Msg("Failed to clear employee photo during update")
			writeError(w, http.StatusInternalServerError, "Failed to clear employee photo")
			return
		}
		emp.PhotoURL = ""
		emp.PhotoData = nil
		emp.FaceID = ""
	}

	s.queueEmployeeAutoSync(emp, "update", payload.PhotoRemoved)

	writeJSON(w, http.StatusOK, emp)
}

func (s *Server) queueEmployeeAutoSync(emp *employees.Employee, action string, photoRemoved bool) {
	if emp == nil || strings.TrimSpace(emp.EmployeeNo) == "" {
		return
	}

	employeeNo := strings.TrimSpace(emp.EmployeeNo)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		empCopy, err := s.Store.GetEmployeeByNo(ctx, employeeNo)
		if err != nil {
			log.Warn().Err(err).Str("employeeNo", employeeNo).Msg("Failed to reload employee for auto-sync")
			return
		}
		if empCopy == nil {
			return
		}

		summary := s.syncEmployeeToAllDevices(ctx, empCopy, photoRemoved, "PushEmployee")
		if summary.DevicesFailed > 0 {
			log.Warn().
				Str("employeeNo", empCopy.EmployeeNo).
				Int("devicesFailed", summary.DevicesFailed).
				Msgf("Employee auto-sync %s completed with errors", action)
		}
	}()
}

func (s *Server) handleUploadEmployeePhoto(w http.ResponseWriter, r *http.Request) {
	employeeNo := strings.TrimSpace(chi.URLParam(r, "employeeNo"))
	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "Employee number is required")
		return
	}

	var req struct {
		Photo string `json:"photo"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 7<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "Photo is required")
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	photoBase64 := strings.TrimSpace(req.Photo)
	if comma := strings.Index(photoBase64, ","); comma >= 0 {
		photoBase64 = photoBase64[comma+1:]
	}
	if photoBase64 == "" {
		writeError(w, http.StatusBadRequest, "Photo is required")
		return
	}

	photoData, err := base64.StdEncoding.DecodeString(photoBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid photo format")
		return
	}
	if len(photoData) > 5<<20 {
		writeError(w, http.StatusBadRequest, "Photo too large (max 5MB)")
		return
	}

	emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Employee not found")
			return
		}
		log.Error().Err(err).Str("employeeNo", employeeNo).Msg("Failed to load employee before photo save")
		writeError(w, http.StatusInternalServerError, "Failed to load employee")
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	photoData, meta, err := prepareFaceImage(photoData)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.storeEmployeePhoto(r.Context(), emp, photoData); err != nil {
		log.Error().Err(err).Str("employeeNo", employeeNo).Msg("Failed to save employee photo")
		writeError(w, http.StatusInternalServerError, "Failed to save photo")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	summary := s.syncEmployeeToAllDevices(ctx, emp, false, "PushEmployee")
	if summary.DevicesTotal == 0 {
		log.Warn().Str("employeeNo", employeeNo).Msg("No managed devices configured. Photo saved locally only.")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Foto actualizada para empleado %s", employeeNo),
		"image": map[string]int{
			"width":  meta.Width,
			"height": meta.Height,
			"bytes":  meta.Bytes,
		},
		"sync": summary,
	})
}

func (s *Server) handleDeleteEmployeePhoto(w http.ResponseWriter, r *http.Request) {
	employeeNo := strings.TrimSpace(chi.URLParam(r, "employeeNo"))
	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "Employee number is required")
		return
	}

	emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get employee")
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, "Employee not found")
		return
	}
	if err := s.markEmployeeAdminStatus(r.Context(), emp); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to resolve employee role")
		return
	}
	if emp.IsSystemAdmin {
		writeError(w, http.StatusConflict, "No se puede quitar la foto de un empleado administrador del sistema")
		return
	}

	if emp.PhotoURL != "" {
		photoPath := filepath.Join("web", strings.TrimPrefix(emp.PhotoURL, "/"))
		if err := os.Remove(photoPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warn().Err(err).Str("path", photoPath).Msg("Failed to delete local employee photo")
		}
	}

	if err := s.Store.ClearEmployeePhoto(r.Context(), employeeNo); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Employee not found")
			return
		}
		log.Error().Err(err).Str("employeeNo", employeeNo).Msg("Failed to clear employee photo data")
		writeError(w, http.StatusInternalServerError, "Failed to clear employee photo")
		return
	}

	emp.PhotoURL = ""
	emp.PhotoData = nil
	emp.FaceID = ""

	go func(employeeNo string) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		summary := s.removeEmployeePhotoFromAllDevices(ctx, employeeNo)
		if summary.DevicesFailed > 0 {
			log.Warn().Str("employeeNo", employeeNo).Int("devicesFailed", summary.DevicesFailed).Msg("Photo deletion sync completed with errors")
		}
	}(employeeNo)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Foto eliminada para empleado %s", employeeNo),
	})
}

func (s *Server) handleDeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Employee ID is required")
		return
	}

	// Get employee to check for photo before deleting
	emp, err := s.Store.GetEmployee(r.Context(), id)
	if err == nil && emp != nil && emp.PhotoURL != "" {
		// Attempt to delete local photo
		photoPath := filepath.Join("web", emp.PhotoURL)
		if err := os.Remove(photoPath); err != nil {
			log.Warn().Err(err).Str("path", photoPath).Msg("Failed to delete local employee photo")
		}
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
