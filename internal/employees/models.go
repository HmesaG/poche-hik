package employees

import (
	"time"
)

type Employee struct {
	ID         string    `json:"id"`
	EmployeeNo string    `json:"employeeNo"` // Linked to Hikvision
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	IDNumber   string    `json:"idNumber"` // Cédula/DNI
	Gender     string    `json:"gender"`
	BirthDate  time.Time `json:"birthDate"`
	PhotoURL   string    `json:"photoUrl"`

	// Contact
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`

	// Laboral
	DepartmentID string    `json:"departmentId"`
	PositionID   string    `json:"positionId"`
	HireDate     time.Time `json:"hireDate"`
	Status       string    `json:"status"` // Active, Inactive, etc.
	BaseSalary   float64   `json:"baseSalary"`
	FaceID       string    `json:"faceId"`     // URL or ID of the registered face
	FleetNo      string    `json:"fleetNo"`    // Número de flota
	PersonalNo   string    `json:"personalNo"` // Número personal interno

	// Relationships
	EmergencyContacts []EmergencyContact `json:"emergencyContacts"`
	Contracts         []Contract         `json:"contracts"`
}

type EmergencyContact struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	Phone        string `json:"phone"`
}

type Department struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parentId"`
	ManagerID   string `json:"managerId"`
	ManagerName string `json:"managerName,omitempty"` // populated via JOIN
}

type Position struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DepartmentID string `json:"departmentId"`
	Level        int    `json:"level"`
}

type Contract struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"` // Indefinite, Fixed-term, Probation
	StartDate time.Time  `json:"startDate"`
	EndDate   *time.Time `json:"endDate"`
	Salary    float64    `json:"salary"`
	FileID    string     `json:"fileId"`
}

// LeaveType defines the category of absence
type LeaveType string

const (
	LeaveVacation   LeaveType = "Vacation"   // Vacaciones
	LeavePermission LeaveType = "Permission" // Permiso autorizado por supervisor
	LeaveSick       LeaveType = "Sick"       // Enfermedad
	LeavePersonal   LeaveType = "Personal"   // Personal
	LeaveOther      LeaveType = "Other"      // Otro
)

// Leave represents an approved absence or permission record
type Leave struct {
	ID             string    `json:"id"`
	EmployeeID     string    `json:"employeeId"`
	EmployeeName   string    `json:"employeeName,omitempty"` // via JOIN
	EmployeeNo     string    `json:"employeeNo,omitempty"`   // via JOIN
	Department     string    `json:"department,omitempty"`   // via JOIN
	Type           LeaveType `json:"type"`
	StartDate      time.Time `json:"startDate"`
	EndDate        time.Time `json:"endDate"`
	Days           int       `json:"days"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"` // Pending, Approved, Rejected
	AuthorizedBy   string    `json:"authorizedBy,omitempty"`
	AuthorizerName string    `json:"authorizerName,omitempty"` // via JOIN
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TravelAllowanceRate defines a configurable per-diem rate for travel allowance calculations.
// Type "percentage" computes from the employee's daily salary; "fixed" uses a flat amount per day.
type TravelAllowanceRate struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`  // "percentage" | "fixed"
	Value     float64 `json:"value"` // % of daily salary, or flat RD$/day
	Active    bool    `json:"active"`
	CreatedAt string  `json:"createdAt,omitempty"`
}

// TravelAllowance represents a single travel allowance request with automatic amount calculation.
type TravelAllowance struct {
	ID               string    `json:"id"`
	EmployeeID       string    `json:"employeeId"`
	EmployeeName     string    `json:"employeeName,omitempty"` // populated via JOIN
	EmployeeIDs      []string  `json:"employeeIds,omitempty"`
	RateID           string    `json:"rateId"`
	RateName         string    `json:"rateName,omitempty"` // populated via JOIN
	RateType         string    `json:"rateType,omitempty"` // populated via JOIN
	Destination      string    `json:"destination"`
	DepartureDate    time.Time `json:"departureDate"`
	ReturnDate       time.Time `json:"returnDate"`
	Days             int       `json:"days"`
	Reason           string    `json:"reason"`
	CalculatedAmount float64   `json:"calculatedAmount"`
	Status           string    `json:"status"` // Pending, Approved, Rejected
	ApprovedBy       string    `json:"approvedBy,omitempty"`
	ApproverName     string    `json:"approverName,omitempty"` // populated via JOIN
	ApprovalNotes    string    `json:"approvalNotes,omitempty"`
	GroupID          string    `json:"groupId,omitempty"`
	GroupName        string    `json:"groupName,omitempty"`
	GroupSize        int       `json:"groupSize,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
