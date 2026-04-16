package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"ponches/internal/auth"
	"ponches/internal/employees"
	"ponches/internal/reports"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Daily salary divisor (standard Dominican Republic: 365/12/30.44 ≈ 23.83)
const dailySalaryDivisor = 23.83

// ==================== TRAVEL RATES ====================

func (s *Server) handleListTravelRates(w http.ResponseWriter, r *http.Request) {
	rates, err := s.Store.ListTravelRates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list travel rates")
		return
	}
	if rates == nil {
		rates = []*employees.TravelAllowanceRate{}
	}
	writeJSON(w, http.StatusOK, rates)
}

func (s *Server) handleCreateTravelRate(w http.ResponseWriter, r *http.Request) {
	var rate employees.TravelAllowanceRate
	if err := json.NewDecoder(r.Body).Decode(&rate); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateTravelRate(&rate); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if rate.ID == "" {
		rate.ID = uuid.New().String()
	}

	if err := s.Store.CreateTravelRate(r.Context(), &rate); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create travel rate")
		return
	}

	writeJSON(w, http.StatusCreated, rate)
}

func (s *Server) handleUpdateTravelRate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Rate ID is required")
		return
	}

	var rate employees.TravelAllowanceRate
	if err := json.NewDecoder(r.Body).Decode(&rate); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	rate.ID = id

	if err := validateTravelRate(&rate); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.Store.UpdateTravelRate(r.Context(), &rate); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Rate not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update travel rate")
		return
	}

	writeJSON(w, http.StatusOK, rate)
}

func (s *Server) handleDeleteTravelRate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Rate ID is required")
		return
	}

	if err := s.Store.DeleteTravelRate(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Rate not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete travel rate")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateTravelRate(r *employees.TravelAllowanceRate) error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if r.Type != "percentage" && r.Type != "fixed" {
		return errors.New("type must be 'percentage' or 'fixed'")
	}
	if r.Value <= 0 {
		return errors.New("value must be greater than zero")
	}
	if r.Type == "percentage" && r.Value > 100 {
		return errors.New("percentage value cannot exceed 100")
	}
	return nil
}

// ==================== TRAVEL ALLOWANCES ====================

func (s *Server) handleListTravelAllowances(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListTravelAllowances(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list travel allowances")
		return
	}
	if list == nil {
		list = []*employees.TravelAllowance{}
	}
	groupSizes := map[string]int{}
	for _, item := range list {
		if strings.TrimSpace(item.GroupID) != "" {
			groupSizes[item.GroupID]++
		}
	}
	for _, item := range list {
		if strings.TrimSpace(item.GroupID) != "" {
			item.GroupSize = groupSizes[item.GroupID]
		}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetTravelAllowance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "ID is required")
		return
	}

	ta, err := s.Store.GetTravelAllowance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get travel allowance")
		return
	}
	if ta == nil {
		writeError(w, http.StatusNotFound, "Travel allowance not found")
		return
	}
	writeJSON(w, http.StatusOK, ta)
}

