# Documentación Técnica — Ponches v2.0

> Sistema de Control de Asistencia Empresarial con Biometría Hikvision  
> **Empresa:** Grupo MV | **Plataforma:** Windows (red corporativa local)  
> **Estado:** ✅ Producción local (Fases 1-6 completadas)

---

## 1. Resumen Ejecutivo

**Ponches** es un sistema integral de control de asistencia que se integra con terminales biométricas Hikvision para registrar entradas/salidas de empleados mediante reconocimiento facial. Calcula horas extras, tardanzas, genera reportes de nómina en PDF/Excel, y gestiona la estructura organizacional completa.

### Capacidades Principales
- Registro biométrico facial vía dispositivos Hikvision ISAPI
- Cálculo automático de asistencia, horas extras y tardanzas
- Reportes de nómina, asistencia diaria y KPIs en PDF y Excel
- Gestión de permisos/ausencias y viáticos
- Dashboard en tiempo real con WebSocket
- Sincronización LDAP/Active Directory
- Progressive Web App (PWA) instalable
- Directorio público de empleados con vCards

---

## 2. Stack Tecnológico

| Capa | Tecnología | Versión |
|------|-----------|---------|
| Lenguaje Backend | Go | 1.25 |
| Router HTTP | Chi | v5.2.5 |
| Base de Datos | SQLite (embebida) | modernc.org/sqlite v1.46.1 |
| Frontend | Vanilla JS + CSS | SPA monolítica |
| WebSocket | gorilla/websocket | v1.5.3 |
| Reportes PDF | go-pdf/fpdf | v0.9.0 |
| Reportes Excel | xuri/excelize | v2.10.1 |
| Autenticación | golang-jwt/v5 + bcrypt | v5.3.1 |
| Logging | zerolog | v1.32.0 |
| Config | godotenv | v1.5.1 |
| IDs | google/uuid | v1.6.0 |
| LDAP | go-ldap/v3 | v3.4.13 |
| Hardware | Hikvision ISAPI | Digest Auth (MD5) |

---

## 3. Arquitectura del Sistema

### 3.1 Diagrama de Capas

```
┌─────────────────────────────────────────────────┐
│                   FRONTEND                       │
│          web/index.html + app.js + style.css     │
│              (SPA Vanilla JS, PWA)               │
├──────────────┬──────────────┬────────────────────┤
│  REST API    │  WebSocket   │  Static Files      │
│  /api/*      │  /ws         │  /uploads/*        │
├──────────────┴──────────────┴────────────────────┤
│                 HTTP SERVER                       │
│        Chi Router + Middleware Stack              │
│   (Logger, Recoverer, RealIP, Timeout, CORS)     │
├──────────────────────────────────────────────────┤
│              CAPA DE NEGOCIO                     │
│  ┌──────────┐ ┌───────────┐ ┌──────────────┐    │
│  │attendance│ │  reports   │ │  hikvision   │    │
│  │ engine   │ │ pdf/excel  │ │ ISAPI client │    │
│  └──────────┘ └───────────┘ └──────────────┘    │
├──────────────────────────────────────────────────┤
│            CAPA DE PERSISTENCIA                  │
│         store.Repository (interfaz)              │
│         SQLiteStore (implementación)             │
│              ponches.db (WAL mode)               │
└──────────────────────────────────────────────────┘
```

### 3.2 Estructura de Directorios

