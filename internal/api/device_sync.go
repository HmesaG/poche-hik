package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"ponches/internal/employees"
	"ponches/internal/hikvision"
)

type employeeSyncSummary struct {
	DevicesTotal   int               `json:"devicesTotal"`
	DevicesSuccess int               `json:"devicesSuccess"`
	DevicesFailed  int               `json:"devicesFailed"`
	PhotoSync      map[string]int    `json:"photoSync"`
	Errors         []string          `json:"errors,omitempty"`
	ByDevice       []deviceSyncEntry `json:"byDevice,omitempty"`
}

type deviceSyncEntry struct {
	DeviceID string `json:"deviceId"`
	Device   string `json:"device"`
	Status   string `json:"status"`
	Photo    string `json:"photo"`
	Error    string `json:"error,omitempty"`
}

func (s *Server) loadSyncDevices(ctx context.Context) ([]ManagedDevice, error) {
	devices, err := s.loadManagedDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devices) > 0 {
		return devices, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if strings.TrimSpace(s.Config.HikvisionIP) == "" || strings.TrimSpace(s.Config.HikvisionUsername) == "" {
		return []ManagedDevice{}, nil
	}

	return []ManagedDevice{
		{
			ID:             "legacy-default",
			Name:           firstNonEmpty(s.Config.HikvisionIP, "Legacy Device"),
			IP:             s.Config.HikvisionIP,
			Port:           normalizedDevicePort(s.Config.HikvisionPort),
			Username:       s.Config.HikvisionUsername,
			Password:       s.Config.HikvisionPassword,
			IsDefault:      true,
			TimezoneOffset: "+08:00",
			UpdatedAt:      time.Now(),
		},
	}, nil
}

func (s *Server) syncEmployeeToAllDevices(ctx context.Context, emp *employees.Employee, photoRemoved bool, operation string) employeeSyncSummary {
	summary := employeeSyncSummary{
		PhotoSync: map[string]int{},
	}
	if emp == nil || strings.TrimSpace(emp.EmployeeNo) == "" {
		summary.Errors = append(summary.Errors, "employeeNo is required")
		return summary
	}

	devices, err := s.loadSyncDevices(ctx)
	if err != nil {
		summary.Errors = append(summary.Errors, err.Error())
		return summary
	}
	summary.DevicesTotal = len(devices)
	if len(devices) == 0 {
		summary.Errors = append(summary.Errors, "no managed devices configured")
		return summary
	}

	for _, device := range devices {
		client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
		entry := deviceSyncEntry{
			DeviceID: managedDeviceLogID(&device),
			Device:   firstNonEmpty(device.Name, device.IP),
			Status:   "ok",
		}

		if err := client.CreateUser(ctx, emp); err != nil {
			entry.Status = "failed"
			entry.Error = err.Error()
			summary.DevicesFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", entry.Device, err))
			s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), operation, "error", fmt.Sprintf("Empleado %s: %v", emp.EmployeeNo, err))
			summary.ByDevice = append(summary.ByDevice, entry)
			continue
		}

		// Sync Card if available
		if strings.TrimSpace(emp.CardNo) != "" {
			card := hikvision.CardInfo{
				EmployeeNo: emp.EmployeeNo,
				CardNo:     emp.CardNo,
				CardType:   "normalCard",
			}
			// We attempt to create the card. If it fails, we log it but don't fail the whole sync
			// as the user record is already on the device.
			if err := client.CreateCard(ctx, card); err != nil {
				// If card already exists, we might need to delete it first, but for now we just log
				log.Warn().Err(err).Str("employeeNo", emp.EmployeeNo).Str("cardNo", emp.CardNo).Msg("Failed to sync card to device")
			}
		}

		entry.Photo = s.syncEmployeePhotoWithDevice(ctx, client, emp, managedDeviceLogID(&device), photoRemoved)
		summary.PhotoSync[entry.Photo]++
		if isPhotoSyncFailure(entry.Photo) {
			entry.Status = "failed"
			entry.Error = fmt.Sprintf("photo sync result: %s", entry.Photo)
			summary.DevicesFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %s", entry.Device, entry.Error))
			s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), operation, "warning", fmt.Sprintf("Empleado %s enviado, pero la foto fallo: %s", emp.EmployeeNo, entry.Photo))
		} else {
			summary.DevicesSuccess++
			s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), operation, "info", fmt.Sprintf("Empleado %s enviado. Foto: %s", emp.EmployeeNo, entry.Photo))
		}
		summary.ByDevice = append(summary.ByDevice, entry)
	}

	return summary
}

func (s *Server) removeEmployeePhotoFromAllDevices(ctx context.Context, employeeNo string) employeeSyncSummary {
	summary := employeeSyncSummary{
		PhotoSync: map[string]int{},
	}
	employeeNo = strings.TrimSpace(employeeNo)
	if employeeNo == "" {
		summary.Errors = append(summary.Errors, "employeeNo is required")
		return summary
	}

	devices, err := s.loadSyncDevices(ctx)
	if err != nil {
		summary.Errors = append(summary.Errors, err.Error())
		return summary
	}
	summary.DevicesTotal = len(devices)
	if len(devices) == 0 {
		summary.Errors = append(summary.Errors, "no managed devices configured")
		return summary
	}

	for _, device := range devices {
		entry := deviceSyncEntry{
			DeviceID: managedDeviceLogID(&device),
			Device:   firstNonEmpty(device.Name, device.IP),
			Status:   "ok",
			Photo:    "removed",
		}

		client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
		if err := client.DeleteFace(ctx, employeeNo); err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			if recreated, recreateErr := s.recreateUserWithoutPhoto(ctx, client, employeeNo); recreateErr == nil && recreated {
				summary.DevicesSuccess++
				entry.Photo = "recreated_without_photo"
				s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "DeleteFace", "info", fmt.Sprintf("Empleado %s recreado sin foto", employeeNo))
			} else {
				entry.Status = "failed"
				entry.Photo = "delete_failed"
				entry.Error = err.Error()
				summary.DevicesFailed++
				summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", entry.Device, err))
				s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "DeleteFace", "warning", fmt.Sprintf("Empleado %s: %v", employeeNo, err))
			}
		} else {
			summary.DevicesSuccess++
			s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "DeleteFace", "info", fmt.Sprintf("Foto eliminada para empleado %s", employeeNo))
		}

		summary.PhotoSync[entry.Photo]++
		summary.ByDevice = append(summary.ByDevice, entry)
	}

	return summary
}

