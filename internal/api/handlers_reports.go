package api

import (
	"fmt"
	"net/http"
	"ponches/internal/attendance"
	"ponches/internal/reports"
	"ponches/internal/store"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func (s *Server) handleReportDaily(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	dateStr := r.URL.Query().Get("date")
	employeeNo := r.URL.Query().Get("employee")

	// Parse date
	var targetDate time.Time
	if dateStr != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			targetDate = time.Now()
		}
	} else {
		targetDate = time.Now()
	}

	// Create processor with config from server
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	var results []attendance.DayResult

	if employeeNo != "" {
		// Get results for specific employee
		result, err := processor.CalculateDayResult(r.Context(), employeeNo, targetDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if result != nil && !result.IsAbsent {
			results = []attendance.DayResult{*result}
		}
	} else {
		// Get results for all employees
		dayResults, err := processor.CalculateAllEmployees(r.Context(), targetDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, r := range dayResults {
			results = append(results, *r)
		}
	}

	// Generate report
	filename := fmt.Sprintf("reporte_diario_%s", targetDate.Format("2006-01-02"))

	if format == "excel" {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".xlsx")
		reports.GenerateDailyExcel(w, s.Config.CompanyName, results)
	} else {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".pdf")
		reports.GenerateDailyPDF(w, s.Config.CompanyName, results)
	}
}

func (s *Server) handleReportPayroll(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Parse date range (default to current pay period - 1st to 15th or 16th to end)
	now := time.Now()
	var from, to time.Time

	if fromStr != "" && toStr != "" {
		var err error
		from, err = time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		}
		to, err = time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
		}
	} else {
		// Default: current month 1st to today
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	}

	// Get all employees
	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create processor
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	var payrollResults []attendance.PayrollResult

	for _, emp := range emps {
		if emp.Status != "Active" {
			continue
		}

		// Calculate attendance for the period
		results, err := processor.CalculateDateRange(r.Context(), emp.EmployeeNo, from, to)
		if err != nil {
			continue
		}

		// Convert to DayResult slice
		dayResults := make([]attendance.DayResult, len(results))
		for i, r := range results {
			dayResults[i] = *r
		}

		// Calculate payroll
		payroll := attendance.CalculatePayroll(
			emp.EmployeeNo,
			emp.FirstName+" "+emp.LastName,
			emp.BaseSalary,
			dayResults,
			s.Config.OvertimeMultiplierSimple,
			s.Config.OvertimeMultiplierDouble,
			s.Config.OvertimeMultiplierTriple,
		)
		payroll.PeriodFrom = from
		payroll.PeriodTo = to

		payrollResults = append(payrollResults, payroll)
	}

	// Generate report
	filename := fmt.Sprintf("prenomina_%s_%s", from.Format("2006-01-02"), to.Format("2006-01-02"))

	if format == "pdf" {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".pdf")
		reports.GeneratePayrollPDF(w, s.Config.CompanyName, payrollResults)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename+".xlsx")

	reports.GeneratePayrollExcel(w, s.Config.CompanyName, payrollResults)
}

