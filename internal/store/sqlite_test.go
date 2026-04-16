package store

import (
	"context"
	"os"
	"testing"
	"time"

	"ponches/internal/employees"
)

func setupTestStore(t *testing.T) (*SQLiteStore, func()) {
	// Create temp DB file
	tmpFile := "test_ponches_" + time.Now().Format("20060102150405") + ".db"
	store, err := NewSQLiteStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	cleanup := func() {
		store.db.Close()
		os.Remove(tmpFile)
	}

	return store, cleanup
}

func TestCreateEmployee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	emp := &employees.Employee{
		ID:           "test-emp-1",
		EmployeeNo:   "001",
		FirstName:    "Juan",
		LastName:     "Pérez",
		IDNumber:     "123456789",
		Gender:       "M",
		BirthDate:    now.AddDate(-30, 0, 0),
		Phone:        "555-1234",
		Email:        "juan@example.com",
		DepartmentID: "dept-1",
		PositionID:   "pos-1",
		HireDate:     now,
		Status:       "Active",
		BaseSalary:   15000.00,
	}

	err := store.CreateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("CreateEmployee failed: %v", err)
	}

	// Verify employee was created
	retrieved, err := store.GetEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("GetEmployee failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetEmployee returned nil")
	}
	if retrieved.EmployeeNo != "001" {
		t.Errorf("Expected EmployeeNo 001, got %s", retrieved.EmployeeNo)
	}
	if retrieved.FirstName != "Juan" {
		t.Errorf("Expected FirstName Juan, got %s", retrieved.FirstName)
	}
}

func TestGetEmployeeByNo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	emp := &employees.Employee{
		ID:         "test-emp-2",
		EmployeeNo: "002",
		FirstName:  "Maria",
		LastName:   "García",
		BirthDate:  now.AddDate(-25, 0, 0),
		HireDate:   now,
		Status:     "Active",
		BaseSalary: 18000.00,
	}

	err := store.CreateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("CreateEmployee failed: %v", err)
	}

	// Test GetEmployeeByNo
	retrieved, err := store.GetEmployeeByNo(ctx, "002")
	if err != nil {
		t.Fatalf("GetEmployeeByNo failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetEmployeeByNo returned nil")
	}
	if retrieved.LastName != "García" {
		t.Errorf("Expected LastName García, got %s", retrieved.LastName)
	}

	// Test non-existent employee
	notFound, err := store.GetEmployeeByNo(ctx, "999")
	if err != nil {
		t.Fatalf("GetEmployeeByNo failed for non-existent: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent employee")
	}
}

func TestUpdateEmployee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	emp := &employees.Employee{
		ID:         "test-emp-3",
		EmployeeNo: "003",
		FirstName:  "Carlos",
		LastName:   "López",
		BirthDate:  now.AddDate(-35, 0, 0),
		HireDate:   now,
		Status:     "Active",
		BaseSalary: 20000.00,
	}

	err := store.CreateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("CreateEmployee failed: %v", err)
	}

	// Update employee
	emp.FirstName = "Carlos Alberto"
	emp.BaseSalary = 22000.00
	emp.Status = "Inactive"

	err = store.UpdateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("UpdateEmployee failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("GetEmployee failed: %v", err)
	}
	if retrieved.FirstName != "Carlos Alberto" {
		t.Errorf("Expected FirstName Carlos Alberto, got %s", retrieved.FirstName)
	}
	if retrieved.Status != "Inactive" {
		t.Errorf("Expected Status Inactive, got %s", retrieved.Status)
	}
	if retrieved.BaseSalary != 22000.00 {
		t.Errorf("Expected BaseSalary 22000, got %f", retrieved.BaseSalary)
	}

	emp.EmployeeNo = "003A"
	err = store.UpdateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("UpdateEmployee employee number failed: %v", err)
	}

	retrieved, err = store.GetEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("GetEmployee after employee number update failed: %v", err)
	}
	if retrieved.EmployeeNo != "003A" {
		t.Errorf("Expected EmployeeNo 003A, got %s", retrieved.EmployeeNo)
	}
}

func TestListEmployees(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Create multiple employees
	employees := []*employees.Employee{
		{ID: "emp-1", EmployeeNo: "001", FirstName: "Ana", LastName: "Díaz", BirthDate: now.AddDate(-28, 0, 0), HireDate: now, Status: "Active"},
		{ID: "emp-2", EmployeeNo: "002", FirstName: "Beto", LastName: "Ruiz", BirthDate: now.AddDate(-32, 0, 0), HireDate: now, Status: "Active"},
		{ID: "emp-3", EmployeeNo: "003", FirstName: "Carla", LastName: "Méndez", BirthDate: now.AddDate(-29, 0, 0), HireDate: now, Status: "Inactive"},
	}

	for _, emp := range employees {
		err := store.CreateEmployee(ctx, emp)
		if err != nil {
			t.Fatalf("CreateEmployee failed: %v", err)
		}
	}

	// List employees
	list, err := store.ListEmployees(ctx)
	if err != nil {
		t.Fatalf("ListEmployees failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 employees, got %d", len(list))
	}
}

func TestDeleteEmployee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	emp := &employees.Employee{
		ID:         "test-emp-del",
		EmployeeNo: "099",
		FirstName:  "Delete",
		LastName:   "Me",
		BirthDate:  now.AddDate(-30, 0, 0),
		HireDate:   now,
		Status:     "Active",
	}

	err := store.CreateEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("CreateEmployee failed: %v", err)
	}

	// Delete employee
	err = store.DeleteEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("DeleteEmployee failed: %v", err)
	}

	// Verify deletion
	retrieved, err := store.GetEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("GetEmployee failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil after deletion")
	}
}

func TestDepartmentCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create
	dept := &employees.Department{
		ID:          "dept-test",
		Name:        "Tecnología",
		Description: "Infraestructura y soporte",
		ParentID:    "",
	}
	err := store.CreateDepartment(ctx, dept)
	if err != nil {
		t.Fatalf("CreateDepartment failed: %v", err)
	}

	// Get
	retrieved, err := store.GetDepartment(ctx, dept.ID)
	if err != nil {
		t.Fatalf("GetDepartment failed: %v", err)
	}
	if retrieved.Description != "Infraestructura y soporte" {
		t.Errorf("Expected department description, got %q", retrieved.Description)
	}
	if retrieved == nil || retrieved.Name != "Tecnología" {
		t.Errorf("Expected department name Tecnología")
	}

	// Update
	dept.Name = "TI y Sistemas"
	dept.Description = "Operaciones tecnicas"
	err = store.UpdateDepartment(ctx, dept)
	if err != nil {
		t.Fatalf("UpdateDepartment failed: %v", err)
	}

	// Verify update
	updated, _ := store.GetDepartment(ctx, dept.ID)
	if updated.Name != "TI y Sistemas" {
		t.Errorf("Expected updated name TI y Sistemas, got %s", updated.Name)
	}
	if updated.Description != "Operaciones tecnicas" {
		t.Errorf("Expected updated description, got %q", updated.Description)
	}

	// List
	list, err := store.ListDepartments(ctx)
	if err != nil {
		t.Fatalf("ListDepartments failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 department, got %d", len(list))
	}

	// Delete
	err = store.DeleteDepartment(ctx, dept.ID)
	if err != nil {
		t.Fatalf("DeleteDepartment failed: %v", err)
	}
}

func TestPositionCRUD(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create department first
	dept := &employees.Department{ID: "dept-pos", Name: "Ventas"}
	store.CreateDepartment(ctx, dept)

	// Create position
	pos := &employees.Position{
		ID:           "pos-test",
		Name:         "Gerente",
		DepartmentID: dept.ID,
		Level:        5,
	}
	err := store.CreatePosition(ctx, pos)
	if err != nil {
		t.Fatalf("CreatePosition failed: %v", err)
	}

	// Get
	retrieved, err := store.GetPosition(ctx, pos.ID)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if retrieved == nil || retrieved.Name != "Gerente" {
		t.Errorf("Expected position name Gerente")
	}
	if retrieved.Level != 5 {
		t.Errorf("Expected level 5, got %d", retrieved.Level)
	}

	// Update
	pos.Name = "Gerente Senior"
	pos.Level = 7
	err = store.UpdatePosition(ctx, pos)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	// Verify update
	updated, _ := store.GetPosition(ctx, pos.ID)
	if updated.Name != "Gerente Senior" {
		t.Errorf("Expected updated name Gerente Senior")
	}

	// Delete
	err = store.DeletePosition(ctx, pos.ID)
	if err != nil {
		t.Fatalf("DeletePosition failed: %v", err)
	}
}

func TestUpsertEmployee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	emp := &employees.Employee{
		ID:           "test-upsert",
		EmployeeNo:   "100",
		FirstName:    "Initial",
		LastName:     "Name",
		BirthDate:    now.AddDate(-30, 0, 0),
		HireDate:     now,
		Status:       "Active",
		DepartmentID: "dept-1",
		PositionID:   "pos-1",
		Email:        "initial@example.com",
	}

	// Create
	err := store.UpsertEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("UpsertEmployee (create) failed: %v", err)
	}

	// Update (same ID)
	emp.FirstName = "Updated"
	emp.Email = "updated@example.com"
	err = store.UpsertEmployee(ctx, emp)
	if err != nil {
		t.Fatalf("UpsertEmployee (update) failed: %v", err)
	}

	// Verify
	retrieved, err := store.GetEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("GetEmployee failed: %v", err)
	}
	if retrieved.FirstName != "Updated" {
		t.Errorf("Expected FirstName Updated, got %s", retrieved.FirstName)
	}
	if retrieved.Email != "updated@example.com" {
		t.Errorf("Expected Email updated@example.com, got %s", retrieved.Email)
	}
}

func TestSaveAndGetEvents(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	// Save events
	events := []*AttendanceEvent{
		{DeviceID: "dev-1", EmployeeNo: "001", Timestamp: now, Type: "Access"},
		{DeviceID: "dev-1", EmployeeNo: "001", Timestamp: now.Add(-1 * time.Hour), Type: "Access"},
		{DeviceID: "dev-2", EmployeeNo: "002", Timestamp: now, Type: "Alert"},
	}

	for _, ev := range events {
		err := store.SaveEvent(ctx, ev)
		if err != nil {
			t.Fatalf("SaveEvent failed: %v", err)
		}
	}

	// Get all events
	all, err := store.GetEvents(ctx, EventFilter{})
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 events, got %d", len(all))
	}

	// Filter by employee
	filtered, err := store.GetEvents(ctx, EventFilter{EmployeeNo: "001"})
	if err != nil {
		t.Fatalf("GetEvents filtered failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("Expected 2 events for employee 001, got %d", len(filtered))
	}
}