func (s *Server) recreateUserWithoutPhoto(ctx context.Context, client *hikvision.Client, employeeNo string) (bool, error) {
	emp, err := s.Store.GetEmployeeByNo(ctx, employeeNo)
	if err != nil || emp == nil {
		if err == nil {
			err = fmt.Errorf("employee %s not found locally", employeeNo)
		}
		return false, err
	}

	if err := client.DeleteUser(ctx, employeeNo); err != nil {
		return false, err
	}
	if err := client.CreateUser(ctx, emp); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) importEmployeePhotoFromDevices(ctx context.Context, emp *employees.Employee, redistribute bool) ([]byte, *ManagedDevice, error) {
	if emp == nil || strings.TrimSpace(emp.EmployeeNo) == "" {
		return nil, nil, errors.New("employeeNo is required")
	}

	devices, err := s.loadSyncDevices(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(devices) == 0 {
		return nil, nil, errors.New("no managed devices configured")
	}

	var errs []string
	for _, device := range devices {
		client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
		imageData, err := client.DownloadFace(ctx, emp.EmployeeNo)
		if err != nil || len(imageData) == 0 {
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", firstNonEmpty(device.Name, device.IP), err))
			}
			continue
		}

		if err := s.storeEmployeePhoto(ctx, emp, imageData); err != nil {
			return nil, &device, err
		}
		s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "ImportFace", "info", fmt.Sprintf("Imagen importada para empleado %s", emp.EmployeeNo))

		if redistribute {
			emp.PhotoData = imageData
			s.syncEmployeeToAllDevices(ctx, emp, false, "PushEmployee")
		}

		return imageData, &device, nil
	}

	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("no se pudo importar la foto desde los dispositivos: %s", strings.Join(errs, "; "))
	}
	return nil, nil, errors.New("photo not found on any configured device")
}

func (s *Server) storeEmployeePhoto(ctx context.Context, emp *employees.Employee, imageData []byte) error {
	if emp == nil || strings.TrimSpace(emp.EmployeeNo) == "" {
		return errors.New("employeeNo is required")
	}
	if len(imageData) == 0 {
		return errors.New("empty photo data")
	}

	uploadDir := filepath.Join("web", "uploads", "employees")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s.jpg", emp.EmployeeNo)
	filePath := filepath.Join(uploadDir, fileName)
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return err
	}

	emp.PhotoURL = "/uploads/employees/" + fileName
	emp.PhotoData = imageData
	emp.FaceID = emp.EmployeeNo

	if err := s.Store.UpdateEmployee(ctx, emp); err != nil {
		return err
	}
	if err := s.Store.UpdateEmployeePhoto(ctx, emp.EmployeeNo, imageData); err != nil {
		return err
	}
	return nil
}

func (s *Server) importEmployeesFromDevices(ctx context.Context) (map[string]interface{}, error) {
	devices, err := s.loadSyncDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errors.New("no managed devices configured")
	}

	imported := 0
	existing := 0
	failed := 0
	var problems []string

	for _, device := range devices {
		client := hikvision.NewClient(device.IP, device.Port, device.Username, device.Password)
		users, err := client.GetUsers(ctx)
		if err != nil {
			failed++
			problems = append(problems, fmt.Sprintf("%s: %v", firstNonEmpty(device.Name, device.IP), err))
			s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "ImportUsers", "error", err.Error())
			continue
		}

		deviceImported := 0
		for i := range users {
			candidate := hikvision.UserInfoToEmployee(&users[i])
			if candidate == nil || strings.TrimSpace(candidate.EmployeeNo) == "" {
				continue
			}

			current, err := s.Store.GetEmployeeByNo(ctx, candidate.EmployeeNo)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %v", firstNonEmpty(device.Name, device.IP), candidate.EmployeeNo, err))
				continue
			}
			if current != nil {
				existing++
				continue
			}

			candidate.ID = uuid.New().String()
			if strings.TrimSpace(candidate.FirstName) == "" {
				candidate.FirstName = candidate.EmployeeNo
			}
			if strings.TrimSpace(candidate.LastName) == "" {
				candidate.LastName = "Importado"
			}

			if err := s.Store.UpsertEmployee(ctx, candidate); err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %v", firstNonEmpty(device.Name, device.IP), candidate.EmployeeNo, err))
				continue
			}

			imported++
			deviceImported++
		}

		s.saveDeviceOperationLog(ctx, managedDeviceLogID(&device), "ImportUsers", "info", fmt.Sprintf("Importacion completada. Nuevos empleados: %d", deviceImported))
	}

	return map[string]interface{}{
		"message":  "Importacion de empleados completada",
		"imported": imported,
		"existing": existing,
		"failed":   failed,
		"errors":   problems,
	}, nil
}

func isPhotoSyncFailure(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "failed", "delete_failed":
		return true
	default:
		return false
	}
}
