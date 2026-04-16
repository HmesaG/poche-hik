# Sección de Solicitud de Viáticos

Módulo completo de **solicitudes de viáticos** con una **tabla de tarifas configurable** que soporta dos modos de cálculo: **porcentual** (% del salario base) y **monto fijo** (RD$/día).

---

## Concepto: Tabla de Tarifas

El admin configura tarifas de viáticos desde Configuración. Cada tarifa tiene:

| Campo | Descripción | Ejemplo |
|-------|-------------|---------|
| Nombre | Identificador de la tarifa | "Interior Gerencial" |
| Tipo | `percentage` o `fixed` | `percentage` |
| Valor | % del salario diario o monto fijo/día | `25` (25%) ó `2500` (RD$) |

**Cálculo al crear solicitud:**
- **Porcentual:** [(salario_base / 23.83) × (valor / 100) × días_de_viaje](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/employees/models.go#64-73)
- **Monto fijo:** `valor × días_de_viaje`

> [!IMPORTANT]
> **23.83** es el divisor estándar dominicano para calcular el salario diario (365/12/30.44 ≈ 23.83). ¿Usas otro divisor en tu empresa? Si no, usaré este valor por defecto, configurable.

---

## Proposed Changes

### Backend — Modelos

#### [MODIFY] [models.go](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/employees/models.go)

Agregar dos structs:

```go
// TravelAllowanceRate — tarifa configurable para cálculo de viáticos
type TravelAllowanceRate struct {
    ID        string  `json:"id"`
    Name      string  `json:"name"`        // "Interior Gerencial", etc.
    Type      string  `json:"type"`        // "percentage" | "fixed"
    Value     float64 `json:"value"`       // % o monto fijo por día
    Active    bool    `json:"active"`
}

// TravelAllowance — solicitud individual de viático
type TravelAllowance struct {
    ID              string    `json:"id"`
    EmployeeID      string    `json:"employeeId"`
    EmployeeName    string    `json:"employeeName"`    // JOIN
    RateID          string    `json:"rateId"`          // tarifa aplicada
    RateName        string    `json:"rateName"`        // JOIN
    Destination     string    `json:"destination"`
    DepartureDate   time.Time `json:"departureDate"`
    ReturnDate      time.Time `json:"returnDate"`
    Days            int       `json:"days"`            // calculado
    Reason          string    `json:"reason"`
    CalculatedAmount float64  `json:"calculatedAmount"` // auto-calculado
    Status          string    `json:"status"`           // Pending|Approved|Rejected
    ApprovedBy      string    `json:"approvedBy"`
    ApprovalNotes   string    `json:"approvalNotes"`
    CreatedAt       time.Time `json:"createdAt"`
}
```

---

### Backend — SQLite

#### [MODIFY] [store.go](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/store/store.go)

Agregar a la interfaz [Repository](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/store/store.go#11-55):

```go
// Travel Allowance Rates
CreateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error
ListTravelRates(ctx context.Context) ([]*employees.TravelAllowanceRate, error)
UpdateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error
DeleteTravelRate(ctx context.Context, id string) error

// Travel Allowances
CreateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error
GetTravelAllowance(ctx context.Context, id string) (*employees.TravelAllowance, error)
ListTravelAllowances(ctx context.Context) ([]*employees.TravelAllowance, error)
UpdateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error
DeleteTravelAllowance(ctx context.Context, id string) error
```

#### [MODIFY] [sqlite.go](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/store/sqlite.go)

Dos tablas nuevas en [initSchema()](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/store/sqlite.go#30-114):

```sql
CREATE TABLE IF NOT EXISTS travel_allowance_rates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,     -- 'percentage' | 'fixed'
    value REAL NOT NULL,
    active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS travel_allowances (
    id TEXT PRIMARY KEY,
    employee_id TEXT NOT NULL,
    rate_id TEXT,
    destination TEXT NOT NULL,
    departure_date DATETIME NOT NULL,
    return_date DATETIME NOT NULL,
    days INTEGER NOT NULL,
    reason TEXT,
    calculated_amount REAL NOT NULL,
    status TEXT DEFAULT 'Pending',
    approved_by TEXT,
    approval_notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (employee_id) REFERENCES employees(id),
    FOREIGN KEY (rate_id) REFERENCES travel_allowance_rates(id)
);
```

Implementar los 9 métodos CRUD (4 para rates + 5 para allowances).

---

### Backend — API

#### [NEW] [handlers_travel.go](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/api/handlers_travel.go)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/travel-rates` | Listar tarifas |
| POST | `/api/travel-rates` | Crear tarifa |
| PUT | `/api/travel-rates/{id}` | Actualizar tarifa |
| DELETE | `/api/travel-rates/{id}` | Eliminar tarifa |
| GET | `/api/travel-allowances` | Listar solicitudes |
| POST | `/api/travel-allowances` | Crear solicitud (calcula monto) |
| PUT | `/api/travel-allowances/{id}` | Actualizar solicitud |
| DELETE | `/api/travel-allowances/{id}` | Eliminar solicitud |
| POST | `/api/travel-allowances/{id}/approve` | Aprobar |
| POST | `/api/travel-allowances/{id}/reject` | Rechazar |

**Lógica de cálculo en `handleCreateTravelAllowance`:**
1. Obtener la tarifa (`rate`) por su ID
2. Obtener el empleado para su `baseSalary`
3. Calcular días = `returnDate - departureDate + 1`
4. Si `rate.Type == "percentage"` → [(baseSalary / 23.83) × (rate.Value / 100) × days](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/employees/models.go#64-73)
5. Si `rate.Type == "fixed"` → `rate.Value × days`

#### [MODIFY] [router.go](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/internal/api/router.go)

Agregar rutas en el bloque protegido por JWT.

---

### Frontend — HTML

#### [MODIFY] [index.html](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/web/index.html)

1. **SVG:** Icono `icon-travel` en el sprite
2. **Sidebar:** Item "Viáticos" bajo Organización
3. **Página `<section id="travel-allowances">`:** Tabla de solicitudes + filtros por estado + botón nueva solicitud
4. **Modal `travel-modal`:** Formulario con selects de empleado y tarifa, campos de fecha, destino, motivo. Muestra el monto calculado en tiempo real al cambiar tarifa/fechas
5. **Configuración:** Tab nueva "Viáticos" en la página de settings para gestionar la tabla de tarifas (CRUD inline)

---

### Frontend — JavaScript & CSS

#### [MODIFY] [app.js](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/web/app.js)

- `initTravelAllowances()`, `loadTravelAllowances()`, `loadTravelRates()`
- Cálculo en tiempo real en el modal al seleccionar tarifa + empleado + fechas
- Aprobar/Rechazar con confirmación
- Admin CRUD de tarifas en settings

#### [MODIFY] [style.css](file:///c:/Users/Hector/Desktop/Grupo%20MV/Proyectos/Ponches/web/style.css)

Badges de estado (Pendiente=amarillo, Aprobado=verde, Rechazado=rojo), monto calculado destacado.

---

## Verification Plan

### Automated Tests

```bash
go test ./internal/store/ -v -run TestTravelAllowance
go test ./internal/store/ -v -run TestTravelRate
go build ./...
```

### Manual Verification

- Crear tarifas (porcentual y fija) en Configuración
- Crear solicitud → verificar cálculo automático
- Aprobar/Rechazar → verificar cambio de estado
- Filtrar por estado
