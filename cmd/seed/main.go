// cmd/seed/main.go — Carga de datos de prueba para Ponches
// Uso: go run ./cmd/seed
// Seguro de ejecutar múltiples veces (usa INSERT OR IGNORE).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"ponches/internal/config"
	"ponches/internal/employees"
	"ponches/internal/store"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	repo, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}

	ctx := context.Background()

	fmt.Println("🌱 Cargando datos de prueba...")

	seedDepartments(ctx, repo)
	seedPositions(ctx, repo)
	seedEmployees(ctx, repo)
	seedTravelRates(ctx, repo)
	seedTravelAllowances(ctx, repo)

	fmt.Println("\n✅ Datos de prueba cargados exitosamente.")
	os.Exit(0)
}

// ─── DEPARTAMENTOS ────────────────────────────────────────────────────────────

func seedDepartments(ctx context.Context, repo store.Repository) {
	depts := []*employees.Department{
		{ID: "dept-ventas", Name: "Ventas y Comercial", Description: "Gestión de clientes, ventas y desarrollo de negocios."},
		{ID: "dept-ti", Name: "Tecnología e IT", Description: "Infraestructura, sistemas y soporte técnico."},
		{ID: "dept-admin", Name: "Administración", Description: "Recursos humanos, finanzas y operaciones internas."},
		{ID: "dept-logistica", Name: "Logística y Operaciones", Description: "Distribución, almacén y cadena de suministro."},
	}

	count := 0
	for _, d := range depts {
		existing, _ := repo.GetDepartment(ctx, d.ID)
		if existing != nil {
			continue
		}
		if err := repo.CreateDepartment(ctx, d); err != nil {
			fmt.Printf("  ⚠ Departamento %s: %v\n", d.Name, err)
		} else {
			count++
			fmt.Printf("  ✓ Departamento: %s\n", d.Name)
		}
	}
	fmt.Printf("  → %d departamentos insertados\n", count)
}

// ─── CARGOS / POSICIONES ─────────────────────────────────────────────────────

func seedPositions(ctx context.Context, repo store.Repository) {
	positions := []*employees.Position{
		{ID: "pos-gerente-ventas", Name: "Gerente de Ventas", DepartmentID: "dept-ventas", Level: 5},
		{ID: "pos-ejecutivo-cuenta", Name: "Ejecutivo de Cuenta", DepartmentID: "dept-ventas", Level: 3},
		{ID: "pos-dev-senior", Name: "Desarrollador Senior", DepartmentID: "dept-ti", Level: 4},
		{ID: "pos-soporte-ti", Name: "Técnico de Soporte IT", DepartmentID: "dept-ti", Level: 2},
		{ID: "pos-rrhh", Name: "Coordinador de RRHH", DepartmentID: "dept-admin", Level: 3},
		{ID: "pos-contador", Name: "Contador General", DepartmentID: "dept-admin", Level: 4},
		{ID: "pos-conductor", Name: "Conductor / Mensajero", DepartmentID: "dept-logistica", Level: 1},
		{ID: "pos-supervisor-log", Name: "Supervisor de Logística", DepartmentID: "dept-logistica", Level: 4},
	}

	count := 0
	for _, p := range positions {
		existing, _ := repo.GetPosition(ctx, p.ID)
		if existing != nil {
			continue
		}
		if err := repo.CreatePosition(ctx, p); err != nil {
			fmt.Printf("  ⚠ Cargo %s: %v\n", p.Name, err)
		} else {
			count++
			fmt.Printf("  ✓ Cargo: %s\n", p.Name)
		}
	}
	fmt.Printf("  → %d cargos insertados\n", count)
}

// ─── EMPLEADOS ───────────────────────────────────────────────────────────────

