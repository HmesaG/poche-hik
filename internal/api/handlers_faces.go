package api

import (
	"fmt"
	"io"
	"net/http"
	"ponches/internal/hikvision"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// handleRegisterFace registers a face for an employee
func (s *Server) handleRegisterFace(w http.ResponseWriter, r *http.Request) {
	employeeNo := chi.URLParam(r, "employeeNo")
	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "Employee number is required")
		return
	}

	// Get employee to verify exists
	emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get employee")
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	// Parse multipart form
	err = r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "File too large (max 10MB)")
		return
	}

	file, _, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Photo is required")
		return
	}
	defer file.Close()

	// Validate file type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		writeError(w, http.StatusBadRequest, "Invalid content type")
		return
	}

	imageData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read image")
		return
	}

	// Validate image size (min 10KB, max 10MB)
	if len(imageData) < 10<<10 {
		writeError(w, http.StatusBadRequest, "Image too small (min 10KB)")
		return
	}

	targetDeviceIP := s.Config.HikvisionIP
	if targetDeviceIP == "" {
		writeError(w, http.StatusServiceUnavailable, "No device IP configured")
		return
	}

	client := hikvision.NewClient(
		targetDeviceIP,
		80,
		s.Config.HikvisionUsername,
		s.Config.HikvisionPassword,
	)

	log.Info().Str("employeeNo", employeeNo).Str("device", targetDeviceIP).Msg("Registering face on device")

	err = client.RegisterFace(r.Context(), employeeNo, imageData)
	if err != nil {
		log.Error().Err(err).Msg("Hikvision face registration failed")
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Error en dispositivo: %v", err))
		return
	}

	// Update employee with face_id
	emp.FaceID = employeeNo // Use employeeNo as face ID
	if err := s.Store.UpdateEmployee(r.Context(), emp); err != nil {
		log.Warn().Err(err).Msg("Failed to update employee face_id")
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Rostro registrado exitosamente",
	})
}

// handleDeleteFace deletes a face from the device
func (s *Server) handleDeleteFace(w http.ResponseWriter, r *http.Request) {
	employeeNo := chi.URLParam(r, "employeeNo")
	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "Employee number is required")
		return
	}

	targetDeviceIP := s.Config.HikvisionIP
	if targetDeviceIP == "" {
		writeError(w, http.StatusServiceUnavailable, "No device IP configured")
		return
	}

	client := hikvision.NewClient(
		targetDeviceIP,
		80,
		s.Config.HikvisionUsername,
		s.Config.HikvisionPassword,
	)

	err := client.DeleteFace(r.Context(), employeeNo)
	if err != nil {
		log.Error().Err(err).Msg("Hikvision face deletion failed")
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Error al eliminar: %v", err))
		return
	}

	// Update employee
	emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
	if err == nil && emp != nil {
		emp.FaceID = ""
		s.Store.UpdateEmployee(r.Context(), emp)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Rostro eliminado exitosamente",
	})
}

// handleListFaces returns list of employees with registered faces
func (s *Server) handleListFaces(w http.ResponseWriter, r *http.Request) {
	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list employees")
		return
	}

	type FaceInfo struct {
		EmployeeNo  string `json:"employeeNo"`
		EmployeeName string `json:"employeeName"`
		HasFace     bool   `json:"hasFace"`
	}

	var faces []FaceInfo
	for _, emp := range emps {
		faces = append(faces, FaceInfo{
			EmployeeNo:   emp.EmployeeNo,
			EmployeeName: emp.FirstName + " " + emp.LastName,
			HasFace:      emp.FaceID != "",
		})
	}

	writeJSON(w, http.StatusOK, faces)
}

// handleFaceStatus returns the face registration status for an employee
func (s *Server) handleFaceStatus(w http.ResponseWriter, r *http.Request) {
	employeeNo := chi.URLParam(r, "employeeNo")
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"employeeNo":  employeeNo,
		"employeeName": emp.FirstName + " " + emp.LastName,
		"hasFace":     emp.FaceID != "",
		"faceId":      emp.FaceID,
	})
}
