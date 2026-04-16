package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"ponches/internal/employees"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ==================== LEAVES (PERMISOS Y AUSENCIAS) ====================

var validLeaveStatuses = map[string]bool{
	"Approved": true,
	"Pending":  true,
	"Rejected": true,
}

func parseLeaveDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC().Truncate(24 * time.Hour), nil
	}
	return time.Time{}, fmt.Errorf("invalid date format")
}

func (s *Server) handleListLeaves(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListLeaves(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []*employees.Leave{}
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleGetLeave(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l, err := s.Store.GetLeave(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if l == nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, 200, l)
}

func (s *Server) handleCreateLeave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmployeeID   string `json:"employeeId"`
		Type         string `json:"type"`
		StartDate    string `json:"startDate"`
		EndDate      string `json:"endDate"`
		Days         int    `json:"days"`
		Reason       string `json:"reason"`
		Status       string `json:"status"`
		AuthorizedBy string `json:"authorizedBy"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	if req.EmployeeID == "" || req.Type == "" || req.StartDate == "" || req.EndDate == "" {
		http.Error(w, "employeeId, type, startDate and endDate are required", 400)
		return
	}

	start, err := parseLeaveDate(req.StartDate)
	if err != nil {
		http.Error(w, "invalid startDate format", 400)
		return
	}
	end, err := parseLeaveDate(req.EndDate)
	if err != nil {
		http.Error(w, "invalid endDate format", 400)
		return
	}
	if end.Before(start) {
		http.Error(w, "endDate must be on or after startDate", 400)
		return
	}

	days := req.Days
	if days <= 0 {
		days = int(end.Sub(start).Hours()/24) + 1
	}
	status := req.Status
	if status == "" {
		status = "Approved"
	}
	if !validLeaveStatuses[status] {
		http.Error(w, "invalid status", 400)
		return
	}

	l := &employees.Leave{
		ID:           uuid.New().String(),
		EmployeeID:   req.EmployeeID,
		Type:         employees.LeaveType(req.Type),
		StartDate:    start,
		EndDate:      end,
		Days:         days,
		Reason:       req.Reason,
		Status:       status,
		AuthorizedBy: req.AuthorizedBy,
		Notes:        req.Notes,
	}

	if err := s.Store.CreateLeave(r.Context(), l); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Re-fetch to get JOINed fields
	created, _ := s.Store.GetLeave(r.Context(), l.ID)
	if created != nil {
		writeJSON(w, 201, created)
	} else {
		writeJSON(w, 201, l)
	}
}

func (s *Server) handleUpdateLeave(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	existing, err := s.Store.GetLeave(r.Context(), id)
	if err != nil || existing == nil {
		http.Error(w, "not found", 404)
		return
	}

	var req struct {
		EmployeeID   string `json:"employeeId"`
		Type         string `json:"type"`
		StartDate    string `json:"startDate"`
		EndDate      string `json:"endDate"`
		Days         int    `json:"days"`
		Reason       string `json:"reason"`
		Status       string `json:"status"`
		AuthorizedBy string `json:"authorizedBy"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	if req.EmployeeID != "" {
		existing.EmployeeID = req.EmployeeID
	}
	if req.Type != "" {
		existing.Type = employees.LeaveType(req.Type)
	}
	if req.StartDate != "" {
		t, err := parseLeaveDate(req.StartDate)
		if err != nil {
			http.Error(w, "invalid startDate format", 400)
			return
		}
		existing.StartDate = t
	}
	if req.EndDate != "" {
		t, err := parseLeaveDate(req.EndDate)
		if err != nil {
			http.Error(w, "invalid endDate format", 400)
			return
		}
		existing.EndDate = t
	}
	if req.Days > 0 {
		existing.Days = req.Days
	}
	if req.Reason != "" {
		existing.Reason = req.Reason
	}
	if req.Status != "" {
		if !validLeaveStatuses[req.Status] {
			http.Error(w, "invalid status", 400)
			return
		}
		existing.Status = req.Status
	}
	if req.AuthorizedBy != "" {
		existing.AuthorizedBy = req.AuthorizedBy
	}
	if req.Notes != "" {
		existing.Notes = req.Notes
	}
	if existing.EndDate.Before(existing.StartDate) {
		http.Error(w, "endDate must be on or after startDate", 400)
		return
	}
	if existing.Days <= 0 {
		existing.Days = int(existing.EndDate.Sub(existing.StartDate).Hours()/24) + 1
	}

	if err := s.Store.UpdateLeave(r.Context(), existing); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	updated, _ := s.Store.GetLeave(r.Context(), id)
	if updated != nil {
		writeJSON(w, 200, updated)
	} else {
		writeJSON(w, 200, existing)
	}
}

func (s *Server) handleDeleteLeave(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.Store.DeleteLeave(r.Context(), id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, map[string]string{"message": "deleted"})
}

// ==================== NOTIFICACIÓN / REPORTE DE EMPLEADO ====================

// handleNotifyEmployee builds a WhatsApp/email notification payload for a specific
// attendance incident and returns the pre-filled message + wa.me link to the frontend.
func (s *Server) handleNotifyEmployee(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmployeeNo   string  `json:"employeeNo"`
		EmployeeName string  `json:"employeeName"`
		Department   string  `json:"department"`
		Date         string  `json:"date"`
		Status       string  `json:"status"`
		CheckIn      string  `json:"checkIn"`
		CheckOut     string  `json:"checkOut"`
		LateMinutes  int     `json:"lateMinutes"`
		TotalHours   float64 `json:"totalHours"`
		Notes        string  `json:"notes"`
		// Manager contact from department
		ManagerPhone string `json:"managerPhone"`
		ManagerEmail string `json:"managerEmail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	// Build message
	statusLabel := map[string]string{
		"Falta":      "FALTA SIN JUSTIFICAR",
		"Tarde":      "TARDANZA",
		"Incompleto": "SALIDA INCOMPLETA",
		"Presente":   "ASISTENCIA",
	}[req.Status]
	if statusLabel == "" {
		statusLabel = req.Status
	}

	checkInStr := req.CheckIn
	if checkInStr == "" {
		checkInStr = "No registrado"
	}
	checkOutStr := req.CheckOut
	if checkOutStr == "" {
		checkOutStr = "No registrado"
	}

	lateStr := ""
	if req.LateMinutes > 0 {
		lateStr = fmt.Sprintf("\n⏱ Minutos de tardanza: *%d min*", req.LateMinutes)
	}

	dateFormatted := req.Date
	if t, err := time.Parse("2006-01-02", req.Date); err == nil {
		dateFormatted = t.Format("02/01/2006")
	}

	msg := fmt.Sprintf(`📋 *REPORTE DE ASISTENCIA*
━━━━━━━━━━━━━━━━━━━━━━━━
👤 Empleado: *%s* (No. %s)
🏢 Departamento: *%s*
📅 Fecha: *%s*
━━━━━━━━━━━━━━━━━━━━━━━━
🔴 Estado: *%s*
🟢 Entrada: *%s*
🔴 Salida: *%s*
⏰ Horas trabajadas: *%.2f h*%s
━━━━━━━━━━━━━━━━━━━━━━━━
%s`,
		req.EmployeeName, req.EmployeeNo,
		req.Department,
		dateFormatted,
		statusLabel,
		checkInStr,
		checkOutStr,
		req.TotalHours,
		lateStr,
		func() string {
			if req.Notes != "" {
				return "📝 Nota: " + req.Notes
			}
			return "Generado por Sistema de Ponches"
		}(),
	)

	result := map[string]interface{}{
		"message": msg,
	}

	// Build wa.me link if phone provided
	if req.ManagerPhone != "" {
		phone := req.ManagerPhone
		// Strip common non-digit chars
		cleanPhone := ""
		for _, c := range phone {
			if c >= '0' && c <= '9' {
				cleanPhone += string(c)
			}
		}
		result["whatsappUrl"] = "https://wa.me/" + cleanPhone + "?text=" + url.QueryEscape(msg)
	}

	// Email subject/body
	result["emailSubject"] = fmt.Sprintf("Reporte de Asistencia - %s - %s", req.EmployeeName, dateFormatted)
	result["emailBody"] = msg
	if req.ManagerEmail != "" {
		result["mailtoUrl"] = fmt.Sprintf("mailto:%s?subject=%s&body=%s",
			url.QueryEscape(req.ManagerEmail),
			url.QueryEscape(fmt.Sprintf("Reporte de Asistencia - %s - %s", req.EmployeeName, dateFormatted)),
			url.QueryEscape(msg))
	}

	writeJSON(w, 200, result)
}
