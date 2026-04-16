# CLAUDE.md — Ponches Project Context

> Este archivo proporciona contexto al asistente de IA (Claude, Gemini, etc.) para trabajar con el proyecto **Ponches**.

## Descripción del Proyecto

**Ponches** es un sistema de control de asistencia empresarial integrado con dispositivos biométricos Hikvision. Permite registrar entradas/salidas de empleados, calcular horas extras, tardanzas, generar reportes de nómina, y gestionar la estructura organizacional.

- **Empresa:** Grupo MV
- **Idioma del proyecto:** Código en inglés, UI y documentación en español
- **Plataforma objetivo:** Windows (uso local en red corporativa)

---

## Stack Tecnológico

| Capa       | Tecnología                                    |
|------------|-----------------------------------------------|
| Backend    | **Go 1.25** con **Chi v5** router              |
| Base de datos | **SQLite** (embebida via `modernc.org/sqlite`) |
| Frontend   | **Vanilla JS** + **CSS custom** (SPA monolítica en `web/`) |
| WebSocket  | `gorilla/websocket` para eventos en tiempo real |
| Reportes   | **fpdf** (PDF) + **excelize** (Excel)           |
| Auth       | **JWT** (`golang-jwt/v5`) + **bcrypt**          |
| Logging    | **zerolog** con output a consola                |
| Hardware   | Integración **Hikvision ISAPI** (cámaras/lectores faciales) |

---

## Estructura del Proyecto

```
ponches/
├── cmd/
│   ├── server/main.go       # Punto de entrada principal
│   └── seed/                 # Seed de datos de prueba
├── internal/
│   ├── api/                  # HTTP handlers + router (Chi)
│   │   ├── router.go         # Definición de rutas y middleware
│   │   ├── handlers_auth.go  # Login, logout, usuarios
│   │   ├── handlers_employees.go
│   │   ├── handlers_org.go   # Departamentos y posiciones
│   │   ├── handlers_reports.go # Reportes PDF/Excel
│   │   ├── handlers_config.go
│   │   ├── handlers_devices.go
│   │   ├── handlers_faces.go # Registro facial Hikvision
│   │   ├── handlers_leaves.go # Permisos y ausencias
│   │   ├── handlers_travel.go # Viáticos
│   │   └── handlers_public_directory.go
│   ├── attendance/           # Motor de cálculo de asistencia
│   │   ├── engine.go         # ProcessEvents, CalculateDayResult
│   │   ├── models.go         # DayResult, PayrollResult, Shift
│   │   └── engine_test.go    # Tests unitarios
│   ├── auth/                 # JWT service + middleware + bcrypt
│   │   ├── jwt.go
│   │   ├── middleware.go
│   │   └── password.go
│   ├── config/               # Carga de .env + AppConfig modelo
│   ├── discovery/            # Descubrimiento SADP de dispositivos
│   ├── employees/            # Modelos: Employee, Department, Position, Leave, TravelAllowance
│   ├── hikvision/            # Cliente ISAPI + EventListener + PushListener
│   ├── ldap/                 # Sincronización LDAP/Active Directory
│   ├── middleware/           # Rate limiting (API + Auth)
│   ├── realtime/             # WebSocket Hub para broadcast
│   ├── reports/              # Generadores PDF y Excel
│   ├── setup/                # InitDefaultAdmin
│   ├── store/                # Capa de persistencia SQLite
│   │   ├── store.go          # Interfaz Repository (contrato)
│   │   ├── sqlite.go         # Implementación SQLite (~30K)
│   │   ├── users.go          # CRUD de usuarios
│   │   ├── config.go         # CRUD de configuración
│   │   └── sqlite_test.go    # Tests de store
│   └── users/                # Modelos de usuario y auth requests
├── web/                      # Frontend (SPA)
│   ├── index.html            # HTML principal (~84K)
│   ├── app.js                # Lógica JS (~116K)
│   ├── style.css             # Estilos (~49K)
│   ├── directorio.html       # Directorio público de empleados
│   ├── directory.js/css      # JS/CSS del directorio
│   ├── service-worker.js     # PWA service worker
│   ├── manifest.webmanifest  # PWA manifest
│   └── icons/                # Íconos de la app
├── docs/
│   └── openapi.yaml          # Especificación OpenAPI 3.0
├── .env.example              # Variables de entorno de ejemplo
├── PLAN_TRABAJO.md           # Plan de trabajo con fases completadas
├── README.md                 # Documentación del proyecto
├── go.mod / go.sum           # Dependencias Go
└── ponches.db                # Base de datos SQLite (producción local)
```

---

## Convenciones de Código

### Go (Backend)

- **Patrón de diseño:** Arquitectura limpia con interfaz `Repository` en `store/store.go`
- **Routing:** Chi v5 con grupos por rol (`admin`, `manager`, `viewer`)
- **Naming:** Handlers siguen el patrón `handle<Action><Resource>` (ej: `handleCreateEmployee`, `handleReportPayroll`)
- **Helpers:** `writeJSON(w, status, data)` y `writeError(w, status, msg)` para respuestas
- **Contexto:** Siempre pasar `r.Context()` a las funciones del store
- **Errores:** Logs con `zerolog` (`log.Error().Err(err).Msg("...")`)
- **IDs:** UUIDs generados con `github.com/google/uuid`
- **Módulo:** `ponches` (importar como `ponches/internal/...`)