func seedEmployees(ctx context.Context, repo store.Repository) {
	now := time.Now()

	emps := []*employees.Employee{
		{
			ID: "emp-001", EmployeeNo: "101",
			FirstName: "Carlos", LastName: "Mendoza",
			IDNumber: "001-1234567-8", Gender: "M",
			BirthDate: mustDate("1988-04-15"), HireDate: mustDate("2018-03-01"),
			Email: "cmendoza@empresa.com", Phone: "849-123-0001",
			DepartmentID: "dept-ventas", PositionID: "pos-gerente-ventas",
			BaseSalary: 95000, Status: "Active",
		},
		{
			ID: "emp-002", EmployeeNo: "102",
			FirstName: "María", LastName: "Rodríguez",
			IDNumber: "001-2345678-9", Gender: "F",
			BirthDate: mustDate("1992-07-22"), HireDate: mustDate("2020-01-15"),
			Email: "mrodriguez@empresa.com", Phone: "849-123-0002",
			DepartmentID: "dept-ventas", PositionID: "pos-ejecutivo-cuenta",
			BaseSalary: 55000, Status: "Active",
		},
		{
			ID: "emp-003", EmployeeNo: "103",
			FirstName: "Luis", LastName: "García",
			IDNumber: "001-3456789-0", Gender: "M",
			BirthDate: mustDate("1990-11-08"), HireDate: mustDate("2019-06-01"),
			Email: "lgarcia@empresa.com", Phone: "849-123-0003",
			DepartmentID: "dept-ti", PositionID: "pos-dev-senior",
			BaseSalary: 85000, Status: "Active",
		},
		{
			ID: "emp-004", EmployeeNo: "104",
			FirstName: "Ana", LastName: "Martínez",
			IDNumber: "002-4567890-1", Gender: "F",
			BirthDate: mustDate("1995-02-14"), HireDate: mustDate("2022-08-10"),
			Email: "amartinez@empresa.com", Phone: "849-123-0004",
			DepartmentID: "dept-ti", PositionID: "pos-soporte-ti",
			BaseSalary: 45000, Status: "Active",
		},
		{
			ID: "emp-005", EmployeeNo: "105",
			FirstName: "Sofía", LastName: "Jiménez",
			IDNumber: "002-5678901-2", Gender: "F",
			BirthDate: mustDate("1987-09-30"), HireDate: mustDate("2016-04-01"),
			Email: "sjimenez@empresa.com", Phone: "849-123-0005",
			DepartmentID: "dept-admin", PositionID: "pos-rrhh",
			BaseSalary: 62000, Status: "Active",
		},
		{
			ID: "emp-006", EmployeeNo: "106",
			FirstName: "Pedro", LastName: "López",
			IDNumber: "002-6789012-3", Gender: "M",
			BirthDate: mustDate("1983-12-05"), HireDate: mustDate("2015-01-20"),
			Email: "plopez@empresa.com", Phone: "849-123-0006",
			DepartmentID: "dept-admin", PositionID: "pos-contador",
			BaseSalary: 78000, Status: "Active",
		},
		{
			ID: "emp-007", EmployeeNo: "107",
			FirstName: "Jean", LastName: "Reyes",
			IDNumber: "003-7890123-4", Gender: "M",
			BirthDate: mustDate("1997-06-19"), HireDate: mustDate("2023-02-01"),
			Email: "jreyes@empresa.com", Phone: "849-123-0007",
			DepartmentID: "dept-logistica", PositionID: "pos-conductor",
			BaseSalary: 32000, Status: "Active",
		},
		{
			ID: "emp-008", EmployeeNo: "108",
			FirstName: "Carmen", LastName: "Herrera",
			IDNumber: "003-8901234-5", Gender: "F",
			BirthDate: mustDate("1982-03-27"), HireDate: mustDate("2014-09-15"),
			Email: "cherrera@empresa.com", Phone: "849-123-0008",
			DepartmentID: "dept-logistica", PositionID: "pos-supervisor-log",
			BaseSalary: 72000, Status: "Active",
		},
		{
			ID: "emp-009", EmployeeNo: "109",
			FirstName: "Roberto", LastName: "Santos",
			IDNumber: "003-9012345-6", Gender: "M",
			BirthDate: mustDate("1994-08-11"), HireDate: mustDate("2021-05-03"),
			Email: "rsantos@empresa.com", Phone: "849-123-0009",
			DepartmentID: "dept-ventas", PositionID: "pos-ejecutivo-cuenta",
			BaseSalary: 55000, Status: "Active",
		},
		{
			ID: "emp-010", EmployeeNo: "110",
			FirstName: "Laura", LastName: "Díaz",
			IDNumber: "004-0123456-7", Gender: "F",
			BirthDate: mustDate("1991-01-25"), HireDate: mustDate("2020-11-01"),
			Email: "ldiaz@empresa.com", Phone: "849-123-0010",
			DepartmentID: "dept-ti", PositionID: "pos-dev-senior",
			BaseSalary: 82000, Status: "Active",
		},
		{
			ID: "emp-011", EmployeeNo: "111",
			FirstName: "Miguel", LastName: "Fernández",
			IDNumber: "004-1234567-8", Gender: "M",
			BirthDate: mustDate("1986-05-16"), HireDate: mustDate("2017-07-01"),
			Email: "mfernandez@empresa.com", Phone: "849-123-0011",
			DepartmentID: "dept-admin", PositionID: "pos-rrhh",
			BaseSalary: 58000, Status: "Active",
		},
		{
			ID: "emp-012", EmployeeNo: "112",
			FirstName: "Valentina", LastName: "Pérez",
			IDNumber: "004-2345678-9", Gender: "F",
			BirthDate: mustDate("1999-10-03"), HireDate: mustDate("2024-01-08"),
			Email: "vperez@empresa.com", Phone: "849-123-0012",
			DepartmentID: "dept-ventas", PositionID: "pos-ejecutivo-cuenta",
			BaseSalary: 48000, Status: "Active",
		},
	}

	_ = now // unused but kept for reference

	count := 0
	for _, e := range emps {
		existing, _ := repo.GetEmployee(ctx, e.ID)
		if existing != nil {
			continue
		}
		if err := repo.CreateEmployee(ctx, e); err != nil {
			fmt.Printf("  ⚠ Empleado %s %s: %v\n", e.FirstName, e.LastName, err)
		} else {
			count++
			fmt.Printf("  ✓ Empleado: %s %s (No. %s) — RD$ %.0f\n", e.FirstName, e.LastName, e.EmployeeNo, e.BaseSalary)
		}
	}
	fmt.Printf("  → %d empleados insertados\n", count)
}

