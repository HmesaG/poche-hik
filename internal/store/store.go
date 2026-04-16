package store

import (
	"context"
	"time"
	"ponches/internal/employees"
	"ponches/internal/users"
)

// Repository defines the interface for data persistence
type Repository interface {
	// Employees
	CreateEmployee(ctx context.Context, e *employees.Employee) error
	GetEmployee(ctx context.Context, id string) (*employees.Employee, error)
	GetEmployeeByNo(ctx context.Context, employeeNo string) (*employees.Employee, error)
	ListEmployees(ctx context.Context) ([]*employees.Employee, error)
	UpdateEmployee(ctx context.Context, e *employees.Employee) error
	DeleteEmployee(ctx context.Context, id string) error
	UpsertEmployee(ctx context.Context, e *employees.Employee) error

	// Departments & Positions
	CreateDepartment(ctx context.Context, d *employees.Department) error
	GetDepartment(ctx context.Context, id string) (*employees.Department, error)
	ListDepartments(ctx context.Context) ([]*employees.Department, error)
	UpdateDepartment(ctx context.Context, d *employees.Department) error
	DeleteDepartment(ctx context.Context, id string) error
	UpsertDepartment(ctx context.Context, d *employees.Department) error

	CreatePosition(ctx context.Context, p *employees.Position) error
	GetPosition(ctx context.Context, id string) (*employees.Position, error)
	ListPositions(ctx context.Context) ([]*employees.Position, error)
	UpdatePosition(ctx context.Context, p *employees.Position) error
	DeletePosition(ctx context.Context, id string) error
	UpsertPosition(ctx context.Context, p *employees.Position) error

	// Users
	CreateUser(ctx context.Context, u *users.User) error
	GetUser(ctx context.Context, id string) (*users.User, error)
	GetUserByID(ctx context.Context, id string) (*users.User, error)
	GetUserByUsername(ctx context.Context, username string) (*users.User, error)
	ListUsers(ctx context.Context) ([]*users.User, error)
	UpdateUser(ctx context.Context, u *users.User) error
	DeleteUser(ctx context.Context, id string) error

	// Configuration
	GetConfigValue(ctx context.Context, key string) (string, error)
	SetConfigValue(ctx context.Context, key, value string) error
	GetAllConfig(ctx context.Context) (map[string]string, error)
	SetMultipleConfigValues(ctx context.Context, values map[string]string) error

	// Attendance Events
	SaveEvent(ctx context.Context, event *AttendanceEvent) error
	GetEvents(ctx context.Context, filter EventFilter) ([]*AttendanceEvent, error)

	// Travel Allowance Rates
	CreateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error
	GetTravelRate(ctx context.Context, id string) (*employees.TravelAllowanceRate, error)
	ListTravelRates(ctx context.Context) ([]*employees.TravelAllowanceRate, error)
	UpdateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error
	DeleteTravelRate(ctx context.Context, id string) error

	// Travel Allowances
	CreateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error
	GetTravelAllowance(ctx context.Context, id string) (*employees.TravelAllowance, error)
	ListTravelAllowances(ctx context.Context) ([]*employees.TravelAllowance, error)
	UpdateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error
	DeleteTravelAllowance(ctx context.Context, id string) error

	// Leaves (Permisos y Ausencias)
	CreateLeave(ctx context.Context, l *employees.Leave) error
	GetLeave(ctx context.Context, id string) (*employees.Leave, error)
	ListLeaves(ctx context.Context) ([]*employees.Leave, error)
	ListLeavesByEmployee(ctx context.Context, employeeID string) ([]*employees.Leave, error)
	UpdateLeave(ctx context.Context, l *employees.Leave) error
	DeleteLeave(ctx context.Context, id string) error
}

type AttendanceEvent struct {
	ID         int64     `json:"id"`
	DeviceID   string    `json:"deviceId"`
	EmployeeNo string    `json:"employeeNo"`
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"` // Access, Alert, etc.
}

type EventFilter struct {
	EmployeeNo string
	From       time.Time
	To         time.Time
}
