package attendance

import (
	"context"
	"ponches/internal/employees"
	"ponches/internal/store"
	"ponches/internal/users"
	"testing"
	"time"
)

// MockStore for testing
type MockStore struct {
	events    []*store.AttendanceEvent
	employees map[string]*employees.Employee
}

func (m *MockStore) SaveEvent(ctx context.Context, event *store.AttendanceEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *MockStore) GetEvents(ctx context.Context, filter store.EventFilter) ([]*store.AttendanceEvent, error) {
	var filtered []*store.AttendanceEvent
	for _, e := range m.events {
		if filter.EmployeeNo != "" && e.EmployeeNo != filter.EmployeeNo {
			continue
		}
		if !filter.From.IsZero() && e.Timestamp.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && e.Timestamp.After(filter.To) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered, nil
}

// Employee store methods (not used in these tests)
func (m *MockStore) CreateEmployee(ctx context.Context, e *employees.Employee) error { return nil }
func (m *MockStore) GetEmployee(ctx context.Context, id string) (*employees.Employee, error) {
	return nil, nil
}
func (m *MockStore) GetEmployeeByNo(ctx context.Context, no string) (*employees.Employee, error) {
	if m.employees == nil {
		return nil, nil
	}
	return m.employees[no], nil
}
func (m *MockStore) ListEmployees(ctx context.Context) ([]*employees.Employee, error) {
	if m.employees == nil {
		return nil, nil
	}

	list := make([]*employees.Employee, 0, len(m.employees))
	for _, employee := range m.employees {
		list = append(list, employee)
	}
	return list, nil
}
func (m *MockStore) UpdateEmployee(ctx context.Context, e *employees.Employee) error     { return nil }
func (m *MockStore) DeleteEmployee(ctx context.Context, id string) error                 { return nil }
func (m *MockStore) UpsertEmployee(ctx context.Context, e *employees.Employee) error     { return nil }
func (m *MockStore) CreateDepartment(ctx context.Context, d *employees.Department) error { return nil }
func (m *MockStore) GetDepartment(ctx context.Context, id string) (*employees.Department, error) {
	return nil, nil
}
func (m *MockStore) ListDepartments(ctx context.Context) ([]*employees.Department, error) {
	return nil, nil
}
func (m *MockStore) UpdateDepartment(ctx context.Context, d *employees.Department) error { return nil }
func (m *MockStore) DeleteDepartment(ctx context.Context, id string) error               { return nil }
func (m *MockStore) UpsertDepartment(ctx context.Context, d *employees.Department) error { return nil }
func (m *MockStore) CreatePosition(ctx context.Context, p *employees.Position) error     { return nil }
func (m *MockStore) GetPosition(ctx context.Context, id string) (*employees.Position, error) {
	return nil, nil
}
func (m *MockStore) ListPositions(ctx context.Context) ([]*employees.Position, error) {
	return nil, nil
}
func (m *MockStore) UpdatePosition(ctx context.Context, p *employees.Position) error { return nil }
func (m *MockStore) DeletePosition(ctx context.Context, id string) error             { return nil }
func (m *MockStore) UpsertPosition(ctx context.Context, p *employees.Position) error { return nil }
func (m *MockStore) CreateUser(ctx context.Context, u *users.User) error             { return nil }
func (m *MockStore) GetUser(ctx context.Context, id string) (*users.User, error)     { return nil, nil }
func (m *MockStore) GetUserByID(ctx context.Context, id string) (*users.User, error) { return nil, nil }
func (m *MockStore) GetUserByUsername(ctx context.Context, username string) (*users.User, error) {
	return nil, nil
}
func (m *MockStore) ListUsers(ctx context.Context) ([]*users.User, error)           { return nil, nil }
func (m *MockStore) UpdateUser(ctx context.Context, u *users.User) error            { return nil }
func (m *MockStore) DeleteUser(ctx context.Context, id string) error                { return nil }
func (m *MockStore) GetConfigValue(ctx context.Context, key string) (string, error) { return "", nil }
func (m *MockStore) SetConfigValue(ctx context.Context, key, value string) error    { return nil }
func (m *MockStore) GetAllConfig(ctx context.Context) (map[string]string, error)    { return nil, nil }
func (m *MockStore) SetMultipleConfigValues(ctx context.Context, values map[string]string) error {
	return nil
}
func (m *MockStore) CreateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error {
	return nil
}
func (m *MockStore) GetTravelRate(ctx context.Context, id string) (*employees.TravelAllowanceRate, error) {
	return nil, nil
}
func (m *MockStore) ListTravelRates(ctx context.Context) ([]*employees.TravelAllowanceRate, error) {
	return nil, nil
}
func (m *MockStore) UpdateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error {
	return nil
}
func (m *MockStore) DeleteTravelRate(ctx context.Context, id string) error { return nil }
func (m *MockStore) CreateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error {
	return nil
}
func (m *MockStore) GetTravelAllowance(ctx context.Context, id string) (*employees.TravelAllowance, error) {
	return nil, nil
}
func (m *MockStore) ListTravelAllowances(ctx context.Context) ([]*employees.TravelAllowance, error) {
	return nil, nil
}
func (m *MockStore) UpdateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error {
	return nil
}
func (m *MockStore) DeleteTravelAllowance(ctx context.Context, id string) error { return nil }
func (m *MockStore) CreateLeave(ctx context.Context, l *employees.Leave) error  { return nil }
func (m *MockStore) GetLeave(ctx context.Context, id string) (*employees.Leave, error) {
	return nil, nil
}
func (m *MockStore) ListLeaves(ctx context.Context) ([]*employees.Leave, error) { return nil, nil }
func (m *MockStore) ListLeavesByEmployee(ctx context.Context, employeeID string) ([]*employees.Leave, error) {
	return nil, nil
}
func (m *MockStore) UpdateLeave(ctx context.Context, l *employees.Leave) error { return nil }
func (m *MockStore) DeleteLeave(ctx context.Context, id string) error          { return nil }

func TestProcessEvents_EmptyEvents(t *testing.T) {
	cfg := DefaultConfig()
	processor := NewEventProcessor(&MockStore{}, cfg)

	date := time.Now()
	result := processor.ProcessEvents("001", date, []*store.AttendanceEvent{})

	if !result.IsAbsent {
		t.Error("Expected IsAbsent to be true for empty events")
	}
	if result.EmployeeNo != "001" {
		t.Errorf("Expected EmployeeNo 001, got %s", result.EmployeeNo)
	}
}

func TestProcessEvents_SingleEvent(t *testing.T) {
	cfg := AttendanceConfig{
		ShiftStart:         "08:00",
		ShiftEnd:           "17:00",
		LunchBreakMinutes:  60,
		GracePeriodMinutes: 5,
		OvertimeThreshold:  8.0,
	}
	processor := NewEventProcessor(&MockStore{}, cfg)

	date := time.Date(2024, 1, 15, 8, 0, 0, 0, time.Local)
	events := []*store.AttendanceEvent{
		{EmployeeNo: "001", Timestamp: date, Type: "Access"},
	}

	result := processor.ProcessEvents("001", date, events)

	if !result.IsAbsent {
		t.Error("Expected IsAbsent to be true for incomplete attendance")
	}
	if result.CheckIn == nil {
		t.Error("Expected CheckIn to be set")
	}
	if result.CheckOut != nil {
		t.Error("Expected CheckOut to be nil for incomplete attendance")
	}
	if !result.IsIncomplete {
		t.Error("Expected IsIncomplete to be true for single event")
	}
}

func TestProcessEvents_CheckInLate(t *testing.T) {
	cfg := AttendanceConfig{
		ShiftStart:         "08:00",
		GracePeriodMinutes: 5,
	}
	processor := NewEventProcessor(&MockStore{}, cfg)

	date := time.Date(2024, 1, 15, 8, 15, 0, 0, time.Local) // 15 minutes late
	events := []*store.AttendanceEvent{
		{EmployeeNo: "001", Timestamp: date, Type: "Access"},
	}

	result := processor.ProcessEvents("001", date, events)

	if result.IsLate {
		t.Error("Expected incomplete attendance not to be marked as late")
	}
}

func TestProcessEvents_CheckInOnTime(t *testing.T) {
	cfg := AttendanceConfig{
		ShiftStart:         "08:00",
		GracePeriodMinutes: 5,
	}
	processor := NewEventProcessor(&MockStore{}, cfg)

	date := time.Date(2024, 1, 15, 8, 3, 0, 0, time.Local) // 3 minutes late but within grace
	events := []*store.AttendanceEvent{
		{EmployeeNo: "001", Timestamp: date, Type: "Access"},
	}

	result := processor.ProcessEvents("001", date, events)

	if result.IsLate {
		t.Error("Expected IsLate to be false (within grace period)")
	}
}

func TestCalculateDayResult_AttachesEmployeeName(t *testing.T) {
	cfg := DefaultConfig()
	mockStore := &MockStore{
		employees: map[string]*employees.Employee{
			"001": {EmployeeNo: "001", FirstName: "Ana", LastName: "Lopez"},
		},
	}
	processor := NewEventProcessor(mockStore, cfg)

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	result, err := processor.CalculateDayResult(context.Background(), "001", date)
	if err != nil {
		t.Fatalf("CalculateDayResult failed: %v", err)
	}

	if result.EmployeeName != "Ana Lopez" {
		t.Fatalf("Expected EmployeeName Ana Lopez, got %q", result.EmployeeName)
	}
}

func TestProcessEvents_FullDay(t *testing.T) {
	cfg := AttendanceConfig{
		ShiftStart:         "08:00",
		ShiftEnd:           "17:00",
		LunchBreakMinutes:  60,
		GracePeriodMinutes: 5,
		OvertimeThreshold:  8.0,
	}
	processor := NewEventProcessor(&MockStore{}, cfg)

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	checkIn := time.Date(2024, 1, 15, 8, 0, 0, 0, time.Local)
	checkOut := time.Date(2024, 1, 15, 17, 0, 0, 0, time.Local)

	events := []*store.AttendanceEvent{
		{EmployeeNo: "001", Timestamp: checkIn, Type: "Access"},
		{EmployeeNo: "001", Timestamp: checkOut, Type: "Access"},
	}

	result := processor.ProcessEvents("001", date, events)

	expectedHours := 8.0 // 9 hours - 1 hour lunch
	if result.TotalHours != expectedHours {
		t.Errorf("Expected TotalHours %.2f, got %.2f", expectedHours, result.TotalHours)
	}
	if result.Overtime != 0 {
		t.Errorf("Expected Overtime 0, got %.2f", result.Overtime)
	}
}

func TestProcessEvents_WithOvertime(t *testing.T) {
	cfg := AttendanceConfig{
		ShiftStart:         "08:00",
		ShiftEnd:           "17:00",
		LunchBreakMinutes:  60,
		GracePeriodMinutes: 5,
		OvertimeThreshold:  8.0,
	}
	processor := NewEventProcessor(&MockStore{}, cfg)

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	checkIn := time.Date(2024, 1, 15, 8, 0, 0, 0, time.Local)
	checkOut := time.Date(2024, 1, 15, 19, 0, 0, 0, time.Local) // 2 hours overtime

	events := []*store.AttendanceEvent{
		{EmployeeNo: "001", Timestamp: checkIn, Type: "Access"},
		{EmployeeNo: "001", Timestamp: checkOut, Type: "Access"},
	}

	result := processor.ProcessEvents("001", date, events)

	expectedHours := 10.0   // 11 hours - 1 hour lunch
	expectedOvertime := 2.0 // 10 hours - 8 threshold

	if result.TotalHours != expectedHours {
		t.Errorf("Expected TotalHours %.2f, got %.2f", expectedHours, result.TotalHours)
	}
	if result.Overtime != expectedOvertime {
		t.Errorf("Expected Overtime %.2f, got %.2f", expectedOvertime, result.Overtime)
	}
}

func TestCalculateSummary(t *testing.T) {
	results := []DayResult{
		{EmployeeNo: "001", IsAbsent: false, IsLate: false, TotalHours: 8, Overtime: 0},
		{EmployeeNo: "001", IsAbsent: false, IsLate: true, TotalHours: 8, Overtime: 1},
		{EmployeeNo: "001", IsAbsent: true, TotalHours: 0, Overtime: 0},
		{EmployeeNo: "001", IsAbsent: false, IsLate: false, TotalHours: 9, Overtime: 1},
	}

	summary := CalculateSummary("001", results)

	if summary.TotalDays != 4 {
		t.Errorf("Expected TotalDays 4, got %d", summary.TotalDays)
	}
	if summary.PresentDays != 3 {
		t.Errorf("Expected PresentDays 3, got %d", summary.PresentDays)
	}
	if summary.AbsentDays != 1 {
		t.Errorf("Expected AbsentDays 1, got %d", summary.AbsentDays)
	}
	if summary.LateDays != 1 {
		t.Errorf("Expected LateDays 1, got %d", summary.LateDays)
	}
	if summary.TotalHours != 25 {
		t.Errorf("Expected TotalHours 25, got %.2f", summary.TotalHours)
	}
	if summary.OvertimeHours != 2 {
		t.Errorf("Expected OvertimeHours 2, got %.2f", summary.OvertimeHours)
	}
}

func TestCalculatePayroll(t *testing.T) {
	results := []DayResult{
		{Date: time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), IsAbsent: false, IsLate: false, TotalHours: 8, Overtime: 1},
		{Date: time.Date(2024, 1, 16, 0, 0, 0, 0, time.Local), IsAbsent: false, IsLate: false, TotalHours: 8, Overtime: 0},
		{Date: time.Date(2024, 1, 17, 0, 0, 0, 0, time.Local), IsAbsent: true, TotalHours: 0, Overtime: 0},
	}

	payroll := CalculatePayroll("001", "Test User", 16000.0, results, 1.5, 2.0, 3.0)

	if payroll.EmployeeNo != "001" {
		t.Errorf("Expected EmployeeNo 001, got %s", payroll.EmployeeNo)
	}
	if payroll.DaysWorked != 2 {
		t.Errorf("Expected DaysWorked 2, got %d", payroll.DaysWorked)
	}
	if payroll.DaysAbsent != 1 {
		t.Errorf("Expected DaysAbsent 1, got %d", payroll.DaysAbsent)
	}
	if payroll.OvertimeHours != 1 {
		t.Errorf("Expected OvertimeHours 1, got %.2f", payroll.OvertimeHours)
	}
}

func TestCalculatePayroll_WeekendOvertime(t *testing.T) {
	// Saturday (2024-01-13 is a Saturday)
	results := []DayResult{
		{Date: time.Date(2024, 1, 13, 0, 0, 0, 0, time.Local), IsAbsent: false, IsLate: false, TotalHours: 8, Overtime: 2, IsHoliday: false},
	}

	payroll := CalculatePayroll("001", "Test User", 16000.0, results, 1.5, 2.0, 3.0)

	// Weekend overtime should be double (2.0x)
	expectedOvertimePay := 2.0 * (16000.0 / 160.0) * 2.0 // 2 hours * hourly rate * 2.0
	if payroll.OvertimePay != expectedOvertimePay {
		t.Errorf("Expected OvertimePay %.2f, got %.2f", expectedOvertimePay, payroll.OvertimePay)
	}
}
