package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const managedDevicesConfigKey = "managed_devices"

type managedDevice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	Port      int       `json:"port"`
	Model     string    `json:"model,omitempty"`
	Serial    string    `json:"serial,omitempty"`
	Source    string    `json:"source,omitempty"`
	IsDefault bool      `json:"isDefault"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type managedDeviceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	Username    string    `json:"username"`
	Port        int       `json:"port"`
	Model       string    `json:"model,omitempty"`
	Serial      string    `json:"serial,omitempty"`
	Source      string    `json:"source,omitempty"`
	IsDefault   bool      `json:"isDefault"`
	HasPassword bool      `json:"hasPassword"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s *Server) handleListManagedDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	writeJSON(w, http.StatusOK, toManagedDeviceResponses(devices))
}

func (s *Server) handleCreateManagedDevice(w http.ResponseWriter, r *http.Request) {
	var req managedDevice
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.IP) == "" {
		writeError(w, http.StatusBadRequest, "Name and IP are required")
		return
	}

	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	req.ID = uuid.New().String()
	req.Name = strings.TrimSpace(req.Name)
	req.IP = strings.TrimSpace(req.IP)
	req.Username = strings.TrimSpace(req.Username)
	req.Port = normalizedDevicePort(req.Port)
	req.Source = firstNonEmpty(req.Source, "manual")
	req.UpdatedAt = time.Now()

	if req.IsDefault || len(devices) == 0 {
		req.IsDefault = true
		clearDefaultDevice(devices)
	}

	devices = append(devices, req)
	if err := s.persistManagedDevices(r.Context(), devices); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save device")
		return
	}

	if req.IsDefault {
		s.applyDefaultDevice(r.Context(), req)
	}

	writeJSON(w, http.StatusCreated, toManagedDeviceResponse(req))
}

func (s *Server) handleUpdateManagedDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Device ID is required")
		return
	}

	var req managedDevice
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.IP) == "" {
		writeError(w, http.StatusBadRequest, "Name and IP are required")
		return
	}

	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	index := indexManagedDevice(devices, id)
	if index < 0 {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	current := devices[index]
	current.Name = strings.TrimSpace(req.Name)
	current.IP = strings.TrimSpace(req.IP)
	current.Username = strings.TrimSpace(req.Username)
	current.Port = normalizedDevicePort(req.Port)
	current.Model = strings.TrimSpace(req.Model)
	current.Serial = strings.TrimSpace(req.Serial)
	current.Source = firstNonEmpty(req.Source, current.Source, "manual")
	current.UpdatedAt = time.Now()
	if req.Password != "" {
		current.Password = req.Password
	}

	if req.IsDefault {
		clearDefaultDevice(devices)
		current.IsDefault = true
	}

	devices[index] = current
	if err := s.persistManagedDevices(r.Context(), devices); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update device")
		return
	}

	if current.IsDefault {
		s.applyDefaultDevice(r.Context(), current)
	}

	writeJSON(w, http.StatusOK, toManagedDeviceResponse(current))
}

func (s *Server) handleDeleteManagedDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Device ID is required")
		return
	}

	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	index := indexManagedDevice(devices, id)
	if index < 0 {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	wasDefault := devices[index].IsDefault
	devices = append(devices[:index], devices[index+1:]...)
	if wasDefault && len(devices) > 0 {
		devices[0].IsDefault = true
		s.applyDefaultDevice(r.Context(), devices[0])
	}

	if err := s.persistManagedDevices(r.Context(), devices); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	if len(devices) == 0 {
		s.clearDefaultDeviceConfig(r.Context())
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetManagedDeviceDefault(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Device ID is required")
		return
	}

	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	index := indexManagedDevice(devices, id)
	if index < 0 {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	clearDefaultDevice(devices)
	devices[index].IsDefault = true
	devices[index].UpdatedAt = time.Now()

	if err := s.persistManagedDevices(r.Context(), devices); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update device")
		return
	}

	s.applyDefaultDevice(r.Context(), devices[index])
	writeJSON(w, http.StatusOK, toManagedDeviceResponse(devices[index]))
}

func (s *Server) loadManagedDevices(ctx context.Context) ([]managedDevice, error) {
	raw, err := s.Store.GetConfigValue(ctx, managedDevicesConfigKey)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return []managedDevice{}, nil
	}

	var devices []managedDevice
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		return []managedDevice{}, nil
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].IsDefault != devices[j].IsDefault {
			return devices[i].IsDefault
		}
		return strings.ToLower(devices[i].Name) < strings.ToLower(devices[j].Name)
	})

	return devices, nil
}

func (s *Server) persistManagedDevices(ctx context.Context, devices []managedDevice) error {
	payload, err := json.Marshal(devices)
	if err != nil {
		return err
	}
	return s.Store.SetConfigValue(ctx, managedDevicesConfigKey, string(payload))
}

func (s *Server) applyDefaultDevice(ctx context.Context, device managedDevice) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Config.HikvisionIP = device.IP
	s.Config.HikvisionUsername = device.Username
	if device.Password != "" {
		s.Config.HikvisionPassword = device.Password
	}

	_ = s.Store.SetMultipleConfigValues(ctx, map[string]string{
		"hikvision_ip":       device.IP,
		"hikvision_username": device.Username,
		"hikvision_password": device.Password,
	})
}

func (s *Server) clearDefaultDeviceConfig(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Config.HikvisionIP = ""
	s.Config.HikvisionUsername = ""
	s.Config.HikvisionPassword = ""

	_ = s.Store.SetMultipleConfigValues(ctx, map[string]string{
		"hikvision_ip":       "",
		"hikvision_username": "",
		"hikvision_password": "",
	})
}

func indexManagedDevice(devices []managedDevice, id string) int {
	for i, device := range devices {
		if device.ID == id {
			return i
		}
	}
	return -1
}

func clearDefaultDevice(devices []managedDevice) {
	for i := range devices {
		devices[i].IsDefault = false
	}
}

func normalizedDevicePort(port int) int {
	if port <= 0 {
		return 80
	}
	return port
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func toManagedDeviceResponses(devices []managedDevice) []managedDeviceResponse {
	response := make([]managedDeviceResponse, len(devices))
	for i, device := range devices {
		response[i] = toManagedDeviceResponse(device)
	}
	return response
}

func toManagedDeviceResponse(device managedDevice) managedDeviceResponse {
	return managedDeviceResponse{
		ID:          device.ID,
		Name:        device.Name,
		IP:          device.IP,
		Username:    device.Username,
		Port:        device.Port,
		Model:       device.Model,
		Serial:      device.Serial,
		Source:      device.Source,
		IsDefault:   device.IsDefault,
		HasPassword: device.Password != "",
		UpdatedAt:   device.UpdatedAt,
	}
}