```
ponches/
├── cmd/
│   ├── server/main.go          # Punto de entrada — bootstrap del sistema
│   └── seed/                    # Seed de datos de prueba
├── internal/
│   ├── api/                     # 13 archivos — Handlers HTTP + Router
│   │   ├── router.go            # Definición de rutas, Server struct, middleware
│   │   ├── handlers_auth.go     # Login, logout, registro de usuarios
│   │   ├── handlers_employees.go# CRUD empleados + fotos
│   │   ├── handlers_devices.go  # Gestión dispositivos Hikvision (34KB)
│   │   ├── handlers_reports.go  # Generación reportes PDF/Excel (28KB)
│   │   ├── handlers_org.go      # Departamentos y posiciones
│   │   ├── handlers_config.go   # Configuración del sistema
│   │   ├── handlers_faces.go    # Registro/eliminación facial
│   │   ├── handlers_leaves.go   # Permisos y ausencias
│   │   ├── handlers_travel.go   # Viáticos
│   │   ├── handlers_public_directory.go # Directorio público
│   │   ├── device_sync.go       # Sincronización batch con dispositivos
│   │   └── face_image.go        # Procesamiento de imágenes faciales
│   ├── attendance/              # Motor de cálculo de asistencia
│   │   ├── engine.go            # EventProcessor, cálculos de jornada
│   │   ├── models.go            # DayResult, PayrollResult, Shift
│   │   └── engine_test.go       # Tests unitarios
│   ├── auth/                    # Autenticación JWT + RBAC
│   │   ├── jwt.go               # Generación/validación de tokens
│   │   ├── middleware.go         # Middleware JWT + RequireRole
│   │   └── password.go          # Hashing bcrypt
│   ├── config/                  # Configuración
│   │   ├── config.go            # Struct Config + carga de .env
│   │   ├── app_config.go        # Configuración persistida en DB
│   │   └── overrides.go         # Merge de config DB → runtime
│   ├── employees/models.go      # Employee, Department, Position, Leave, Travel
│   ├── hikvision/               # Integración hardware
│   │   ├── client.go            # Cliente ISAPI con Digest Auth
│   │   ├── listener.go          # EventListener (polling) + PushListener
│   │   ├── users.go             # CRUD usuarios en dispositivo
│   │   ├── faces.go             # Registro facial en dispositivo
│   │   ├── device.go            # Info del dispositivo
│   │   ├── doors.go             # Control de puertas
│   │   └── cards.go             # Tarjetas de acceso
│   ├── ldap/                    # Sincronización Active Directory
│   ├── middleware/ratelimit.go   # Rate limiting (API + Auth)
│   ├── realtime/hub.go          # WebSocket Hub para broadcast
│   ├── reports/                 # Generadores de reportes
│   │   ├── pdf.go               # Reportes PDF (fpdf)
│   │   ├── excel.go             # Reportes Excel (excelize)
│   │   ├── attendance_period.go # Reporte de período de asistencia
│   │   └── company.go           # Datos de empresa para reportes
│   ├── setup/                   # Inicialización (admin por defecto)
│   ├── store/                   # Capa de persistencia
│   │   ├── store.go             # Interfaz Repository (contrato)
│   │   ├── sqlite.go            # Implementación SQLite (36KB, ~1155 líneas)
│   │   ├── users.go             # CRUD usuarios
│   │   ├── config.go            # CRUD configuración key-value
│   │   └── sqlite_test.go       # Tests de integración
│   └── users/                   # Modelos de usuario
├── web/                         # Frontend SPA
│   ├── index.html               # HTML principal (113KB)
│   ├── app.js                   # Lógica JS completa (176KB)
│   ├── style.css                # Estilos CSS (70KB)
│   ├── directorio.html          # Directorio público
│   ├── service-worker.js        # PWA service worker
│   └── manifest.webmanifest     # PWA manifest
├── docs/                        # Documentación
│   └── openapi.yaml             # Especificación OpenAPI 3.0
├── .env / .env.example          # Variables de entorno
├── Dockerfile                   # Contenedor Docker
└── ponches.db                   # Base de datos SQLite (producción)
```

---

## 4. Base de Datos

### 4.1 Esquema de Tablas

SQLite con modo WAL, `busy_timeout=5000`, `synchronous=NORMAL`, una sola conexión compartida.

| Tabla | Descripción | Campos Clave |
|-------|------------|-------------|
| `users` | Usuarios del sistema | id, username, email, password (bcrypt), role, active |
| `employees` | Empleados | id, employee_no (unique), first/last_name, department_id, position_id, status, base_salary, photo_data, face_id |
| `departments` | Departamentos | id, name, description, parent_id, manager_id |
| `positions` | Cargos | id, name, department_id, level |
| `attendance_events` | Eventos de ponche | id (autoincrement), device_id, employee_no, timestamp, type |
| `app_config` | Config key-value | key (PK), value |
| `leaves` | Permisos/ausencias | id, employee_id, type, start/end_date, days, status, authorized_by |
| `travel_allowance_rates` | Tarifas viáticos | id, name, type (percentage/fixed), value, active |
| `travel_allowances` | Solicitudes viáticos | id, employee_id, rate_id, destination, departure/return_date, calculated_amount, status |
| `device_logs` | Logs de dispositivos | id, device_id, operation, error_message, level |