func (s *Server) handleCreateTravelAllowance(w http.ResponseWriter, r *http.Request) {
	var req employees.TravelAllowance
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateTravelAllowance(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rate, err := s.Store.GetTravelRate(r.Context(), req.RateID)
	if err != nil || rate == nil {
		writeError(w, http.StatusBadRequest, "Invalid or missing travel rate")
		return
	}

	employeeIDs := normalizeTravelEmployeeIDs(req.EmployeeID, req.EmployeeIDs)
	days := int(req.ReturnDate.Sub(req.DepartureDate).Hours()/24) + 1
	if days < 1 {
		days = 1
	}

	groupID := ""
	groupName := strings.TrimSpace(req.GroupName)
	if len(employeeIDs) > 1 {
		groupID = uuid.New().String()
		if groupName == "" {
			groupName = fmt.Sprintf("Grupo %s", req.Destination)
		}
	}

	employeeMap := make(map[string]*employees.Employee, len(employeeIDs))
	for _, employeeID := range employeeIDs {
		emp, err := s.Store.GetEmployee(r.Context(), employeeID)
		if err != nil || emp == nil {
			writeError(w, http.StatusBadRequest, "Invalid or missing employee")
			return
		}
		employeeMap[employeeID] = emp
	}

	createdItems := make([]*employees.TravelAllowance, 0, len(employeeIDs))
	for _, employeeID := range employeeIDs {
		emp := employeeMap[employeeID]

		item := employees.TravelAllowance{
			ID:               uuid.New().String(),
			EmployeeID:       employeeID,
			EmployeeIDs:      employeeIDs,
			RateID:           req.RateID,
			Destination:      req.Destination,
			DepartureDate:    req.DepartureDate,
			ReturnDate:       req.ReturnDate,
			Days:             days,
			Reason:           req.Reason,
			CalculatedAmount: calculateTravelAmount(rate, emp.BaseSalary, days),
			Status:           "Pending",
			GroupID:          groupID,
			GroupName:        groupName,
		}

		if err := s.Store.CreateTravelAllowance(r.Context(), &item); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create travel allowance")
			return
		}

		created, _ := s.Store.GetTravelAllowance(r.Context(), item.ID)
		if created == nil {
			created = &item
		}
		if groupID != "" {
			created.GroupSize = len(employeeIDs)
		}
		createdItems = append(createdItems, created)
	}

	if len(createdItems) == 1 {
		writeJSON(w, http.StatusCreated, createdItems[0])
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"groupId":      groupID,
		"groupName":    groupName,
		"createdCount": len(createdItems),
		"records":      createdItems,
	})
}

func (s *Server) handleUpdateTravelAllowance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "ID is required")
		return
	}

	existing, err := s.Store.GetTravelAllowance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get travel allowance")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "Travel allowance not found")
		return
	}

	// Only allow editing if still Pending
	if existing.Status != "Pending" {
		writeError(w, http.StatusBadRequest, "Only pending requests can be edited")
		return
	}

	var ta employees.TravelAllowance
	if err := json.NewDecoder(r.Body).Decode(&ta); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	ta.ID = id

	if err := validateTravelAllowance(&ta); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(ta.EmployeeIDs) > 1 {
		writeError(w, http.StatusBadRequest, "Grouped travel requests must be edited from a new group submission")
		return
	}

	// Recalculate
	ta.Days = int(ta.ReturnDate.Sub(ta.DepartureDate).Hours()/24) + 1
	if ta.Days < 1 {
		ta.Days = 1
	}

	rate, err := s.Store.GetTravelRate(r.Context(), ta.RateID)
	if err != nil || rate == nil {
		writeError(w, http.StatusBadRequest, "Invalid or missing travel rate")
		return
	}

	emp, err := s.Store.GetEmployee(r.Context(), ta.EmployeeID)
	if err != nil || emp == nil {
		writeError(w, http.StatusBadRequest, "Invalid or missing employee")
		return
	}

	ta.CalculatedAmount = calculateTravelAmount(rate, emp.BaseSalary, ta.Days)
	ta.Status = "Pending"
	ta.GroupID = existing.GroupID
	ta.GroupName = existing.GroupName

	if err := s.Store.UpdateTravelAllowance(r.Context(), &ta); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Travel allowance not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update travel allowance")
		return
	}

	updated, _ := s.Store.GetTravelAllowance(r.Context(), ta.ID)
	if updated != nil {
		writeJSON(w, http.StatusOK, updated)
	} else {
		writeJSON(w, http.StatusOK, ta)
	}
}

func (s *Server) handleDeleteTravelAllowance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "ID is required")
		return
	}

	existing, err := s.Store.GetTravelAllowance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get travel allowance")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "Travel allowance not found")
		return
	}
	if existing.Status != "Pending" {
		writeError(w, http.StatusBadRequest, "Only pending requests can be deleted")
		return
	}

	if err := s.Store.DeleteTravelAllowance(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete travel allowance")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleApproveTravelAllowance(w http.ResponseWriter, r *http.Request) {
	s.handleTravelDecision(w, r, "Approved")
}