// handleGetAttendanceSummary returns attendance summary for an employee or all employees
func (s *Server) handleGetAttendanceSummary(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	employeeNo := r.URL.Query().Get("employee")

	// Parse dates
	now := time.Now()
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		}
	} else {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toStr != "" {
		to, err = time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
		}
	} else {
		to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	}

	// Create processor
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	type SummaryResponse struct {
		EmployeeNo     string  `json:"employeeNo"`
		EmployeeName   string  `json:"employeeName"`
		TotalDays      int     `json:"totalDays"`
		PresentDays    int     `json:"presentDays"`
		AbsentDays     int     `json:"absentDays"`
		LateDays       int     `json:"lateDays"`
		TotalHours     float64 `json:"totalHours"`
		OvertimeHours  float64 `json:"overtimeHours"`
		AttendanceRate float64 `json:"attendanceRate"`
	}

	var summaries []SummaryResponse

	if employeeNo != "" {
		// Get specific employee
		emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if emp == nil {
			http.Error(w, "Employee not found", http.StatusNotFound)
			return
		}

		results, err := processor.CalculateDateRange(r.Context(), employeeNo, from, to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dayResults := make([]attendance.DayResult, len(results))
		for i, r := range results {
			dayResults[i] = *r
		}

		summary := attendance.CalculateSummary(employeeNo, dayResults)
		summaries = append(summaries, SummaryResponse{
			EmployeeNo:     employeeNo,
			EmployeeName:   emp.FirstName + " " + emp.LastName,
			TotalDays:      summary.TotalDays,
			PresentDays:    summary.PresentDays,
			AbsentDays:     summary.AbsentDays,
			LateDays:       summary.LateDays,
			TotalHours:     summary.TotalHours,
			OvertimeHours:  summary.OvertimeHours,
			AttendanceRate: summary.AttendanceRate,
		})
	} else {
		// Get all employees
		emps, err := s.Store.ListEmployees(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, emp := range emps {
			if emp.Status != "Active" {
				continue
			}

			results, err := processor.CalculateDateRange(r.Context(), emp.EmployeeNo, from, to)
			if err != nil {
				continue
			}

			dayResults := make([]attendance.DayResult, len(results))
			for i, r := range results {
				dayResults[i] = *r
			}

			summary := attendance.CalculateSummary(emp.EmployeeNo, dayResults)
			summaries = append(summaries, SummaryResponse{
				EmployeeNo:     emp.EmployeeNo,
				EmployeeName:   emp.FirstName + " " + emp.LastName,
				TotalDays:      summary.TotalDays,
				PresentDays:    summary.PresentDays,
				AbsentDays:     summary.AbsentDays,
				LateDays:       summary.LateDays,
				TotalHours:     summary.TotalHours,
				OvertimeHours:  summary.OvertimeHours,
				AttendanceRate: summary.AttendanceRate,
			})
		}
	}

	writeJSON(w, http.StatusOK, summaries)
}

// handleGetDailyAttendance returns attendance for a specific date
func (s *Server) handleGetDailyAttendance(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")

	var targetDate time.Time
	var err error

	if dateStr != "" {
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			targetDate = time.Now()
		}
	} else {
		targetDate = time.Now()
	}

	// Create processor
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	results, err := processor.CalculateAllEmployees(r.Context(), targetDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// handleGetStats returns dashboard statistics
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	// Create processor
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	// Get today's attendance
	results, err := processor.CalculateAllEmployees(r.Context(), today)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate stats
	present := 0
	late := 0
	absent := 0

	for _, r := range results {
		if r.IsAbsent {
			absent++
		} else {
			present++
			if r.IsLate {
				late++
			}
		}
	}

	// Get device count
	devices, _ := s.loadManagedDevices(r.Context())
	deviceCount := len(devices)

	// Get recent events (last 10)
	recentEvents, _ := s.Store.GetEvents(r.Context(), store.EventFilter{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now().Add(1 * time.Hour),
	})
	
	// Sort by timestamp descending
	sort.Slice(recentEvents, func(i, j int) bool {
		return recentEvents[i].Timestamp.After(recentEvents[j].Timestamp)
	})
	
	if len(recentEvents) > 10 {
		recentEvents = recentEvents[:10]
	}

	// Fetch employee names for recent events
	type EventWithEmployee struct {
		store.AttendanceEvent
		EmployeeName string `json:"employeeName"`
	}
	
	eventsWithNames := make([]EventWithEmployee, 0, len(recentEvents))
	for _, ev := range recentEvents {
		emp, _ := s.Store.GetEmployeeByNo(r.Context(), ev.EmployeeNo)
		name := "Desconocido"
		if emp != nil {
			name = emp.FirstName + " " + emp.LastName
		}
		eventsWithNames = append(eventsWithNames, EventWithEmployee{
			AttendanceEvent: *ev,
			EmployeeName:    name,
		})
	}

	stats := map[string]interface{}{
		"present":      present,
		"late":         late,
		"absent":       absent,
		"devices":      deviceCount,
		"date":         today.Format("2006-01-02"),
		"recentEvents": eventsWithNames,
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleReportLate genera el reporte de tardanzas y faltas
func (s *Server) handleReportLate(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Parse dates
	now := time.Now()
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		}
	} else {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toStr != "" {
		to, err = time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
		}
	} else {
		to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	}

	// Create processor
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	// Get all employees
	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Collect all late/absent results
	var lateResults []attendance.DayResult

	for _, emp := range emps {
		if emp.Status != "Active" {
			continue
		}

		results, err := processor.CalculateDateRange(r.Context(), emp.EmployeeNo, from, to)
		if err != nil {
			continue
		}

		for _, r := range results {
			if r.IsLate || r.IsAbsent {
				lateResults = append(lateResults, *r)
			}
		}
	}

	// Generate report
	filename := fmt.Sprintf("reporte_tardanzas_%s_%s", from.Format("2006-01-02"), to.Format("2006-01-02"))

	if format == "excel" {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".xlsx")
		reports.GenerateLateExcel(w, s.Config.CompanyName, lateResults)
	} else {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".pdf")
		reports.GenerateLateReportPDF(w, s.Config.CompanyName, lateResults, from, to)
	}
}