### 4.2 Índices

- `idx_attendance_unique` — UNIQUE(device_id, employee_no, timestamp, type)
- `idx_employees_employee_no`, `idx_employees_department`, `idx_employees_status`
- `idx_attendance_employee_no`, `idx_attendance_timestamp`, `idx_attendance_device`
- `idx_users_username`, `idx_users_email`
- `idx_leaves_employee`, `idx_leaves_dates`
- `idx_travel_allowances_employee`, `idx_travel_allowances_status`

### 4.3 Patrón Repository

```go
// store/store.go — Interfaz (contrato)
type Repository interface {
    // Employees: Create, Get, GetByNo, List, Update, UpdatePhoto, ClearPhoto, Delete, Upsert
    // Departments: Create, Get, List, Update, Delete, Upsert
    // Positions: Create, Get, List, Update, Delete, Upsert
    // Users: Create, Get, GetByID, GetByUsername, HasAdminByEmail, List, Update, Delete
    // Config: GetValue, SetValue, GetAll, SetMultiple
    // Events: Save (INSERT OR IGNORE), Get (con filtros)
    // DeviceLogs: Save, Get
    // TravelRates: CRUD completo
    // TravelAllowances: CRUD completo
    // Leaves: CRUD + ListByEmployee
}
```

---

## 5. Autenticación y Autorización

### 5.1 JWT

- **Algoritmo:** HS256
- **Header:** `Authorization: Bearer <token>` o cookie `ponches_auth`
- **Expiración:** Configurable (default 24h)
- **Claims:** userId, username, email, role, fullName

### 5.2 Roles (RBAC)

| Rol | Permisos |
|-----|---------|
| `admin` | Acceso total: usuarios, config, dispositivos, LDAP, CRUD completo |
| `manager` | Empleados, organización, permisos, viáticos (lectura/escritura) |
| `viewer` | Solo lectura: empleados, departamentos, asistencia, reportes |

### 5.3 Rate Limiting

| Tipo | Límite | Implementación |
|------|--------|---------------|
| API general | 10 req/s, burst 20 | Token bucket por IP (`x/time/rate`) |
| Autenticación | 5 intentos / 5 min | Contador por IP con ventana deslizante |

### 5.4 Credenciales por Defecto
```
Usuario: admin | Contraseña: admin123
```

---

## 6. API REST — Catálogo de Endpoints

### 6.1 Públicos (sin auth)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/public/auth/login` | Iniciar sesión |
| GET | `/api/public/directory` | Directorio público |
| GET | `/api/public/directory/{employeeNo}/contact.vcf` | vCard del empleado |

### 6.2 Protegidos — Todos los roles

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/employees` | Listar empleados |
| GET | `/api/departments` | Listar departamentos |
| GET | `/api/positions` | Listar posiciones |
| GET | `/api/attendance/events` | Eventos de asistencia |
| GET | `/api/attendance/summary` | Resumen por período |
| GET | `/api/attendance/daily` | Asistencia del día |
| GET | `/api/attendance/stats` | Estadísticas dashboard |
| GET | `/api/reports/daily` | Reporte diario (PDF/Excel) |
| GET | `/api/reports/payroll` | Pre-nómina |
| GET | `/api/reports/late` | Reporte de tardanzas |
| GET | `/api/reports/kpis` | KPIs de asistencia |
| GET | `/api/reports/attendance` | Reporte por período |
| GET | `/api/leaves` | Listar permisos |
| GET | `/api/travel-rates` | Tarifas de viáticos |
| GET | `/api/travel-allowances` | Listar viáticos |

### 6.3 Admin + Manager

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST/PUT/DELETE | `/api/employees/{id}` | CRUD empleados |
| PUT/DELETE | `/api/employees/{employeeNo}/photo` | Foto del empleado (BLOB en DB) |
| POST/DELETE | `/api/employees/{employeeNo}/face` | Registro facial |
| POST/PUT/DELETE | `/api/departments/{id}` | CRUD departamentos |
| POST/PUT/DELETE | `/api/positions/{id}` | CRUD posiciones |
| POST/PUT/DELETE | `/api/leaves/{id}` | CRUD permisos |
| POST/PUT/DELETE | `/api/travel-allowances/{id}` | CRUD viáticos |
| POST | `/api/travel-allowances/{id}/approve` | Aprobar viático |

### 6.3.1 Endpoints adicionales (todos los roles autenticados)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/reports/attendance/data` | Datos JSON del reporte de período |
| GET | `/api/travel-allowances/{id}/pdf` | PDF de un viático individual |
| POST | `/api/notify/employee` | Genera mensaje WhatsApp / correo para un empleado |
| POST | `/api/config/rnc-lookup` | Proxy hacia el lookup de RNC del gobierno |
| GET | `/api/devices/logs` | Logs consolidados de todos los dispositivos |

