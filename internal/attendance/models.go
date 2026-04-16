package attendance

import (
	"time"
)

// ShiftType represents the type of work shift
type ShiftType string

const (
	ShiftRegular ShiftType = "regular"
	ShiftSimple  ShiftType = "simple" // 1.5x
	ShiftDouble  ShiftType = "double" // 2.0x
	ShiftTriple  ShiftType = "triple" // 3.0x
)

// Shift represents a work schedule
type Shift struct {
	ID        string
	Name      string
	StartTime string // "08:00"
	EndTime   string // "17:00"
	GraceTime int    // minutes
	Type      ShiftType
	WorkDay   bool // true for regular work days, false for weekends/holidays
}

// DayResult represents the attendance calculation for a single day
type DayResult struct {
	EmployeeNo   string     `json:"employeeNo"`
	EmployeeName string     `json:"employeeName,omitempty"`
	Date         time.Time  `json:"date"`
	CheckIn      *time.Time `json:"checkIn,omitempty"`
	CheckOut     *time.Time `json:"checkOut,omitempty"`
	TotalHours   float64    `json:"totalHours"`
	RegularHours float64    `json:"regularHours"`
	Overtime     float64    `json:"overtime"`
	OvertimeType ShiftType  `json:"overtimeType"`
	IsLate       bool       `json:"isLate"`
	LateMinutes  int        `json:"lateMinutes"`
	IsAbsent     bool       `json:"isAbsent"`
	IsIncomplete bool       `json:"isIncomplete"`
	IsHoliday    bool       `json:"isHoliday"`
	Notes        string     `json:"notes,omitempty"`
}

// CalculateLateMinutes returns the number of minutes late
func (d *DayResult) CalculateLateMinutes(shiftStart string, gracePeriod int) int {
	if !d.IsLate || d.CheckIn == nil {
		return 0
	}

	start, err := time.Parse("15:04", shiftStart)
	if err != nil {
		return 0
	}

	allowedTime := start.Add(time.Duration(gracePeriod) * time.Minute)
	actualIn := time.Date(0, 1, 1, d.CheckIn.Hour(), d.CheckIn.Minute(), 0, 0, time.Local)
	allowedTime = time.Date(0, 1, 1, allowedTime.Hour(), allowedTime.Minute(), 0, 0, time.Local)

	if actualIn.After(allowedTime) {
		return int(actualIn.Sub(allowedTime).Minutes())
	}
	return 0
}

// PayrollResult represents the payroll calculation for an employee
type PayrollResult struct {
	EmployeeNo     string    `json:"employeeNo"`
	EmployeeName   string    `json:"employeeName"`
	BaseSalary     float64   `json:"baseSalary"`
	OvertimeHours  float64   `json:"overtimeHours"`
	OvertimeSimple float64   `json:"overtimeSimple"` // Hours at 1.5x
	OvertimeDouble float64   `json:"overtimeDouble"` // Hours at 2.0x
	OvertimeTriple float64   `json:"overtimeTriple"` // Hours at 3.0x
	OvertimePay    float64   `json:"overtimePay"`
	Deductions     float64   `json:"deductions"`
	Commissions    float64   `json:"commissions"`
	TotalToPay     float64   `json:"totalToPay"`
	PeriodFrom     time.Time `json:"periodFrom"`
	PeriodTo       time.Time `json:"periodTo"`
	DaysWorked     int       `json:"daysWorked"`
	DaysAbsent     int       `json:"daysAbsent"`
	DaysLate       int       `json:"daysLate"`
}

// CalculatePayroll computes the salary for an employee based on attendance results
func CalculatePayroll(empNo string, empName string, baseSalary float64, results []DayResult,
	overtimeSimpleRate, overtimeDoubleRate, overtimeTripleRate float64) PayrollResult {

	res := PayrollResult{
		EmployeeNo:   empNo,
		EmployeeName: empName,
		BaseSalary:   baseSalary,
	}

	totalSimple := 0.0
	totalDouble := 0.0
	totalTriple := 0.0

	for _, day := range results {
		if day.IsAbsent {
			res.DaysAbsent++
		} else {
			res.DaysWorked++
			if day.IsLate {
				res.DaysLate++
			}

			// Categorize overtime by type (simplified: all overtime is simple unless on holiday)
			if day.IsHoliday {
				totalTriple += day.Overtime
			} else {
				// Check if weekend (Saturday = 6, Sunday = 0)
				weekday := int(day.Date.Weekday())
				if weekday == 0 || weekday == 6 {
					totalDouble += day.Overtime
				} else {
					totalSimple += day.Overtime
				}
			}
		}
	}

	res.OvertimeSimple = totalSimple
	res.OvertimeDouble = totalDouble
	res.OvertimeTriple = totalTriple
	res.OvertimeHours = totalSimple + totalDouble + totalTriple

	// Calculate overtime pay
	// Hourly rate = BaseSalary / 160 (assuming 40h/week * 4 weeks = 160h/month)
	hourlyRate := baseSalary / 160.0
	res.OvertimePay = (totalSimple * hourlyRate * overtimeSimpleRate) +
		(totalDouble * hourlyRate * overtimeDoubleRate) +
		(totalTriple * hourlyRate * overtimeTripleRate)

	// Total
	res.TotalToPay = res.BaseSalary + res.OvertimePay + res.Commissions - res.Deductions

	return res
}