// ─── TARIFAS DE VIÁTICOS ─────────────────────────────────────────────────────

func seedTravelRates(ctx context.Context, repo store.Repository) {
	rates := []*employees.TravelAllowanceRate{
		{
			ID:     "rate-interior-fijo",
			Name:   "Viaje Interior (Fijo)",
			Type:   "fixed",
			Value:  2500,
			Active: true,
		},
		{
			ID:     "rate-exterior-fijo",
			Name:   "Viaje Exterior (Fijo)",
			Type:   "fixed",
			Value:  6500,
			Active: true,
		},
		{
			ID:     "rate-gerencial-pct",
			Name:   "Gerencial Porcentual",
			Type:   "percentage",
			Value:  30,
			Active: true,
		},
		{
			ID:     "rate-operativo-pct",
			Name:   "Operativo Porcentual",
			Type:   "percentage",
			Value:  15,
			Active: true,
		},
	}

	count := 0
	for _, r := range rates {
		existing, _ := repo.GetTravelRate(ctx, r.ID)
		if existing != nil {
			continue
		}
		if err := repo.CreateTravelRate(ctx, r); err != nil {
			fmt.Printf("  ⚠ Tarifa %s: %v\n", r.Name, err)
		} else {
			count++
			typeLabel := "fijo"
			if r.Type == "percentage" {
				typeLabel = fmt.Sprintf("%.0f%% salario diario", r.Value)
			} else {
				typeLabel = fmt.Sprintf("RD$ %.0f/día", r.Value)
			}
			fmt.Printf("  ✓ Tarifa: %s (%s)\n", r.Name, typeLabel)
		}
	}
	fmt.Printf("  → %d tarifas insertadas\n", count)
}

// ─── SOLICITUDES DE VIÁTICOS ─────────────────────────────────────────────────