---

### 6.4 Solo Admin

| Método | Ruta | Descripción |
|--------|------|-------------|
| CRUD | `/api/users` | Gestión de usuarios |
| GET/POST | `/api/config` | Configuración del sistema |
| CRUD | `/api/devices/configured` | Dispositivos Hikvision |
| POST | `/api/devices/configured/{id}/sync` | Sync empleados → dispositivo |
| POST | `/api/devices/configured/{id}/sync-one/{no}` | Sync un empleado |
| DELETE | `/api/devices/configured/{id}/sync-one/{no}` | Revocar empleado |
| POST | `/api/devices/import-users` | Importar desde dispositivos |
| POST | `/api/devices/read-events` | Leer eventos recientes |
| POST | `/api/ldap/test` | Probar conexión LDAP |
| POST | `/api/ldap/sync` | Sincronizar LDAP |
| CRUD | `/api/travel-rates` | Tarifas de viáticos |

---

## 7. Motor de Asistencia

### 7.1 Flujo de Cálculo

1. Se obtienen eventos del día para un empleado (`GetEvents` con filtro de fecha)
2. Se ordenan por timestamp
3. El primero = check-in, el último = check-out
4. Se calcula duración total, deduciendo almuerzo si > 5h
5. Se determina tardanza basada en hora de entrada + período de gracia
6. Se separan horas regulares vs. horas extras según el umbral

### 7.2 Configuración de Horario

```go
type AttendanceConfig struct {
    ShiftStart         string  // "08:00"
    ShiftEnd           string  // "17:00"
    WeeklyScheduleJSON string  // JSON con horario por día de la semana
    LunchBreakMinutes  int     // 60
    GracePeriodMinutes int     // 5
    OvertimeThreshold  float64 // 8.0
}
```

Soporta horario semanal personalizado por día (JSON `map[string]DaySchedule`).

### 7.3 Horas Extras

| Tipo | Multiplicador | Aplica |
|------|-------------|--------|
| Simple | 1.5x | Días laborables |
| Doble | 2.0x | Fines de semana |
| Triple | 3.0x | Feriados |

**Fórmula:** `Tarifa/hora = Salario Base / 160h`

### 7.4 Modelos Principales

- **DayResult:** CheckIn, CheckOut, TotalHours, RegularHours, Overtime, IsLate, LateMinutes, IsAbsent, IsIncomplete
- **PayrollResult:** BaseSalary, OvertimeSimple/Double/Triple, OvertimePay, Deductions, DaysWorked/Absent/Late
- **AttendanceSummary:** TotalDays, PresentDays, AbsentDays, LateDays, AttendanceRate

---

## 8. Integración Hikvision

### 8.1 Cliente ISAPI

- **Autenticación:** HTTP Digest Auth (RFC 2617, MD5)
- **Protocolo:** HTTP con soporte JSON y XML (fallback)
- **Reintentos:** 3 intentos con backoff exponencial (500ms, 1s)
- **Timeout:** 20s por request

### 8.2 Operaciones Soportadas

