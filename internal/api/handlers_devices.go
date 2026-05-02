package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"ponches/internal/config"
	"ponches/internal/discovery"
	"ponches/internal/employees"
	"ponches/internal/hikvision"
	"ponches/internal/store"
	"strconv"

	"github.com/rs/zerolog/log"
)

const managedDevicesConfigKey = "managed_devices"

type ManagedDevice struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	IP             string    `json:"ip"`
	Username       string    `json:"username"`
	Password       string    `json:"password,omitempty"`
	Port           int       `json:"port"`
	Model          string    `json:"model,omitempty"`
	Serial         string    `json:"serial,omitempty"`
	Source         string    `json:"source,omitempty"`
	IsDefault      bool      `json:"isDefault"`
	TimezoneOffset string    `json:"timezoneOffset,omitempty"` // e.g. "+08:00"
	UpdatedAt      time.Time `json:"updatedAt"`
}

type managedDeviceResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	IP             string    `json:"ip"`
	Username       string    `json:"username"`
	Port           int       `json:"port"`
	Model          string    `json:"model,omitempty"`
	Serial         string    `json:"serial,omitempty"`
	Source         string    `json:"source,omitempty"`
	IsDefault      bool      `json:"isDefault"`
	HasPassword    bool      `json:"hasPassword"`
	IsOnline       bool      `json:"isOnline"`
	Error          string    `json:"error,omitempty"`
	TimezoneOffset string    `json:"timezoneOffset,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type networkConfigResponse struct {
	Range          string `json:"range"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	MaxConcurrency int    `json:"maxConcurrency"`
	EnableAutoScan bool   `json:"enableAutoScan"`
}

func managedDeviceLogID(device *ManagedDevice) string {
	if device == nil {
		return "unknown"
	}
	if strings.TrimSpace(device.ID) != "" {
		return device.ID
	}
	return firstNonEmpty(device.Name, device.IP, "unknown")
}

func (s *Server) saveDeviceOperationLog(ctx context.Context, deviceID, operation, level, message string) {
	if strings.TrimSpace(deviceID) == "" {
		deviceID = "unknown"
	}
	if strings.TrimSpace(level) == "" {
		level = "info"
	}
	if strings.TrimSpace(message) == "" {
		message = "OK"
	}
	if err := s.Store.SaveDeviceLog(ctx, &store.DeviceLog{
		DeviceID:     deviceID,
		Operation:    operation,
		ErrorMessage: message,
		Level:        level,
	}); err != nil {
		log.Warn().Err(err).Str("deviceId", deviceID).Str("operation", operation).Msg("Failed to persist device operation log")
	}
}

type discoveryResult struct {
	IP         string `json:"ip"`
	Model      string `json:"model,omitempty"`
	Serial     string `json:"serial,omitempty"`
	DeviceType string `json:"deviceType,omitempty"`
	Source     string `json:"source"` // "sadp" | "tcp"
}

func (s *Server) handleListManagedDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	// Concurrently check status for all devices
	responses := make([]managedDeviceResponse, len(devices))
	var wg sync.WaitGroup
	for i, d := range devices {
		wg.Add(1)
		go func(idx int, device ManagedDevice) {
			defer wg.Done()
			res := toManagedDeviceResponse(device)

			// Fast connectivity check
			client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
			ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second) // 1s timeout for list check
			defer cancel()

			info, err := client.GetDeviceInfo(ctx)
			res.IsOnline = err == nil && info != nil
			if err != nil {
				res.Error = err.Error()
			}
			responses[idx] = res
		}(i, d)
	}
	wg.Wait()

	writeJSON(w, http.StatusOK, responses)
}

