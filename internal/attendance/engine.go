package attendance

import (
	"context"
	"encoding/json"
	"ponches/internal/store"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DaySchedule represents the expected schedule for a single day
type DaySchedule struct {
	IsWorkday bool   `json:"is_workday"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

// AttendanceConfig holds configuration for attendance calculations
type AttendanceConfig struct {
	ShiftStart         string  // Global default
	ShiftEnd           string  // Global default
	WeeklyScheduleJSON string  // JSON string of map[string]DaySchedule
	LunchBreakMinutes  int     // Minutes to deduct for lunch
	GracePeriodMinutes int     // Minutes of tolerance for late arrival
	OvertimeThreshold  float64 // Global default
}

// DefaultConfig returns default attendance configuration
func DefaultConfig() AttendanceConfig {
	return AttendanceConfig{
		ShiftStart:         "08:00",
		ShiftEnd:           "17:00",
		LunchBreakMinutes:  60,
		GracePeriodMinutes: 5,
		OvertimeThreshold:  8.0,
	}
}

// EventProcessor handles attendance event processing
type EventProcessor struct {
	store    store.Repository
	config   AttendanceConfig
	schedule map[string]DaySchedule
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(s store.Repository, cfg AttendanceConfig) *EventProcessor {
	schedule := make(map[string]DaySchedule)
	if cfg.WeeklyScheduleJSON != "" {
		_ = json.Unmarshal([]byte(cfg.WeeklyScheduleJSON), &schedule)
	}
	return &EventProcessor{
		store:    s,
		config:   cfg,
		schedule: schedule,
	}
}

// CalculateDayResult calculates attendance for an employee on a specific date
func (p *EventProcessor) CalculateDayResult(ctx context.Context, employeeNo string, date time.Time) (*DayResult, error) {
	// Check if it's a holiday
	isHoliday, holiday, err := p.store.IsHoliday(ctx, date)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check holiday status")
	}

	// Get all events for this employee on this date
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	to := from.Add(24 * time.Hour)

	events, err := p.store.GetEvents(ctx, store.EventFilter{
		EmployeeNo: employeeNo,
		From:       from,
		To:         to,
	})
	if err != nil {
		return nil, err
	}

	result := p.ProcessEvents(employeeNo, date, events, isHoliday)
	if holiday != nil {
		result.Notes = holiday.Name
	}
	if err := p.attachEmployeeName(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ProcessEvents processes a list of attendance events and returns a DayResult
func (p *EventProcessor) ProcessEvents(employeeNo string, date time.Time, events []*store.AttendanceEvent, isHoliday bool) *DayResult {
	result := &DayResult{
		EmployeeNo: employeeNo,
		Date:       date,
		IsHoliday:  isHoliday,
	}

	weekday := date.Weekday().String() // "Monday", "Tuesday", etc.
	
	isWorkday := true
	shiftStart := p.config.ShiftStart
	shiftEnd := p.config.ShiftEnd
	
	if daySch, ok := p.schedule[weekday]; ok {
		isWorkday = daySch.IsWorkday
		shiftStart = daySch.Start
		shiftEnd = daySch.End
	}

	// On holidays, we treat it like a non-workday for regular hours, 
	// but all hours worked are overtime (Triple pay handled in Payroll)
	if isHoliday {
		isWorkday = false
	}

	if len(events) == 0 {
		if !isWorkday {
			result.IsAbsent = false
			result.Notes = "Día Libre"
			return result
		}
		result.IsAbsent = true
		return result
	}

	// Sort events by timestamp
	sortedEvents := make([]*store.AttendanceEvent, len(events))
	copy(sortedEvents, events)
	for i := 0; i < len(sortedEvents)-1; i++ {
		for j := i + 1; j < len(sortedEvents); j++ {
			if sortedEvents[i].Timestamp.After(sortedEvents[j].Timestamp) {
				sortedEvents[i], sortedEvents[j] = sortedEvents[j], sortedEvents[i]
			}
		}
	}

	// First event is check-in, last is check-out
	checkIn := sortedEvents[0].Timestamp
	result.CheckIn = &checkIn
	result.IsAbsent = false

	if len(events) == 1 {
		result.IsIncomplete = true
		result.Notes = "Solo un marcaje detectado"
		return result
	}

	checkOut := sortedEvents[len(sortedEvents)-1].Timestamp
	result.CheckOut = &checkOut

	// Total duration in hours
	duration := checkOut.Sub(checkIn).Hours()

	// Deduct lunch break only if working more than 5 hours
	if duration > 5.0 {
		lunchHours := float64(p.config.LunchBreakMinutes) / 60.0
		duration -= lunchHours
		if duration < 0 {
			duration = 0
		}
	}
	result.TotalHours = duration

	// Check for lateness
	result.IsLate = p.checkLateness(checkIn, shiftStart)
	result.LateMinutes = result.CalculateLateMinutes(shiftStart, p.config.GracePeriodMinutes)

	// Calculate regular and overtime hours
	if isWorkday {
		result.RegularHours, result.Overtime = p.calculateHours(result.TotalHours, shiftStart, shiftEnd)
	} else {
		result.RegularHours = 0
		result.Overtime = result.TotalHours
		result.Notes = "Trabajo en día no laborable"
	}

	return result
}

// checkLateness determines if the employee was late based on check-in time
func (p *EventProcessor) checkLateness(checkIn time.Time, shiftStart string) bool {
	shiftStartParts := strings.Split(shiftStart, ":")
	if len(shiftStartParts) != 2 {
		return false
	}
	
	hour, _ := strconv.Atoi(shiftStartParts[0])
	min, _ := strconv.Atoi(shiftStartParts[1])

	// Current date with shift start time
	startOfShift := time.Date(checkIn.Year(), checkIn.Month(), checkIn.Day(), hour, min, 0, 0, checkIn.Location())
	
	// Add grace period
	allowedTime := startOfShift.Add(time.Duration(p.config.GracePeriodMinutes) * time.Minute)

	return checkIn.After(allowedTime)
}

// calculateHours splits total hours into regular and overtime
func (p *EventProcessor) calculateHours(totalHours float64, shiftStart, shiftEnd string) (regular, overtime float64) {
	threshold := p.config.OvertimeThreshold
	
	startParts := strings.Split(shiftStart, ":")
	endParts := strings.Split(shiftEnd, ":")
	
	if len(startParts) == 2 && len(endParts) == 2 {
		h1, _ := strconv.Atoi(startParts[0])
		m1, _ := strconv.Atoi(startParts[1])
		h2, _ := strconv.Atoi(endParts[0])
		m2, _ := strconv.Atoi(endParts[1])
		
		t1 := float64(h1) + float64(m1)/60.0
		t2 := float64(h2) + float64(m2)/60.0
		
		diff := t2 - t1
		if diff > 5.0 {
			lunchHours := float64(p.config.LunchBreakMinutes) / 60.0
			diff -= lunchHours
		}
		if diff > 0 {
			threshold = diff
		}
	}

	if totalHours <= threshold {
		return totalHours, 0
	}
	return threshold, totalHours - threshold
}

// CalculateDateRange calculates attendance for an employee over a date range
func (p *EventProcessor) CalculateDateRange(ctx context.Context, employeeNo string, from, to time.Time) ([]*DayResult, error) {
	var results []*DayResult

	current := from
	for !current.After(to) {
		result, err := p.CalculateDayResult(ctx, employeeNo, current)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
		current = current.Add(24 * time.Hour)
	}

	return results, nil
}

// CalculateAllEmployees calculates attendance for all employees on a specific date
func (p *EventProcessor) CalculateAllEmployees(ctx context.Context, date time.Time) ([]*DayResult, error) {
	// Check if it's a holiday
	isHoliday, holiday, err := p.store.IsHoliday(ctx, date)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check holiday status")
	}

	employees, err := p.store.ListEmployees(ctx)
	if err != nil {
		return nil, err
	}

	// Get all events for the date
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	to := from.Add(24 * time.Hour)

	events, err := p.store.GetEvents(ctx, store.EventFilter{
		From: from,
		To:   to,
	})
	if err != nil {
		return nil, err
	}

	// Group events by employee
	employeeEvents := make(map[string][]*store.AttendanceEvent)
	for _, event := range events {
		employeeEvents[event.EmployeeNo] = append(employeeEvents[event.EmployeeNo], event)
	}

	sort.Slice(employees, func(i, j int) bool {
		if employees[i].LastName == employees[j].LastName {
			if employees[i].FirstName == employees[j].FirstName {
				return employees[i].EmployeeNo < employees[j].EmployeeNo
			}
			return employees[i].FirstName < employees[j].FirstName
		}
		return employees[i].LastName < employees[j].LastName
	})

	// Calculate results for each active employee, including absences
	var results []*DayResult
	for _, employee := range employees {
		if employee.Status != "" && employee.Status != "Active" {
			continue
		}

		result := p.ProcessEvents(employee.EmployeeNo, date, employeeEvents[employee.EmployeeNo], isHoliday)
		if holiday != nil {
			result.Notes = holiday.Name
		}
		result.EmployeeName = strings.TrimSpace(employee.FirstName + " " + employee.LastName)
		results = append(results, result)
	}

	return results, nil
}

func (p *EventProcessor) attachEmployeeName(ctx context.Context, result *DayResult) error {
	if result == nil || result.EmployeeNo == "" {
		return nil
	}

	employee, err := p.store.GetEmployeeByNo(ctx, result.EmployeeNo)
	if err != nil {
		return err
	}
	if employee != nil {
		result.EmployeeName = strings.TrimSpace(employee.FirstName + " " + employee.LastName)
	}
	return nil
}

// GetAttendanceSummary returns a summary of attendance for a date range
type AttendanceSummary struct {
	EmployeeNo     string  `json:"employeeNo"`
	TotalDays      int     `json:"totalDays"`
	PresentDays    int     `json:"presentDays"`
	AbsentDays     int     `json:"absentDays"`
	LateDays       int     `json:"lateDays"`
	TotalHours     float64 `json:"totalHours"`
	OvertimeHours  float64 `json:"overtimeHours"`
	AverageHours   float64 `json:"averageHours"`
	AttendanceRate float64 `json:"attendanceRate"`
}

// CalculateSummary calculates attendance summary for an employee over a date range
func (p *EventProcessor) CalculateSummary(employeeNo string, results []*DayResult) AttendanceSummary {
	summary := AttendanceSummary{
		EmployeeNo: employeeNo,
		TotalDays:  len(results),
	}

	for _, result := range results {
		if result.IsAbsent {
			summary.AbsentDays++
		} else {
			summary.PresentDays++
			summary.TotalHours += result.TotalHours
			summary.OvertimeHours += result.Overtime
			if result.IsLate {
				summary.LateDays++
			}
		}
	}

	if summary.PresentDays > 0 {
		summary.AverageHours = summary.TotalHours / float64(summary.PresentDays)
		summary.AttendanceRate = float64(summary.PresentDays) / float64(summary.TotalDays) * 100
	}

	return summary
}

// CalculateSummary is a helper function to calculate summary without processor
func CalculateSummary(employeeNo string, results []DayResult) AttendanceSummary {
	// Convert to pointer slice
	ptrResults := make([]*DayResult, len(results))
	for i := range results {
		ptrResults[i] = &results[i]
	}

	processor := &EventProcessor{}
	return processor.CalculateSummary(employeeNo, ptrResults)
}