| Operación | Endpoint ISAPI |
|-----------|---------------|
| Listar usuarios | `POST /ISAPI/AccessControl/UserInfo/Search` |
| Crear usuario | `POST /ISAPI/AccessControl/UserInfo/Record` |
| Modificar usuario | `PUT /ISAPI/AccessControl/UserInfo/Modify` |
| Eliminar usuario | `PUT /ISAPI/AccessControl/UserInfo/Delete` |
| Registrar rostro | `POST /ISAPI/Intelligent/FDLib/FaceDataRecord` |
| Eliminar rostro | `PUT /ISAPI/Intelligent/FDLib/FDSearch/Delete` |
| Consultar eventos | `POST /ISAPI/AccessControl/AcsEvent` |
| Info dispositivo | `GET /ISAPI/System/deviceInfo` |
| Sincronizar hora | `PUT /ISAPI/System/time` |

### 8.3 Event Listener

**Modo Polling:** Cada 5 segundos consulta eventos de las últimas 24h via `GetRecentEvents`. Al detectar un evento nuevo:

1. Guarda en SQLite (`INSERT OR IGNORE` — deduplicación por índice único)
2. Resuelve nombre del empleado en la DB
3. Broadcast via WebSocket a todos los clientes conectados

**Modo Push:** Servidor HTTP auxiliar que recibe notificaciones push del dispositivo vía `POST /ISAPI/Intelligent/Push` o `POST /event` (JSON o XML).

### 8.4 Sincronización Batch (device_sync.go)

Cuando se hace sync de empleados hacia un dispositivo:

1. Se obtiene la lista actual de usuarios en el dispositivo via ISAPI
2. Se determina si cada empleado debe **crearse** (`POST UserInfo/Record`) o **modificarse** (`PUT UserInfo/Modify`)
3. Si el empleado tiene foto registrada (`face_id != ""`), se envía también el registro facial
4. Los errores se persisten en la tabla `device_logs` con nivel `error`/`warning`/`info`

### 8.5 Limitaciones Conocidas del Hardware

> [!WARNING]
> Las políticas de seguridad del firmware Hikvision impiden la **extracción remota** de imágenes biométricas almacenadas en el dispositivo. El flujo es unidireccional: las fotos/templates se **envían** al dispositivo pero no se pueden recuperar mediante ISAPI. La foto del empleado que se muestra en la UI proviene exclusivamente del campo `photo_data` en la base de datos local.

---

## 9. WebSocket (Tiempo Real)

### 9.1 Conexión
```
ws://localhost:8080/ws?token=<JWT>
```
Requiere JWT válido. Validación de mismo origen habilitada.

### 9.2 Mensaje de Evento
```json
{
  "type": "attendance",
  "data": {
    "employeeNo": "101",
    "employeeName": "Juan Pérez",
    "deviceId": "192.168.1.64",
    "timestamp": "2026-04-30T08:05:00-04:00"
  },
  "timestamp": "2026-04-30T12:05:01Z"
}
```

### 9.3 Arquitectura Hub
- **Hub** mantiene mapa de conexiones activas
- Canales: `register`, `unregister`, `broadcast`
- Escritura JSON a cada cliente; desconexión automática en error

---

## 10. Reportes

| Reporte | Formatos | Datos |
|---------|---------|-------|
| Diario | PDF, Excel | Asistencia de todos los empleados en una fecha |
| Pre-nómina | PDF, Excel | Horas regulares, extras, deducciones por período |
| Tardanzas | PDF | Empleados tarde con minutos de retraso |
| KPIs | JSON | Métricas de asistencia del dashboard |
| Período | PDF, Excel | Asistencia detallada por rango de fechas |

---

## 11. Variables de Entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | Puerto HTTP |
| `DB_PATH` | ./ponches.db | Ruta SQLite |
| `LOG_LEVEL` | info | Nivel de log |
| `COMPANY_NAME` | Empresa | Nombre para reportes |
| `COMPANY_RNC` | (vacío) | RNC para reportes |
| `DEFAULT_SHIFT_START` | 08:00 | Inicio de jornada |
| `DEFAULT_SHIFT_END` | 17:00 | Fin de jornada |
| `LUNCH_BREAK_MINUTES` | 60 | Almuerzo (minutos) |
| `GRACE_PERIOD_MINUTES` | 5 | Gracia para tardanza |
| `OVERTIME_MULTIPLIER_SIMPLE` | 1.5 | Multiplicador HE simple |
| `OVERTIME_MULTIPLIER_DOUBLE` | 2.0 | Multiplicador HE doble |
| `OVERTIME_MULTIPLIER_TRIPLE` | 3.0 | Multiplicador HE triple |
| `JWT_SECRET` | (cambiar) | Secreto para firmar JWT |
| `JWT_EXPIRATION_HOURS` | 24 | Expiración del token |
| `TRAVEL_ENABLED` | true | Módulo de viáticos |