func (s *Server) handleCreateManagedDevice(w http.ResponseWriter, r *http.Request) {
	var req ManagedDevice
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
	req.TimezoneOffset = firstNonEmpty(req.TimezoneOffset, "+08:00")
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

	var req ManagedDevice
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
	current.TimezoneOffset = firstNonEmpty(req.TimezoneOffset, current.TimezoneOffset, "+08:00")
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

func (s *Server) handleDiscoverDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	netCfg, err := s.loadNetworkConfig(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load network config")
		return
	}

	results := make(map[string]discoveryResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 1. SADP Discovery
	wg.Add(1)
	go func() {
		defer wg.Done()
		sadpDevices, err := discovery.Discover(netCfg.TimeoutSeconds)
		if err == nil {
			mu.Lock()
			for _, d := range sadpDevices {
				results[d.IPv4Address] = discoveryResult{
					IP:         d.IPv4Address,
					Model:      d.DeviceDesc,
					Serial:     d.DeviceSN,
					DeviceType: d.DeviceType,
					Source:     "sadp",
				}
			}
			mu.Unlock()
		}
	}()

	// 2. TCP Port Scan (Fallback/Complement)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Common Hikvision ISAPI ports
		ports := []int{80, 8000, 8080, 443}
		foundIPs, err := discovery.ScanPorts(ctx, netCfg.Range, ports, time.Duration(netCfg.TimeoutSeconds)*time.Second, netCfg.MaxConcurrency)
		if err == nil {
			mu.Lock()
			for _, ip := range foundIPs {
				if _, exists := results[ip]; !exists {
					results[ip] = discoveryResult{
						IP:     ip,
						Source: "tcp",
					}
				}
			}
			mu.Unlock()
		}
	}()

	wg.Wait()

	finalResults := make([]discoveryResult, 0, len(results))
	for _, res := range results {
		finalResults = append(finalResults, res)
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].IP < finalResults[j].IP
	})

	writeJSON(w, http.StatusOK, finalResults)
}