// handleReportKPIs genera el reporte de KPIs de asistencia
func (s *Server) handleReportKPIs(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Parse dates
	now := time.Now()
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		}
	} else {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toStr != "" {
		to, err = time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
		}
	} else {
		to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	}

	// Create processor
	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}

	processor := attendance.NewEventProcessor(s.Store, cfg)

	// Get all employees
	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate KPIs
	type KPIData struct {
		EmployeeNo      string  `json:"employeeNo"`
		EmployeeName    string  `json:"employeeName"`
		TotalDays       int     `json:"totalDays"`
		PresentDays     int     `json:"presentDays"`
		AbsentDays      int     `json:"absentDays"`
		LateDays        int     `json:"lateDays"`
		AttendanceRate  float64 `json:"attendanceRate"`
		PunctualityRate float64 `json:"punctualityRate"`
		TotalHours      float64 `json:"totalHours"`
		OvertimeHours   float64 `json:"overtimeHours"`
	}

	var kpiData []KPIData

	for _, emp := range emps {
		if emp.Status != "Active" {
			continue
		}

		results, err := processor.CalculateDateRange(r.Context(), emp.EmployeeNo, from, to)
		if err != nil {
			continue
		}

		dayResults := make([]attendance.DayResult, len(results))
		for i, r := range results {
			dayResults[i] = *r
		}

		summary := attendance.CalculateSummary(emp.EmployeeNo, dayResults)

		punctualityRate := 100.0
		if summary.PresentDays > 0 {
			punctualityRate = float64(summary.PresentDays-summary.LateDays) / float64(summary.PresentDays) * 100
		}

		kpiData = append(kpiData, KPIData{
			EmployeeNo:      emp.EmployeeNo,
			EmployeeName:    emp.FirstName + " " + emp.LastName,
			TotalDays:       summary.TotalDays,
			PresentDays:     summary.PresentDays,
			AbsentDays:      summary.AbsentDays,
			LateDays:        summary.LateDays,
			AttendanceRate:  summary.AttendanceRate,
			PunctualityRate: punctualityRate,
			TotalHours:      summary.TotalHours,
			OvertimeHours:   summary.OvertimeHours,
		})
	}

	// Generate Excel report (KPIs are better in Excel)
	filename := fmt.Sprintf("reporte_kpis_%s_%s.xlsx", from.Format("2006-01-02"), to.Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	// Create custom KPI Excel
	f := excelize.NewFile()
	defer f.Close()

	sheet := "KPIs"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{
		"Empleado", "No. Empleado", "Días Totales", "Días Presente",
		"Días Ausente", "Días Tarde", "% Asistencia", "% Puntualidad",
		"Horas Totales", "Horas Extra",
	}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}

	// Style for header
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "J1", style)

	// Data
	for i, k := range kpiData {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), k.EmployeeName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), k.EmployeeNo)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), k.TotalDays)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), k.PresentDays)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), k.AbsentDays)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), k.LateDays)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%.1f%%", k.AttendanceRate))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("%.1f%%", k.PunctualityRate))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%.2f", k.TotalHours))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("%.2f", k.OvertimeHours))
	}

	// Auto-fit columns
	for col := 'A'; col <= 'J'; col++ {
		f.SetColWidth(sheet, string(col), string(col), 14)
	}

	f.Write(w)
}