---

## 12. Seguridad

- **Passwords:** bcrypt hash
- **Tokens:** JWT HS256 con expiración
- **Rate Limiting:** API (token bucket) + Auth (ventana deslizante)
- **Headers:** X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy
- **WebSocket:** Validación JWT + same-origin check
- **SQLite:** Prepared statements para prevenir SQL injection
- **Eventos:** `INSERT OR IGNORE` para deduplicación

---

## 13. Despliegue y Operación

### Compilar
```bash
go build -o ponches.exe ./cmd/server
```

### Arrancar en Producción (Windows)
```bat
# Doble-click en start-ponches.bat
# O ejecutar desde CMD:
start-ponches.bat
```
El script `start-ponches.bat` hace:
1. Cambia a la carpeta del ejecutable (`cd /d "%~dp0"`)
2. Imprime fecha/hora de inicio
3. Lanza `ponches.exe` (bloquea hasta que se detenga)
4. Muestra `pause` al salir para ver errores

### Acceso desde Otros Equipos de la Red
```
http://<IP-del-servidor>:8080
```
- El puerto 8080 debe estar abierto en el Firewall de Windows
- La IP del servidor debe ser fija (IP estática o reserva DHCP)
- Compatible con cualquier navegador moderno (Chrome, Firefox, Edge)
- Puede instalarse como PWA desde el navegador

### Backup
```bash
# Usar backup-db.bat (copia ponches.db con timestamp)
```

### Desarrollo
```bash
go run ./cmd/server    # Ejecutar sin compilar
```

### Tests
```bash
go test ./... -v       # Todos los tests
go test -cover ./...   # Con cobertura
```

---

## 14. Frontend (SPA)

- **Tipo:** Single Page Application monolítica (sin framework)
- **Archivos principales:** `index.html` (113KB), `app.js` (176KB), `style.css` (70KB)
- **Tema:** Oscuro por defecto con variables CSS
- **Navegación:** Sistema de tabs/secciones con visibilidad CSS
- **API calls:** `fetch()` con header `Authorization: Bearer`
- **Modales:** Inline HTML con funciones show/hide
- **PWA:** Service worker + manifest para instalación offline

---

## 15. Módulo LDAP / Active Directory

### 15.1 Funcionamiento

La clase `ldap.Syncer` realiza una sincronización unidireccional **Active Directory → Ponches**:

1. Conecta al servidor LDAP vía `ldap://host:port`
2. Hace bind con `LDAPBindDN` / `LDAPBindPass`
3. Busca entradas con el filtro `LDAPUserFilter` en el `LDAPBaseDN`
4. Por cada entrada extrae: `sAMAccountName`, `employeeNumber`, `givenName`, `sn`, `mail`, `department`, `title`
5. Hace `UpsertDepartment` y `UpsertPosition` si existen esos campos
6. Hace `UpsertEmployee` (insert o update) usando el `distinguishedName` como ID estable

> Si el campo `employeeNumber` está vacío, usa `sAMAccountName` como número de empleado.

### 15.2 Atributos AD Mapeados

| Atributo LDAP | Campo Ponches |
|--------------|---------------|
| `sAMAccountName` | `employeeNo` (fallback) |
| `employeeNumber` | `employeeNo` (preferido) |
| `givenName` | `firstName` |
| `sn` | `lastName` |
| `mail` | `email` |
| `department` | `departmentId` (crea el dept si no existe) |
| `title` | `positionId` (crea la posición si no existe) |
| `distinguishedName` | `id` (UUID estable) |

### 15.3 Configuración

```env
LDAP_HOST=ldap.empresa.com
LDAP_PORT=389
LDAP_BASE_DN=dc=empresa,dc=com
LDAP_BIND_DN=cn=admin,dc=empresa,dc=com
LDAP_BIND_PASS=tu-contrasena
LDAP_USER_FILTER=(objectClass=person)
```

