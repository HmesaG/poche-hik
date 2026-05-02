package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	imageData, meta, err := prepareFaceImage(imageData)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if meta.Bytes < 10<<10 {
		writeError(w, http.StatusBadRequest, "Image too small after normalization (min 10KB)")
		return
	}

	if err := s.storeEmployeePhoto(r.Context(), emp, imageData); err != nil {
		log.Error().Err(err).Str("employeeNo", employeeNo).Msg("Failed to save face locally before sync")
		writeError(w, http.StatusInternalServerError, "Failed to save photo locally")
		return
	}

	summary := s.syncEmployeeToAllDevices(r.Context(), emp, false, "RegisterFace")
	if summary.DevicesTotal == 0 {
		writeError(w, http.StatusServiceUnavailable, "No hay dispositivos configurados")
		return
	}
	if summary.DevicesSuccess == 0 {
		log.Error().Str("employeeNo", employeeNo).Interface("sync", summary).Msg("Face registration failed on all devices")
		writeError(w, http.StatusBadGateway, "No se pudo registrar la foto en los dispositivos configurados")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Rostro registrado exitosamente",
		"sync":    summary,
		"image": map[string]int{
			"width":  meta.Width,
			"height": meta.Height,
			"bytes":  meta.Bytes,
		},
	})
}

// handleDeleteFace deletes a face from the device
func (s *Server) handleDeleteFace(w http.ResponseWriter, r *http.Request) {
	employeeNo := chi.URLParam(r, "employeeNo")
	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "Employee number is required")
		return
	}

	emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
	if err == nil && emp != nil {
		if emp.PhotoURL != "" {
			photoPath := filepath.Join("web", strings.TrimPrefix(emp.PhotoURL, "/"))
			if err := os.Remove(photoPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Warn().Err(err).Str("path", photoPath).Msg("Failed to delete local employee photo")
			}
		}
		if err := s.Store.ClearEmployeePhoto(r.Context(), employeeNo); err != nil {
			log.Warn().Err(err).Str("employeeNo", employeeNo).Msg("Failed to clear local face data")
		}
	}

	summary := s.removeEmployeePhotoFromAllDevices(r.Context(), employeeNo)
	if summary.DevicesTotal == 0 {
		writeError(w, http.StatusServiceUnavailable, "No hay dispositivos configurados")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Rostro eliminado exitosamente",
		"sync":    summary,
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
		EmployeeNo   string `json:"employeeNo"`
		EmployeeName string `json:"employeeName"`
		HasFace      bool   `json:"hasFace"`
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

// handleFaceStatus returns the face registration status for an employee.
// It checks the database and optionally queries the device in real-time.
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

	hasDeviceFace := false
	if len(emp.PhotoData) == 0 || emp.FaceID == "" {
		devices, loadErr := s.loadSyncDevices(r.Context())
		if loadErr == nil {
			for _, device := range devices {
				client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
				deviceUser, userErr := client.GetUser(r.Context(), employeeNo)
				if userErr == nil && deviceUser != nil && deviceUser.NumOfFace > 0 {
					hasDeviceFace = true
					break
				}
			}
		}

		if _, _, err := s.importEmployeePhotoFromDevices(r.Context(), emp, false); err == nil {
			log.Info().Str("employeeNo", employeeNo).Msg("Face found on a configured device and imported locally")
			hasDeviceFace = true
		} else if err != nil {
			log.Debug().Err(err).Str("employeeNo", employeeNo).Msg("Face not found on configured devices or check failed")
		}
	} else {
		hasDeviceFace = true
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"employeeNo":    employeeNo,
		"employeeName":  emp.FirstName + " " + emp.LastName,
		"hasFace":       hasDeviceFace,
		"faceId":        emp.FaceID,
		"photoUrl":      emp.PhotoURL,
		"hasLocalPhoto": emp.PhotoURL != "" || len(emp.PhotoData) > 0,
	})
}

// handleImportFace imports the face image from the device to local storage
func (s *Server) handleImportFace(w http.ResponseWriter, r *http.Request) {
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

	log.Info().Str("employeeNo", employeeNo).Msg("Importing face from device")
	_, source, err := s.importEmployeePhotoFromDevices(r.Context(), emp, true)
	if err != nil {
		log.Error().Err(err).Str("employeeNo", employeeNo).Msg("Failed to download face from device")
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Error al descargar del dispositivo: %v. Este modelo puede reportar que tiene rostro, pero no exponer la imagen via ISAPI.", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"message":  fmt.Sprintf("Imagen importada correctamente desde %s", firstNonEmpty(source.Name, source.IP)),
		"photoUrl": emp.PhotoURL,
	})
}

// handleImportAllFaces imports face photos from the device for ALL employees that lack a local photo.
// Route: POST /api/devices/import-photos
func (s *Server) handleImportAllFaces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	emps, err := s.Store.ListEmployees(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list employees")
		return
	}

	imported := 0
	skipped := 0
	failed := 0
	var failedList []string

	for _, emp := range emps {
		// Skip if employee already has a local photo
		if len(emp.PhotoData) > 0 || emp.PhotoURL != "" {
			skipped++
			continue
		}

		_, _, err := s.importEmployeePhotoFromDevices(ctx, emp, false)
		if err != nil {
			failed++
			failedList = append(failedList, fmt.Sprintf("%s (%s): %v", emp.EmployeeNo, emp.FirstName+" "+emp.LastName, err))
			log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Msg("Could not import photo from device")
			continue
		}
		imported++
		log.Info().Str("employeeNo", emp.EmployeeNo).Msg("Photo imported from device successfully")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "completed",
		"message":  fmt.Sprintf("Importación completada: %d importadas, %d ya tenían foto, %d fallidas", imported, skipped, failed),
		"imported": imported,
		"skipped":  skipped,
		"failed":   failed,
		"errors":   failedList,
	})
}