// buildAttendancePeriodRows is the shared data-fetch logic for attendance period reports.
func (s *Server) buildAttendancePeriodRows(r *http.Request, fromStr, toStr, filterEmployee, filterDept, search, status string) ([]reports.AttendanceRow, time.Time, time.Time, error) {
	now := time.Now()
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		}
	} else {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toStr != "" {
		to, err = time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
		}
	} else {
		to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	}

	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		return nil, from, to, err
	}

	depts, _ := s.Store.ListDepartments(r.Context())
	deptNameMap := map[string]string{}
	for _, d := range depts {
		deptNameMap[d.ID] = d.Name
	}

	nameMap := map[string]string{}
	deptMap := map[string]string{}

	// Fetch ALL events for the entire date range once
	allEvents, err := s.Store.GetEvents(r.Context(), store.EventFilter{
		From: from,
		To:   to.Add(24 * time.Hour),
	})
	if err != nil {
		return nil, from, to, err
	}

	// Map events by employee to avoid N+1 processing
	eventsByEmp := make(map[string][]*store.AttendanceEvent)
	for _, ev := range allEvents {
		eventsByEmp[ev.EmployeeNo] = append(eventsByEmp[ev.EmployeeNo], ev)
	}

	cfg := attendance.AttendanceConfig{
		ShiftStart:         s.Config.DefaultShiftStart,
		ShiftEnd:           s.Config.DefaultShiftEnd,
		LunchBreakMinutes:  s.Config.LunchBreakMinutes,
		GracePeriodMinutes: s.Config.GracePeriodMinutes,
		OvertimeThreshold:  s.Config.OvertimeThresholdHours,
	}
	processor := attendance.NewEventProcessor(s.Store, cfg)

	employeeResults := map[string][]attendance.DayResult{}

	for _, emp := range emps {
		if emp.Status != "Active" {
			continue
		}
		if filterEmployee != "" && emp.EmployeeNo != filterEmployee {
			continue
		}
		if filterDept != "" && emp.DepartmentID != filterDept {
			continue
		}

		nameMap[emp.EmployeeNo] = emp.FirstName + " " + emp.LastName
		deptMap[emp.EmployeeNo] = deptNameMap[emp.DepartmentID]

		// Process rows in memory for this employee
		empEvents := eventsByEmp[emp.EmployeeNo]
		
		// Map employee events by day for faster lookup
		dayEventsMap := make(map[string][]*store.AttendanceEvent)
		for _, ev := range empEvents {
			dKey := ev.Timestamp.Format("2006-01-02")
			dayEventsMap[dKey] = append(dayEventsMap[dKey], ev)
		}

		results := make([]attendance.DayResult, 0)
		curr := from
		for !curr.After(to) {
			dKey := curr.Format("2006-01-02")
			dayEvs := dayEventsMap[dKey]
			
			res := processor.ProcessEvents(emp.EmployeeNo, curr, dayEvs)
			res.EmployeeName = nameMap[emp.EmployeeNo]
			results = append(results, *res)
			
			curr = curr.Add(24 * time.Hour)
		}
		employeeResults[emp.EmployeeNo] = results
	}

	rows := reports.BuildAttendanceRows(employeeResults, nameMap, deptMap)

	// Final filtering by search and status
	filteredRows := make([]reports.AttendanceRow, 0)
	search = strings.ToLower(search)
	for _, row := range rows {
		if status != "" && row.Status != status {
			continue
		}
		if search != "" {
			hay := strings.ToLower(row.EmployeeName + " " + row.EmployeeNo)
			if !strings.Contains(hay, search) {
				continue
			}
		}
		filteredRows = append(filteredRows, row)
	}

	sort.Slice(filteredRows, func(i, j int) bool {
		if filteredRows[i].EmployeeName != filteredRows[j].EmployeeName {
			return filteredRows[i].EmployeeName < filteredRows[j].EmployeeName
		}
		return filteredRows[i].Date.Before(filteredRows[j].Date)
	})

	return filteredRows, from, to, nil
}

