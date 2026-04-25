package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	IsOnline    bool      `json:"isOnline"`
	Error       string    `json:"error,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type networkConfigResponse struct {
	Range          string `json:"range"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	MaxConcurrency int    `json:"maxConcurrency"`
	EnableAutoScan bool   `json:"enableAutoScan"`
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
		config.ConfigKeyNetworkDiscoveryEnableAutoScan:  strconv.FormatBool(req.EnableAutoScan),
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

	// 1. Resolve the target device
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	var target *ManagedDevice
	if deviceID != "" && deviceID != "default" {
		idx := indexManagedDevice(devices, deviceID)
		if idx >= 0 {
			target = &devices[idx]
		}
	} else {
		for i := range devices {
			if devices[i].IsDefault {
				target = &devices[i]
				break
			}
		}
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "Device not found or no default device configured")
		return
	}

	// 2. Look up the employee
	emp, err := s.Store.GetEmployeeByNo(ctx, employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch employee %q", employeeNo))
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Employee %q not found", employeeNo))
		return
	}

	// 3. Push to device
	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	if err := client.CreateUser(ctx, emp); err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Enviar empleado - Dispositivo: %s, Empleado: %s, Error: %v", target.IP, emp.EmployeeNo, err))
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to register employee on device: %v", err))
		return
	}

	// Update local DB status to reflect success
	emp.Status = "Active"
	s.Store.UpdateEmployee(ctx, emp)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    fmt.Sprintf("Empleado %s %s registrado en %s", emp.FirstName, emp.LastName, target.Name),
		"employeeNo": emp.EmployeeNo,
		"device":     target.IP,
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

	// 1. Resolve the target device
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	var target *ManagedDevice
	if deviceID != "" && deviceID != "default" {
		idx := indexManagedDevice(devices, deviceID)
		if idx >= 0 {
			target = &devices[idx]
		}
	} else {
		for i := range devices {
			if devices[i].IsDefault {
				target = &devices[i]
				break
			}
		}
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "Device not found or no default device configured")
		return
	}

	// 2. Verify the employee exists in DB (just for a good error message)
	emp, err := s.Store.GetEmployeeByNo(ctx, employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch employee %q", employeeNo))
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Employee %q not found", employeeNo))
		return
	}

	// 3. Remove from device only
	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	if err := client.DeleteUser(ctx, employeeNo); err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Revocar acceso - Dispositivo: %s, Empleado: %s, Error: %v", target.IP, employeeNo, err))
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to revoke employee from device: %v", err))
		return
	}

	// Update local DB status to reflect success
	emp.Status = "Inactive"
	s.Store.UpdateEmployee(ctx, emp)

	s.Store.SaveDeviceLog(ctx, &store.DeviceLog{
		DeviceID:  target.ID,
		Operation: "RevokeEmployee",
		Level:     "info",
	})

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
			if devices[i].IsDefault { target = &devices[i]; break }
		}
	} else {
		idx := indexManagedDevice(devices, deviceID)
		if idx >= 0 { target = &devices[idx] }
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	if err := client.SyncTime(ctx, time.Now()); err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Sincronizar hora - Dispositivo: %s, Error: %v", target.IP, err))
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to sync device time: %v", err))
		return
	}

	s.Store.SaveDeviceLog(ctx, &store.DeviceLog{
		DeviceID:  target.ID,
		Operation: "SyncTime",
		Level:     "info",
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Hora sincronizada correctamente en %s", target.Name),
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	})
}

func (s *Server) handleSyncEmployeesToDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	// 1. Get the device
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load devices")
		return
	}

	var target *ManagedDevice
	if id != "" && id != "default" {
		idx := indexManagedDevice(devices, id)
		if idx >= 0 {
			target = &devices[idx]
		}
	} else {
		// Find default
		for i := range devices {
			if devices[i].IsDefault {
				target = &devices[i]
				break
			}
		}
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "Device not found or no default device set")
		return
	}

	// 2. Get all employees
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

	// 3. Sync to device
	client := hikvision.NewClient(target.IP, target.Port, target.Username, target.Password)
	
	// Prepare batch (The ISAPI Record endpoint handles multiple users in One XML if structured correctly, 
	// but our current CreateUser in users.go takes one. We have a private upsertUsers that takes variadic.)
	// I'll use the private upsertUsers if I make it public or just call CreateUser in a loop (less efficient but safer).
	// Actually, I'll update users.go to export UpsertUsers for batching.
	
	// For now, let's use the exported CreateUser (which I'll update to handle batching or use a loop).
	// Wait, I wrote users.go earlier, let me check it.
	
	// CreateUser calls upsertUsers(ctx, emp)
	// I should probably export CreateUsers(ctx, emps []*employees.Employee)
	
	err = client.CreateUsers(ctx, activeEmps)
	if err != nil {
		s.logToFile(fmt.Sprintf("ERROR: Sincronización masiva - Dispositivo: %s, Error: %v", target.IP, err))
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Sync failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Successfully synced employees to device",
		"count":   len(activeEmps),
		"device":  target.IP,
	})
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
		EnableAutoScan:  autoScan,
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
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		
		events, err := client.GetEventsInRange(ctx, start, end)
		cancel()
		if err != nil {
			log.Warn().Str("device", d.IP).Err(err).Msg("Failed to read events from device")
			continue
		}

		for _, event := range events {
			// Save to database
			storeEvent := &store.AttendanceEvent{
				DeviceID:   event.DeviceID,
				EmployeeNo: event.EmployeeNo,
				Timestamp:  event.Timestamp,
				Type:       event.EventType,
			}
			s.Store.SaveEvent(r.Context(), storeEvent)

			// Get employee name for broadcast
			emp, _ := s.Store.GetEmployeeByNo(r.Context(), event.EmployeeNo)
			employeeName := event.EmployeeNo
			if emp != nil {
				employeeName = emp.FirstName + " " + emp.LastName
			}

			// Broadcast to WebSocket clients
			s.Hub.BroadcastAttendanceEvent(event.EmployeeNo, employeeName, event.DeviceID, event.Timestamp)
			totalEvents++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
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