func (s *Server) handleRejectTravelAllowance(w http.ResponseWriter, r *http.Request) {
	s.handleTravelDecision(w, r, "Rejected")
}

func (s *Server) handleTravelDecision(w http.ResponseWriter, r *http.Request, decision string) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "ID is required")
		return
	}

	ta, err := s.Store.GetTravelAllowance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get travel allowance")
		return
	}
	if ta == nil {
		writeError(w, http.StatusNotFound, "Travel allowance not found")
		return
	}
	if ta.Status != "Pending" {
		writeError(w, http.StatusBadRequest, "Only pending requests can be "+strings.ToLower(decision))
		return
	}

	// Parse optional notes from body
	var body struct {
		Notes string `json:"notes"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Get current user from JWT context
	userInfo, ok := auth.GetUserFromContext(r.Context())
	if ok {
		ta.ApprovedBy = userInfo.UserID
	}
	ta.Status = decision
	ta.ApprovalNotes = body.Notes

	if err := s.Store.UpdateTravelAllowance(r.Context(), ta); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update travel allowance")
		return
	}

	updated, _ := s.Store.GetTravelAllowance(r.Context(), ta.ID)
	if updated != nil {
		if updated.GroupID != "" {
			list, _ := s.Store.ListTravelAllowances(r.Context())
			for _, item := range list {
				if item.GroupID == updated.GroupID {
					updated.GroupSize++
				}
			}
		}
		writeJSON(w, http.StatusOK, updated)
	} else {
		writeJSON(w, http.StatusOK, ta)
	}
}

func validateTravelAllowance(ta *employees.TravelAllowance) error {
	employeeIDs := normalizeTravelEmployeeIDs(ta.EmployeeID, ta.EmployeeIDs)
	if len(employeeIDs) == 0 {
		return errors.New("employeeId or employeeIds is required")
	}
	if strings.TrimSpace(ta.RateID) == "" {
		return errors.New("rateId is required")
	}
	if strings.TrimSpace(ta.Destination) == "" {
		return errors.New("destination is required")
	}
	if ta.DepartureDate.IsZero() {
		return errors.New("departureDate is required")
	}
	if ta.ReturnDate.IsZero() {
		return errors.New("returnDate is required")
	}
	if ta.ReturnDate.Before(ta.DepartureDate) {
		return errors.New("returnDate must be on or after departureDate")
	}
	ta.EmployeeIDs = employeeIDs
	if len(employeeIDs) == 1 {
		ta.EmployeeID = employeeIDs[0]
	}
	return nil
}

func normalizeTravelEmployeeIDs(employeeID string, employeeIDs []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(employeeIDs)+1)

	push := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	push(employeeID)
	for _, id := range employeeIDs {
		push(id)
	}

	return result
}

func calculateTravelAmount(rate *employees.TravelAllowanceRate, baseSalary float64, days int) float64 {
	var amount float64
	switch rate.Type {
	case "percentage":
		dailySalary := baseSalary / dailySalaryDivisor
		amount = dailySalary * (rate.Value / 100) * float64(days)
	case "fixed":
		amount = rate.Value * float64(days)
	}
	// Round to 2 decimal places
	return math.Round(amount*100) / 100
}

func (s *Server) handleTravelAllowancePDF(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "ID is required")
		return
	}

	ta, err := s.Store.GetTravelAllowance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get travel allowance")
		return
	}
	if ta == nil {
		writeError(w, http.StatusNotFound, "Travel allowance not found")
		return
	}

	filename := fmt.Sprintf("vale_viatico_%s_%s", ta.ID[:8], strings.ReplaceAll(ta.EmployeeName, " ", "_"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename+".pdf")

	if err := reports.GenerateTravelAllowancePDF(w, s.Config.CompanyName, ta); err != nil {
		log.Error().Err(err).Msg("Failed to generate travel allowance PDF")
	}
}