// handleReportAttendancePeriod generates a downloadable attendance report (Excel or PDF).
func (s *Server) handleReportAttendancePeriod(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	rows, from, to, err := s.buildAttendancePeriodRows(r,
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("employee"),
		r.URL.Query().Get("department"),
		r.URL.Query().Get("search"),
		r.URL.Query().Get("status"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("reporte_asistencia_%s_%s", from.Format("2006-01-02"), to.Format("2006-01-02"))

	if format == "pdf" {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".pdf")
		reports.GeneratePeriodAttendancePDF(w, s.Config.CompanyName, from, to, rows)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename+".xlsx")
	reports.GeneratePeriodAttendanceExcel(w, s.Config.CompanyName, from, to, rows)
}

// handleReportAttendanceData returns attendance rows as JSON for the live in-app view.
func (s *Server) handleReportAttendanceData(w http.ResponseWriter, r *http.Request) {
	rows, from, to, err := s.buildAttendancePeriodRows(r,
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("employee"),
		r.URL.Query().Get("department"),
		r.URL.Query().Get("search"),
		r.URL.Query().Get("status"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build summary
	present, late, absent, incomplete := 0, 0, 0, 0
	totalHours, totalOvertime := 0.0, 0.0
	for _, row := range rows {
		switch row.Status {
		case "Presente":
			present++
		case "Tarde":
			late++
		case "Falta":
			absent++
		case "Incompleto":
			incomplete++
		}
		totalHours += row.TotalHours
		totalOvertime += row.OvertimeHrs
	}

	// Serialize rows (need JSON-friendly format)
	type RowJSON struct {
		EmployeeNo   string  `json:"employeeNo"`
		EmployeeName string  `json:"employeeName"`
		Department   string  `json:"department"`
		Date         string  `json:"date"`
		CheckIn      string  `json:"checkIn"`
		CheckOut     string  `json:"checkOut"`
		TotalHours   float64 `json:"totalHours"`
		OvertimeHrs  float64 `json:"overtimeHrs"`
		LateMinutes  int     `json:"lateMinutes"`
		Status       string  `json:"status"`
	}

	jsonRows := make([]RowJSON, len(rows))
	for i, row := range rows {
		inStr := ""
		if row.CheckIn != nil {
			inStr = row.CheckIn.Format("15:04")
		}
		outStr := ""
		if row.CheckOut != nil {
			outStr = row.CheckOut.Format("15:04")
		}
		jsonRows[i] = RowJSON{
			EmployeeNo:   row.EmployeeNo,
			EmployeeName: row.EmployeeName,
			Department:   row.Department,
			Date:         row.Date.Format("2006-01-02"),
			CheckIn:      inStr,
			CheckOut:     outStr,
			TotalHours:   row.TotalHours,
			OvertimeHrs:  row.OvertimeHrs,
			LateMinutes:  row.LateMinutes,
			Status:       row.Status,
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"from":        from.Format("2006-01-02"),
		"to":          to.Format("2006-01-02"),
		"rows":        jsonRows,
		"summary": map[string]interface{}{
			"total":       len(rows),
			"present":     present,
			"late":        late,
			"absent":      absent,
			"incomplete":  incomplete,
			"totalHours":  totalHours,
			"totalOvertime": totalOvertime,
		},
	})
}