### JavaScript (Frontend)

- **SPA monolítica:** Todo el frontend vive en `web/app.js` y `web/index.html`
- **Sin framework:** Vanilla JS con manipulación directa del DOM
- **API calls:** `fetch('/api/...', { headers: { 'Authorization': 'Bearer ' + token } })`
- **Modales:** Patrón de modales inline en HTML con funciones `show/hide`
- **Navegación:** Sistema de pestañas/secciones con visibilidad CSS

### CSS

- **Sin framework:** CSS custom puro en `web/style.css`
- **Variables CSS:** Usar variables existentes para colores y espaciado
- **Dark theme:** El proyecto usa un tema oscuro por defecto

---

## Base de Datos

**SQLite** con las siguientes tablas principales:

| Tabla                 | Descripción                        |
|-----------------------|------------------------------------|
| `users`               | Usuarios del sistema (admin, manager, viewer) |
| `employees`           | Empleados con datos personales     |
| `departments`         | Departamentos de la empresa        |
| `positions`           | Cargos/posiciones                  |
| `attendance_events`   | Eventos de ponche (entrada/salida) |
| `app_config`          | Configuración key-value persistente |
| `travel_allowance_rates` | Tarifas de viáticos              |
| `travel_allowances`   | Solicitudes de viáticos            |
| `leaves`              | Permisos y ausencias               |
| `managed_devices`     | Dispositivos Hikvision configurados |

### Patrones de acceso a datos

- Interfaz `Repository` en `store/store.go` define el contrato
- Implementación SQLite en `store/sqlite.go`
- Usar `context.Background()` en `main.go`, `r.Context()` en handlers
- Prepared statements para todas las queries

---

## Autenticación y Autorización

- **JWT:** Token en header `Authorization: Bearer <token>`
- **Roles:** `admin` (todo), `manager` (empleados + asistencia), `viewer` (solo lectura)
- **Middleware:** `s.JWTService.Middleware` para auth, `auth.RequireRole("admin")` para RBAC
- **Credenciales por defecto:** `admin` / `admin123`
- **Rate limiting:** API (10 req/s, burst 20), Auth (5 intentos / 5 min)

---

## API Principales

### Rutas Públicas (`/api/public/`)
- `POST /api/public/auth/login` — Iniciar sesión
- `GET /api/public/directory` — Directorio público de empleados

### Rutas Protegidas (`/api/`)
- **Empleados:** CRUD en `/api/employees`
- **Organización:** `/api/departments`, `/api/positions`
- **Asistencia:** `/api/attendance/summary`, `/api/attendance/daily`, `/api/attendance/stats`
- **Reportes:** `/api/reports/daily`, `/api/reports/payroll`, `/api/reports/late`, `/api/reports/attendance`
- **Viáticos:** `/api/travel-rates`, `/api/travel-allowances`
- **Permisos:** `/api/leaves`
- **Config:** `/api/config` (admin)
- **Usuarios:** `/api/users` (admin)
- **Dispositivos:** `/api/discovery/scan`, `/api/devices/configured`
- **LDAP:** `/api/ldap/test`, `/api/ldap/sync`

---

## Comandos de Desarrollo

```bash
# Ejecutar en desarrollo
go run ./cmd/server

# Compilar ejecutable
go build -o ponches.exe ./cmd/server

# Ejecutar tests
go test ./... -v

# Tests con cobertura
go test ./... -cover

# Acceder a la app
# http://localhost:8080
```

---

## Configuración

Variables de entorno en `.env` (copiar de `.env.example`):

- `SERVER_PORT` — Puerto del servidor (default: 8080)
- `DB_PATH` — Ruta de la base de datos SQLite
- `HIKVISION_IP/USERNAME/PASSWORD` — Credenciales del dispositivo
- `JWT_SECRET` — Secreto para firmar tokens
- `DEFAULT_SHIFT_START/END` — Horario laboral
- `GRACE_PERIOD_MINUTES` — Minutos de gracia para tardanzas
- `OVERTIME_MULTIPLIER_*` — Multiplicadores de horas extras

---

## Consideraciones Importantes

1. **No usar Docker** para desarrollo local. El proyecto se ejecuta directamente con `ponches.exe` en Windows
2. **El frontend es monolítico** — `app.js` tiene ~116K, `index.html` ~84K. Editar con cuidado
3. **SQLite embebida** — No se necesita servidor de DB externo
4. **Hikvision es opcional** — La app funciona sin dispositivo conectado
5. **WebSocket** — Requiere autenticación JWT para conectar
6. **PWA** — La app tiene service worker y manifest para instalarse como PWA
7. **Idioma:** El código Go está en inglés, pero la UI y los nombres de campos visibles al usuario están en español
8. **El archivo `ponches.db`** es la base de datos de producción local — no eliminarlo

---

## Estado del Proyecto

✅ **Fases 1-6 completadas.** El sistema está en producción local.

### Tecnologías externas integradas
- Hikvision ISAPI (cámaras/lectores faciales)
- LDAP/Active Directory (sincronización de empleados)
- WebSocket (eventos en tiempo real)
- PWA (Progressive Web App)