func (s *Server) handleRefreshDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	type refreshStatus struct {
		ID     string `json:"id"`
		IP     string `json:"ip"`
		Online bool   `json:"online"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]refreshStatus, len(devices))
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	for i, device := range devices {
		wg.Add(1)
		go func(idx int, d ManagedDevice) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client := hikvision.NewClient(d.IP, d.Port, d.Username, d.Password)
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			info, err := client.GetDeviceInfo(ctx)
			status := refreshStatus{
				ID:     d.ID,
				IP:     d.IP,
				Online: err == nil,
			}
			if err != nil {
				status.Error = err.Error()
			} else if info != nil {
				// Update model/serial if changed
				mu.Lock()
				if devices[idx].Model != info.Model || devices[idx].Serial != info.SerialNumber {
					devices[idx].Model = info.Model
					devices[idx].Serial = info.SerialNumber
					devices[idx].UpdatedAt = time.Now()
				}
				mu.Unlock()
			}
			results[idx] = status
		}(i, device)
	}

	wg.Wait()

	// Persist updates if models/serials were updated
	_ = s.persistManagedDevices(r.Context(), devices)

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleGetNetworkConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadNetworkConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load network config")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleUpdateNetworkConfig(w http.ResponseWriter, r *http.Request) {
	var req networkConfigResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]string{
		config.ConfigKeyNetworkDiscoveryRange:          req.Range,
		config.ConfigKeyNetworkDiscoveryTimeout:        strconv.Itoa(req.TimeoutSeconds),
		config.ConfigKeyNetworkDiscoveryMaxConcurrency: strconv.Itoa(req.MaxConcurrency),
		config.ConfigKeyNetworkDiscoveryEnableAutoScan: strconv.FormatBool(req.EnableAutoScan),
	}

	if err := s.Store.SetMultipleConfigValues(r.Context(), updates); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	writeJSON(w, http.StatusOK, req)
}

func (s *Server) handleGetDeviceLogs(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs, err := s.Store.GetDeviceLogs(r.Context(), deviceID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch device logs")
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

// handleSyncOneEmployeeToDevice grants access to a single employee on a specific terminal.
// Route: POST /api/devices/configured/{id}/sync-one/{employeeNo}
// Use id="default" to target the default device.
func (s *Server) handleSyncOneEmployeeToDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	employeeNo := chi.URLParam(r, "employeeNo")
	ctx := r.Context()

	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "employeeNo is required")
		return
	}

	// 1. Look up the employee
	emp, err := s.Store.GetEmployeeByNo(ctx, employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch employee %q", employeeNo))
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Employee %q not found", employeeNo))
		return
	}

	if deviceID == "" || deviceID == "default" || deviceID == "all" {
		summary := s.syncEmployeeToAllDevices(ctx, emp, false, "PushEmployee")
		if summary.DevicesTotal == 0 {
			writeError(w, http.StatusServiceUnavailable, "No hay dispositivos configurados")
			return
		}
		if summary.DevicesSuccess == 0 {
			writeError(w, http.StatusBadGateway, "No se pudo registrar el empleado en los dispositivos configurados")
			return
		}

		emp.Status = "Active"
		s.Store.UpdateEmployee(ctx, emp)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    fmt.Sprintf("Empleado %s %s registrado en %d dispositivo(s)", emp.FirstName, emp.LastName, summary.DevicesSuccess),
			"employeeNo": emp.EmployeeNo,
			"sync":       summary,
		})
		return
	}

	// 2. Resolve a specific target device
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	idx := indexManagedDevice(devices, deviceID)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}
	target := &devices[idx]

	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	if err := client.CreateUser(ctx, emp); err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Enviar empleado - Dispositivo: %s, Empleado: %s, Error: %v", target.IP, emp.EmployeeNo, err))
		s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "PushEmployee", "error", fmt.Sprintf("Empleado %s: %v", emp.EmployeeNo, err))
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to register employee on device: %v", err))
		return
	}

	photoSync := s.syncEmployeePhotoWithDevice(ctx, client, emp, target.ID, false)

	// Update local DB status to reflect success
	emp.Status = "Active"
	s.Store.UpdateEmployee(ctx, emp)
	s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "PushEmployee", "info", fmt.Sprintf("Empleado %s enviado. Foto: %s", emp.EmployeeNo, photoSync))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    fmt.Sprintf("Empleado %s %s registrado en %s", emp.FirstName, emp.LastName, target.Name),
		"employeeNo": emp.EmployeeNo,
		"device":     target.IP,
		"photoSync":  photoSync,
	})
}

// handleRevokeEmployeeFromDevice removes a single employee from a device WITHOUT deleting from DB.
// Route: DELETE /api/devices/configured/{id}/sync-one/{employeeNo}
// Use id="default" to target the default device.
func (s *Server) handleRevokeEmployeeFromDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	employeeNo := chi.URLParam(r, "employeeNo")
	ctx := r.Context()

	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "employeeNo is required")
		return
	}

	// 1. Verify the employee exists in DB (just for a good error message)
	emp, err := s.Store.GetEmployeeByNo(ctx, employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch employee %q", employeeNo))
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Employee %q not found", employeeNo))
		return
	}

	if deviceID == "" || deviceID == "default" || deviceID == "all" {
		devices, err := s.loadSyncDevices(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load devices")
			return
		}
		if len(devices) == 0 {
			writeError(w, http.StatusServiceUnavailable, "No hay dispositivos configurados")
			return
		}

		success := 0
		var errs []string
		for _, device := range devices {
			client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
			if err := client.DeleteUser(ctx, employeeNo); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", firstNonEmpty(device.Name, device.IP), err))
				s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "RevokeEmployee", "error", fmt.Sprintf("Empleado %s: %v", employeeNo, err))
				continue
			}
			success++
			s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "RevokeEmployee", "info", fmt.Sprintf("Empleado %s revocado correctamente", employeeNo))
		}

		if success == 0 {
			writeError(w, http.StatusBadGateway, "No se pudo revocar el empleado en los dispositivos configurados")
			return
		}

		emp.Status = "Inactive"
		s.Store.UpdateEmployee(ctx, emp)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    fmt.Sprintf("Acceso de %s %s revocado en %d dispositivo(s)", emp.FirstName, emp.LastName, success),
			"employeeNo": emp.EmployeeNo,
			"errors":     errs,
		})
		return
	}

	// 2. Resolve the target device
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}
	idx := indexManagedDevice(devices, deviceID)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}
	target := &devices[idx]

	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	if err := client.DeleteUser(ctx, employeeNo); err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Revocar acceso - Dispositivo: %s, Empleado: %s, Error: %v", target.IP, employeeNo, err))
		s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "RevokeEmployee", "error", fmt.Sprintf("Empleado %s: %v", employeeNo, err))
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to revoke employee from device: %v", err))
		return
	}

	emp.Status = "Inactive"
	s.Store.UpdateEmployee(ctx, emp)

	s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "RevokeEmployee", "info", fmt.Sprintf("Empleado %s revocado correctamente", employeeNo))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    fmt.Sprintf("Acceso de %s %s revocado en %s", emp.FirstName, emp.LastName, target.Name),
		"employeeNo": emp.EmployeeNo,
		"device":     target.IP,
	})
}

func (s *Server) handleSyncDeviceTime(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	ctx := r.Context()

	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	var target *ManagedDevice
	if deviceID == "default" {
		for i := range devices {
			if devices[i].IsDefault {
				target = &devices[i]
				break
			}
		}
	} else {
		idx := indexManagedDevice(devices, deviceID)
		if idx >= 0 {
			target = &devices[idx]
		}
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	if err := client.SyncTime(ctx, time.Now()); err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Sincronizar hora - Dispositivo: %s, Error: %v", target.IP, err))
		s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "SyncTime", "error", err.Error())
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to sync device time: %v", err))
		return
	}

	s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "SyncTime", "info", fmt.Sprintf("Hora sincronizada en %s", target.Name))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Hora sincronizada correctamente en %s", target.Name),
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	})
}

func (s *Server) handleSyncEmployeesToDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	// 1. Get all employees
	emps, err := s.Store.ListEmployees(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load employees")
		return
	}

	// filter active ones
	var activeEmps []*employees.Employee
	for i := range emps {
		if strings.EqualFold(emps[i].Status, "active") {
			activeEmps = append(activeEmps, emps[i])
		}
	}

	if len(activeEmps) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "No active employees to sync",
			"count":   0,
		})
		return
	}

	// 2. Sync to all devices when using the default/global action.
	if id == "" || id == "default" || id == "all" {
		devices, err := s.loadSyncDevices(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load devices")
			return
		}
		if len(devices) == 0 {
			writeError(w, http.StatusServiceUnavailable, "No hay dispositivos configurados")
			return
		}

		result := map[string]interface{}{
			"devices":     len(devices),
			"employees":   len(activeEmps),
			"byEmployee":  []employeeSyncSummary{},
			"photoTotals": map[string]int{},
		}

		successEmployees := 0
		allErrors := []string{}
		byEmployee := make([]employeeSyncSummary, 0, len(activeEmps))
		photoTotals := map[string]int{}

		for _, emp := range activeEmps {
			summary := s.syncEmployeeToAllDevices(ctx, emp, false, "Sync")
			byEmployee = append(byEmployee, summary)
			if summary.DevicesSuccess > 0 {
				successEmployees++
			}
			allErrors = append(allErrors, summary.Errors...)
			for key, value := range summary.PhotoSync {
				photoTotals[key] += value
			}
		}

		result["byEmployee"] = byEmployee
		result["photoTotals"] = photoTotals
		result["errors"] = allErrors

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":        fmt.Sprintf("Sincronizados %d de %d empleados con %d dispositivo(s)", successEmployees, len(activeEmps), len(devices)),
			"count":          successEmployees,
			"devicesSynced":  len(devices),
			"employeesTotal": len(activeEmps),
			"photoSync":      photoTotals,
			"errors":         allErrors,
		})
		return
	}

	// 3. Sync to one specific device
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}
	idx := indexManagedDevice(devices, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}
	target := &devices[idx]

	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	err = client.CreateUsers(ctx, activeEmps)
	if err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Sincronización masiva - Dispositivo: %s, Error: %v", target.IP, err))
		s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "Sync", "error", err.Error())
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Sync failed: %v", err))
		return
	}

	photoStats := map[string]int{}
	for _, emp := range activeEmps {
		photoStats[s.syncEmployeePhotoWithDevice(ctx, client, emp, target.ID, false)]++
	}
	s.saveDeviceOperationLog(ctx, managedDeviceLogID(target), "Sync", "info", fmt.Sprintf("Sincronizados %d empleados. Fotos: %+v", len(activeEmps), photoStats))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Successfully synced employees to device",
		"count":     len(activeEmps),
		"device":    target.IP,
		"photoSync": photoStats,
	})
}

func (s *Server) syncEmployeePhotoWithDevice(ctx context.Context, client *hikvision.Client, emp *employees.Employee, deviceID string, photoRemoved bool) string {
	if emp == nil || emp.EmployeeNo == "" {
		return "missing"
	}

	if photoRemoved {
		if err := client.DeleteFace(ctx, emp.EmployeeNo); err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			if recreated, recreateErr := s.recreateUserWithoutPhoto(ctx, client, emp.EmployeeNo); recreateErr == nil && recreated {
				return "recreated_without_photo"
			}
			log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Str("device", deviceID).Msg("Failed to delete employee photo from Hikvision during sync")
			return "delete_failed"
		}
		return "removed"
	}

	photoData := emp.PhotoData
	if len(photoData) == 0 && emp.PhotoURL != "" {
		photoPath := filepath.Join("web", filepath.FromSlash(strings.TrimPrefix(emp.PhotoURL, "/")))
		fileData, err := os.ReadFile(photoPath)
		if err != nil {
			log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Str("path", photoPath).Msg("Failed to read local employee photo during sync")
		} else {
			photoData = fileData
			emp.PhotoData = fileData
			if err := s.Store.UpdateEmployeePhoto(ctx, emp.EmployeeNo, fileData); err != nil {
				log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Msg("Failed to backfill photo_data from local file during sync")
			}
		}
	}

	if len(photoData) > 0 {
		if err := client.UploadPhotoToHikvision(ctx, emp.EmployeeNo, photoData); err != nil {
			log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Str("device", deviceID).Msg("Failed to upload local photo to Hikvision during sync")
			return "failed"
		}
		return "uploaded"
	}

	if err := client.DeleteFace(ctx, emp.EmployeeNo); err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Str("device", deviceID).Msg("Failed to delete remote photo while enforcing project state")
		return "delete_failed"
	}
	return "removed"
}

func (s *Server) handleImportEmployeesFromDevices(w http.ResponseWriter, r *http.Request) {
	result, err := s.importEmployeesFromDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) loadNetworkConfig(ctx context.Context) (networkConfigResponse, error) {
	values, err := s.Store.GetAllConfig(ctx)
	if err != nil {
		return networkConfigResponse{}, err
	}

	defaults := config.DefaultConfig()
	getVal := func(key string) string {
		if v, ok := values[key]; ok && v != "" {
			return v
		}
		return defaults[key]
	}

	timeout, _ := strconv.Atoi(getVal(config.ConfigKeyNetworkDiscoveryTimeout))
	concurrency, _ := strconv.Atoi(getVal(config.ConfigKeyNetworkDiscoveryMaxConcurrency))
	autoScan, _ := strconv.ParseBool(getVal(config.ConfigKeyNetworkDiscoveryEnableAutoScan))

	return networkConfigResponse{
		Range:          getVal(config.ConfigKeyNetworkDiscoveryRange),
		TimeoutSeconds: timeout,
		MaxConcurrency: concurrency,
		EnableAutoScan: autoScan,
	}, nil
}

func (s *Server) loadManagedDevices(ctx context.Context) ([]ManagedDevice, error) {
	raw, err := s.Store.GetConfigValue(ctx, managedDevicesConfigKey)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return []ManagedDevice{}, nil
	}

	var devices []ManagedDevice
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		return []ManagedDevice{}, nil
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].IsDefault != devices[j].IsDefault {
			return devices[i].IsDefault
		}
		return strings.ToLower(devices[i].Name) < strings.ToLower(devices[j].Name)
	})

	return devices, nil
}

func (s *Server) persistManagedDevices(ctx context.Context, devices []ManagedDevice) error {
	payload, err := json.Marshal(devices)
	if err != nil {
		return err
	}
	return s.Store.SetConfigValue(ctx, managedDevicesConfigKey, string(payload))
}

func (s *Server) applyDefaultDevice(ctx context.Context, device ManagedDevice) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Config.HikvisionIP = device.IP
	s.Config.HikvisionPort = device.Port
	s.Config.HikvisionUsername = device.Username
	if device.Password != "" {
		s.Config.HikvisionPassword = device.Password
	}

	_ = s.Store.SetMultipleConfigValues(ctx, map[string]string{
		"hikvision_ip":       device.IP,
		"hikvision_port":     strconv.Itoa(device.Port),
		"hikvision_username": device.Username,
		"hikvision_password": device.Password,
	})
}

func (s *Server) clearDefaultDeviceConfig(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Config.HikvisionIP = ""
	s.Config.HikvisionPort = 80
	s.Config.HikvisionUsername = ""
	s.Config.HikvisionPassword = ""

	_ = s.Store.SetMultipleConfigValues(ctx, map[string]string{
		"hikvision_ip":       "",
		"hikvision_port":     "80",
		"hikvision_username": "",
		"hikvision_password": "",
	})
}

func indexManagedDevice(devices []ManagedDevice, id string) int {
	for i, device := range devices {
		if device.ID == id {
			return i
		}
	}
	return -1
}

func clearDefaultDevice(devices []ManagedDevice) {
	for i := range devices {
		devices[i].IsDefault = false
	}
}

func (s *Server) handleReadRecentEvents(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var start, end time.Time
	var err error

	if fromStr != "" && toStr != "" {
		start, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			start = time.Now().Add(-24 * time.Hour)
		}
		end, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			end = time.Now().Add(1 * time.Hour)
		} else {
			// Include the full end day
			end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	} else {
		start = time.Now().Add(-24 * time.Hour)
		end = time.Now().Add(1 * time.Hour)
	}

	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	totalEvents := 0
	for _, d := range devices {
		client := hikvision.NewClient(d.IP, d.Port, d.Username, d.Password)
		client.TimezoneOffset = d.TimezoneOffset
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)

		events, err := client.GetEventsInRange(ctx, start, end)
		cancel()
		if err != nil {
			log.Warn().Str("device", d.IP).Err(err).Msg("Failed to read events from device")
			s.saveDeviceOperationLog(r.Context(), managedDeviceLogID(&d), "ReadEvents", "error", err.Error())
			continue
		}
		s.saveDeviceOperationLog(r.Context(), managedDeviceLogID(&d), "ReadEvents", "info", fmt.Sprintf("Lectura completada. Eventos obtenidos: %d", len(events)))

		log.Info().Str("device", d.IP).Int("found", len(events)).Msg("Syncing events from device")

		for _, event := range events {
			// Save to database
			storeEvent := &store.AttendanceEvent{
				DeviceID:   event.DeviceID,
				EmployeeNo: event.EmployeeNo,
				Timestamp:  event.Timestamp,
				Type:       event.EventType,
			}
			err := s.Store.SaveEvent(r.Context(), storeEvent)
			if err != nil {
				log.Error().Err(err).Str("employee", event.EmployeeNo).Msg("Failed to save event to DB")
			} else {
				totalEvents++
			}

			// Get employee name for broadcast
			emp, _ := s.Store.GetEmployeeByNo(r.Context(), event.EmployeeNo)
			employeeName := event.EmployeeNo
			if emp != nil {
				employeeName = emp.FirstName + " " + emp.LastName
			}

			// Broadcast to WebSocket clients
			s.Hub.BroadcastAttendanceEvent(event.EmployeeNo, employeeName, event.DeviceID, event.Timestamp)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"eventsRead": totalEvents,
	})
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

func toManagedDeviceResponses(devices []ManagedDevice) []managedDeviceResponse {
	response := make([]managedDeviceResponse, len(devices))
	for i, device := range devices {
		response[i] = toManagedDeviceResponse(device)
	}
	return response
}

func toManagedDeviceResponse(device ManagedDevice) managedDeviceResponse {
	return managedDeviceResponse{
		ID:             device.ID,
		Name:           device.Name,
		IP:             device.IP,
		Username:       device.Username,
		Port:           device.Port,
		Model:          device.Model,
		Serial:         device.Serial,
		Source:         device.Source,
		IsDefault:      device.IsDefault,
		HasPassword:    device.Password != "",
		TimezoneOffset: device.TimezoneOffset,
		UpdatedAt:      device.UpdatedAt,
	}
}
func (s *Server) logToFile(message string) {
	f, err := os.OpenFile("error_logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("Could not open error_logs.txt")
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if _, err := f.WriteString(fmt.Sprintf("%s\t%s\n", timestamp, message)); err != nil {
		log.Error().Err(err).Msg("Could not write to error_logs.txt")
	}
}

func (s *Server) handleSetupDeviceAlarmHost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	devices, err := s.loadManagedDevices(r.Context())
	if err != nil {
		http.Error(w, "Failed to load devices", http.StatusInternalServerError)
		return
	}

	var target *ManagedDevice
	for _, d := range devices {
		if d.ID == id {
			target = &d
			break
		}
	}

	if target == nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// We need the server's IP. In a real scenario, this would be in config.
	// For now, we'll try to determine it or use a default.
	serverIP := s.Config.ServerIP
	if serverIP == "" {
		serverIP = "192.168.1.100" // Default/Example
	}
	serverPort := s.Config.Port
	if serverPort == 0 {
		serverPort = 8080
	}

	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	err = client.SetupAlarmHost(r.Context(), 1, serverIP, serverPort, "/api/callback/hikvision")
	if err != nil {
		s.saveDeviceOperationLog(r.Context(), target.ID, "setup_alarm_host", "error", err.Error())
		http.Error(w, fmt.Sprintf("Failed to setup alarm host: %v", err), http.StatusInternalServerError)
		return
	}

	s.saveDeviceOperationLog(r.Context(), target.ID, "setup_alarm_host", "info", "Successfully configured alarm host")
	s.LogAudit(r.Context(), r, "SETUP_ALARM_HOST", target.IP, nil)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