La config LDAP también se puede ajustar desde la UI en **Configuración → LDAP** sin reiniciar el servidor (ver Sección 16).

---

## 16. Configuración Dinámica (Dos Capas)

El sistema usa un mecanismo de configuración en dos capas para permitir cambios desde la UI sin editar archivos:

```
Capa 1: .env / variables de entorno  (carga al inicio)
           ↓
       config.Load()  →  Config struct
           ↓
Capa 2: tabla app_config en SQLite   (se lee al inicio)
           ↓
       config.ApplyOverrides()  →  Config struct (actualizada)
           ↓
       Runtime del servidor
```

**Regla:** Los valores de la DB **sobreescriben** los del `.env` si están presentes y no vacíos.

### Claves en app_config

Todas las claves usan snake_case:

| Clave DB | Corresponde a |
|----------|---------------|
| `company_name` | `Config.CompanyName` |
| `company_rnc` | `Config.CompanyRNC` |
| `default_shift_start` | `Config.DefaultShiftStart` |
| `grace_period_minutes` | `Config.GracePeriodMinutes` |
| `overtime_multiplier_simple/double/triple` | Multiplicadores HE |
| `hikvision_ip/port/username/password` | Credenciales dispositivo |
| `ldap_host/port/base_dn/bind_dn/bind_pass` | Config LDAP |
| `managed_devices` | JSON array de dispositivos (ver Sección 17) |

> **Nota:** El servidor **no se reinicia** cuando se guarda la configuración desde la UI. Los cambios en config de asistencia se aplican en la próxima solicitud de cálculo. Los cambios en dispositivos Hikvision requieren reinicio del servidor para que el Event Listener los tome.

---

## 17. Gestión de Dispositivos Hikvision

### 17.1 Almacenamiento

Los dispositivos gestionados **no tienen tabla propia**. Se almacenan como un JSON array en `app_config` con la clave `managed_devices`:

```json
[
  {
    "id": "uuid-del-dispositivo",
    "name": "Entrada Principal",
    "ip": "192.168.1.64",
    "port": 80,
    "username": "admin",
    "password": "****",
    "isDefault": true
  }
]
```

### 17.2 Ciclo de Vida

| Etapa | Acción |
|-------|--------|
| **Registro** | UI → `POST /api/devices/configured` → guarda en `app_config` |
| **Arranque** | `main.go` → `loadManagedDevices()` → inicia un `EventListener` por dispositivo |
| **Sync** | `POST /api/devices/configured/{id}/sync` → sincroniza todos los empleados activos |
| **Sync uno** | `POST /api/devices/configured/{id}/sync-one/{employeeNo}` → sincroniza un empleado |
| **Revocar** | `DELETE /api/devices/configured/{id}/sync-one/{employeeNo}` → elimina del dispositivo |
| **Logs** | `GET /api/devices/configured/{id}/logs` → últimos errores de operación |

### 17.3 Descubrimiento SADP

El sistema incluye soporte para descubrimiento de dispositivos en la red local vía protocolo SADP:
```
GET /api/discovery/discover   → escanea la red (timeout configurable)
GET /api/discovery/refresh    → actualiza lista de dispositivos descubiertos
```

---

## 18. Directorio Público de Empleados

Aplicación independiente accesible sin autenticación:

| Ruta | Descripción |
|------|-------------|
| `/directorio` o `/directory` | UI del directorio público |
| `/api/public/directory` | API JSON: lista de empleados activos |
| `/api/public/directory/{employeeNo}/contact.vcf` | Descarga vCard del empleado |

### Datos expuestos en el directorio
- Nombre completo, cargo, departamento
- Teléfono, email (si el empleado tiene)
- Foto (si tiene photo_data en la DB)

### Archivos
- `web/directorio.html` — HTML de la página pública
- `web/directory.js` — Lógica de búsqueda y filtrado
- `web/directory.css` — Estilos del directorio

> El directorio usa rate limiting del mismo `authLimiter` que el login, para prevenir scraping masivo.

---

*Documento actualizado: 30 de abril de 2026*