func seedTravelAllowances(ctx context.Context, repo store.Repository) {
	// Helper: calcular monto
	calcAmount := func(rate *employees.TravelAllowanceRate, salary float64, days int) float64 {
		const divisor = 23.83
		var amount float64
		if rate.Type == "percentage" {
			amount = (salary / divisor) * (rate.Value / 100) * float64(days)
		} else {
			amount = rate.Value * float64(days)
		}
		return float64(int(amount*100)) / 100
	}

	type taData struct {
		id          string
		empID       string
		rateID      string
		destination string
		departure   string
		returnDate  string
		reason      string
		status      string
		notes       string
	}

	records := []taData{
		{"ta-001", "emp-001", "rate-exterior-fijo", "Miami, Estados Unidos",
			"2026-03-05", "2026-03-08", "Feria internacional de tecnología y ventas B2B.", "Approved", "Aprobado por dirección general."},
		{"ta-002", "emp-003", "rate-gerencial-pct", "Ciudad de México, México",
			"2026-03-12", "2026-03-14", "Conferencia de desarrollo de software y arquitectura cloud.", "Approved", "Misión estratégica IT."},
		{"ta-003", "emp-005", "rate-interior-fijo", "Santiago, República Dominicana",
			"2026-03-18", "2026-03-19", "Visita a clientes y renovación de contratos.", "Approved", ""},
		{"ta-004", "emp-002", "rate-operativo-pct", "La Romana, República Dominicana",
			"2026-03-20", "2026-03-21", "Reunión de seguimiento con distribuidor zonal.", "Pending", ""},
		{"ta-005", "emp-006", "rate-interior-fijo", "Puerto Plata, República Dominicana",
			"2026-03-22", "2026-03-23", "Auditoría y cierre de balances trimestrales.", "Pending", ""},
		{"ta-006", "emp-010", "rate-gerencial-pct", "Bogotá, Colombia",
			"2026-04-02", "2026-04-05", "Capacitación en metodologías ágiles y DevOps.", "Pending", ""},
		{"ta-007", "emp-009", "rate-operativo-pct", "San Francisco de Macorís",
			"2026-04-07", "2026-04-08", "Prospección de nuevos clientes en la región.", "Pending", ""},
		{"ta-008", "emp-008", "rate-interior-fijo", "Higüey, República Dominicana",
			"2026-02-10", "2026-02-11", "Supervisión de operaciones de distribución en el Este.", "Rejected", "No se pudo verificar el presupuesto disponible."},
		{"ta-009", "emp-011", "rate-interior-fijo", "La Vega, República Dominicana",
			"2026-02-18", "2026-02-19", "Capacitación interna a personal administrativo.", "Approved", ""},
		{"ta-010", "emp-004", "rate-interior-fijo", "Barahona, República Dominicana",
			"2026-05-05", "2026-05-07", "Instalación y configuración de equipos de red en nueva sucursal.", "Pending", ""},
		{"ta-011", "emp-001", "rate-exterior-fijo", "Madrid, España",
			"2026-05-12", "2026-05-16", "Negociaciones con proveedor estratégico europeo.", "Pending", ""},
		{"ta-012", "emp-007", "rate-interior-fijo", "Monte Cristi, República Dominicana",
			"2026-03-25", "2026-03-25", "Entrega urgente de documentación en zona fronteriza.", "Pending", ""},
	}

	// Cache rates for calculation
	rateMap := map[string]*employees.TravelAllowanceRate{}
	for _, rID := range []string{"rate-interior-fijo", "rate-exterior-fijo", "rate-gerencial-pct", "rate-operativo-pct"} {
		r, _ := repo.GetTravelRate(ctx, rID)
		if r != nil {
			rateMap[rID] = r
		}
	}

	// Cache employees for salary lookup
	empMap := map[string]*employees.Employee{}
	allEmps, _ := repo.ListEmployees(ctx)
	for _, e := range allEmps {
		empMap[e.ID] = e
	}

	count := 0
	for _, rec := range records {
		existing, _ := repo.GetTravelAllowance(ctx, rec.id)
		if existing != nil {
			continue
		}

		dep := mustDate(rec.departure)
		ret := mustDate(rec.returnDate)
		days := int(ret.Sub(dep).Hours()/24) + 1
		if days < 1 {
			days = 1
		}

		rate := rateMap[rec.rateID]
		emp := empMap[rec.empID]

		if rate == nil || emp == nil {
			fmt.Printf("  ⚠ Saltando %s — tarifa o empleado no encontrado\n", rec.id)
			continue
		}

		amount := calcAmount(rate, emp.BaseSalary, days)

		ta := &employees.TravelAllowance{
			ID:               rec.id,
			EmployeeID:       rec.empID,
			RateID:           rec.rateID,
			Destination:      rec.destination,
			DepartureDate:    dep,
			ReturnDate:       ret,
			Days:             days,
			Reason:           rec.reason,
			CalculatedAmount: amount,
			Status:           rec.status,
			ApprovalNotes:    rec.notes,
		}

		if err := repo.CreateTravelAllowance(ctx, ta); err != nil {
			fmt.Printf("  ⚠ Viático %s: %v\n", rec.id, err)
		} else {
			count++
			fmt.Printf("  ✓ Viático: %s %s → %s (%d días, RD$ %.2f) [%s]\n",
				emp.FirstName, emp.LastName, rec.destination, days, amount, rec.status)
		}
	}
	fmt.Printf("  → %d solicitudes de viáticos insertadas\n", count)
}

// ─── HELPERS ──────────────────────────────────────────────────────────────────

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		log.Fatalf("invalid date %q: %v", s, err)
	}
	return t
}
