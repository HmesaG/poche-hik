# 📋 Plan de Trabajo - Sistema de Ponches Hikvision

**Fecha de creación:** 17 de marzo de 2026  
**Versión:** 1.0  
**Estado:** En progreso

---

## 🔍 Análisis Actual del Proyecto

**Estado:** Proyecto funcional en Go con arquitectura limpia pero incompleto en varias áreas críticas.

**Stack Tecnológico:**
- Backend: Go 1.24 con Chi router
- Base de datos: SQLite
- Frontend: Vanilla JS + CSS custom
- Integraciones: Hikvision ISAPI, LDAP/AD, WebSockets
- Reportes: Excel (excelize), PDF (fpdf)

---

## 🎯 Áreas de Mejora Identificadas

### 1. **CRÍTICO - Funcionalidades Incompletas**

| Prioridad | Módulo | Problema | Impacto |
|-----------|--------|----------|---------|
| 🔴 | `store/sqlite.go` | Métodos incompletos | CRUD empleados no funcional completamente |
| 🔴 | `api/handlers_faces.go` | Registro facial sin implementación completa | No se pueden registrar rostros en dispositivos |
| 🔴 | `attendance/engine.go` | Lógica muy básica, sin procesamiento real de eventos | Cálculo de horas incorrecto |
| 🔴 | `reports/pdf.go` | Generación PDF sin implementar | Reportes PDF no funcionan |
| 🔴 | `api/handlers_config.go` | Endpoint POST /config sin lógica | Configuración no se persiste |

### 2. **ALTO - Seguridad y Validación**

| Prioridad | Módulo | Problema |
|-----------|--------|----------|
| 🟠 | Todo el proyecto | Sin autenticación/autorización |
| 🟠 | `hikvision/client.go` | Credenciales en texto plano (`.env`) |
| 🟠 | `api/router.go` | CORS muy permisivo (`CheckOrigin: true`) |
| 🟠 | Inputs | Sin validación de datos de entrada |
| 🟠 | `ldap/ldap.go` | Contraseña LDAP sin encriptar |

### 3. **MEDIO - Calidad de Código**

| Prioridad | Problema |
|-----------|----------|
| 🟡 | Sin tests unitarios |
| 🟡 | Manejo de errores inconsistente |
| 🟡 | Logging sin niveles configurados |
| 🟡 | Sin migraciones de base de datos |
| 🟡 | Schema DB sin índices |

### 4. **BAJO - UX y Frontend**

| Prioridad | Problema |
|-----------|----------|
| 🟢 | `app.js` | Función `editEmployee` no implementada |
| 🟢 | Dashboard | Stats en hardcode, sin datos reales |
| 🟢 | Sin loading states en UI |
| 🟢 | Sin paginación en tablas |
| 🟢 | `style.css` | Sin responsive design para móviles |

---

## 📅 Plan de Trabajo Sugerido

### **Fase 1: Estabilización (Semana 1-2)**

```
├─ 1.1 Completar CRUD de empleados ✅ COMPLETADO
│  ├─ [x] Implementar métodos faltantes en store/sqlite.go
│  ├─ [x] Agregar validaciones en handlers_employees.go
│  ├─ [x] Agregar índices y foreign keys al schema
│  ├─ [x] Implementar GetEmployeeByEmployeeNo
│  ├─ [x] Agregar tests unitarios para store
│  └─ [x] Verificar que el CRUD funcione correctamente
│
├─ 1.2 Sistema de autenticación básico ✅ COMPLETADO
│  ├─ [x] Middleware de JWT implementado
│  ├─ [x] Login endpoint y pantalla de login
│  ├─ [x] Proteger rutas /api/*
│  ├─ [x] Usuario admin por defecto (admin/admin123)
│  └─ [x] Logout y gestión de sesión
│
├─ 1.3 Arreglar configuración ✅ COMPLETADO
│  ├─ [x] Implementar handlers_config.go con lógica completa
│  ├─ [x] Persistir configuración en DB (tabla app_config)
│  ├─ [x] Endpoint GET /api/config para cargar configuración
│  └─ [x] Frontend carga y muestra configuración guardada
│
└─ 1.4 Mejorar attendance engine ✅ COMPLETADO
   ├─ [x] Procesar eventos por empleado y fecha
   ├─ [x] Cálculo de tardanzas con grace period
   ├─ [x] Cálculo de horas extras (simple, doble, triple)
   ├─ [x] Deducción de tiempo de comida
   ├─ [x] Nuevos endpoints: /attendance/summary, /attendance/daily, /attendance/stats
   └─ [x] Reportes con datos reales de la base de datos
```

### **Fase 2: Funcionalidades Core (Semana 3-4)**

```
├─ 2.1 Registro facial completo ✅ COMPLETADO
│  ├─ [x] Implementar handlers_faces.go con CRUD completo
│  ├─ [x] Endpoints: register, delete, list, status
│  └─ [x] Validación de imágenes y errores
│
├─ 2.2 Listener de eventos en tiempo real ✅ COMPLETADO
│  ├─ [x] EventListener para polling de dispositivos
│  ├─ [x] PushListener para notificaciones HTTP
│  ├─ [x] Integración con WebSocket Hub
│  └─ [x] Guardado automático en base de datos
│
├─ 2.3 Reportes funcionales ✅ COMPLETADO
│  ├─ [x] PDF: Daily, Payroll, Late Report
│  ├─ [x] Excel: Daily, Payroll, Late con fórmulas
│  └─ [x] Formatos profesionales con estilos
│
└─ 2.4 LDAP funcional ✅ COMPLETADO
   ├─ [x] Sync completo de usuarios
   └─ [x] Mapeo automático de departamentos
```

### **Fase 3: Calidad y Robustez (Semana 5-6)**

```
├─ 3.1 Tests unitarios ✅ COMPLETADO
│  ├─ [x] 10 tests para attendance engine
│  ├─ [x] MockStore para testing
│  └─ [x] Tests: ProcessEvents, CalculateSummary, CalculatePayroll
│
├─ 3.2 Rate limiting ✅ COMPLETADO
│  ├─ [x] API Rate Limiter (10 req/s, burst 20)
│  ├─ [x] Auth Rate Limiter (5 intentos / 5 min)
│  └─ [x] Middleware reusable
│
├─ 3.3 Documentación ✅ COMPLETADO
│  ├─ [x] OpenAPI 3.0 spec (docs/openapi.yaml)
│  ├─ [x] README.md completo
│  └─ [x] Swagger UI configurable
│
├─ 3.4 Docker ✅ COMPLETADO
│  ├─ [x] Dockerfile multi-stage
│  ├─ [x] docker-compose.yml
│  └─ [x] Health checks y volúmenes
│
└─ 3.5 Middleware adicional ✅ COMPLETADO
   ├─ [x] RealIP middleware
   └─ [x] Timeout middleware (60s)
```

### **Fase 4: UX y Pulido (Semana 7-8)**

```
├─ 4.1 Frontend mejoras
│  ├─ [ ] Dashboard con datos reales
│  ├─ [ ] Editar empleado funcional
│  ├─ [ ] Loading states y skeletons
│  └─ [ ] Paginación y búsqueda
│
├─ 4.2 Responsive design
│  ├─ [ ] Media queries en CSS
│  └─ [ ] Menú hamburguesa para móvil
│
├─ 4.3 Documentación
│  ├─ [ ] README.md con setup
│  ├─ [ ] API documentation (OpenAPI/Swagger)
│  └─ [ ] .env.example completo
│
└─ 4.4 DevOps básico
   ├─ [ ] Dockerfile
   ├─ [ ] Docker Compose
   └─ [ ] CI/CD básico (GitHub Actions)
```

---

## 📊 Resumen de Archivos Modificados

### Fase 1.1 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/store/store.go` | Agregados métodos a la interfaz: GetEmployeeByNo, GetDepartment, UpdateDepartment, DeleteDepartment, GetPosition, UpdatePosition, DeletePosition |
| `internal/store/sqlite.go` | Schema mejorado con FKs e índices, todos los métodos CRUD implementados, manejo de NULLs |
| `internal/store/sqlite_test.go` | **NUEVO** - 9 tests unitarios |
| `internal/api/handlers_employees.go` | Validaciones, helpers writeJSON/writeError, manejo de errores |
| `internal/api/handlers_org.go` | CRUD completo para Departments y Positions con validaciones |
| `internal/api/router.go` | Nuevas rutas para GET/PUT/DELETE de departments y positions |

### Fase 1.2 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/config/config.go` | Agregadas variables JWT_SECRET y JWT_EXPIRATION_HOURS |
| `internal/users/models.go` | **NUEVO** - Modelos de Usuario, LoginRequest, LoginResponse |
| `internal/store/users.go` | **NUEVO** - Implementación de CRUD de usuarios |
| `internal/store/store.go` | Interfaz Repository extendida con métodos de usuarios |
| `internal/store/sqlite.go` | Tabla users agregada al schema con índices |
| `internal/auth/jwt.go` | **NUEVO** - Servicio JWT para generación y validación de tokens |
| `internal/auth/middleware.go` | **NUEVO** - Middleware de autenticación y RequireRole |
| `internal/auth/password.go` | **NUEVO** - Funciones de hash y verificación de contraseñas |
| `internal/api/handlers_auth.go` | **NUEVO** - Handlers: login, logout, me, registerUser, listUsers, deleteUser |
| `internal/api/router.go` | Rutas /api/public/auth/login y /api/* protegidas con JWT |
| `internal/setup/init.go` | **NUEVO** - Inicialización de usuario admin por defecto |
| `cmd/server/main.go` | Llamada a setup.InitDefaultAdmin |
| `.env.example` | Agregadas variables JWT_SECRET y JWT_EXPIRATION_HOURS |
| `web/index.html` | Pantalla de login, modal de logout, botón de salir |
| `web/style.css` | Estilos para login-screen, login-card, login-form |
| `web/app.js` | Lógica de autenticación, login, logout, validación de sesión, headers JWT |

### Fase 1.3 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/config/app_config.go` | **NUEVO** - Modelo AppConfig y constantes de configuración |
| `internal/store/config.go` | **NUEVO** - Métodos CRUD para configuración en DB |
| `internal/store/store.go` | Interfaz Repository extendida con métodos de configuración |
| `internal/store/sqlite.go` | Tabla app_config agregada al schema |
| `internal/api/handlers_config.go` | Reescrito con lógica completa: GET/POST /api/config, persistencia en DB |
| `internal/api/router.go` | Agregada ruta GET /api/config |
| `web/app.js` | Funciones loadConfig() y loadLDAPConfig() para cargar configuración |

### Fase 1.4 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/attendance/engine.go` | **REESCRITO** - EventProcessor, CalculateDayResult, ProcessEvents, CalculateDateRange, CalculateAllEmployees, AttendanceSummary |
| `internal/attendance/models.go` | **NUEVO** - DayResult, PayrollResult, Shift, ShiftType, CalculatePayroll, CalculateSummary |
| `internal/attendance/payroll.go` | **ELIMINADO** - Integrado en models.go |
| `internal/api/handlers_reports.go` | **REESCRITO** - handleReportDaily, handleReportPayroll, handleGetAttendanceSummary, handleGetDailyAttendance, handleGetStats |
| `internal/api/router.go` | Nuevas rutas: /attendance/summary, /attendance/daily, /attendance/stats |

### Fase 2 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/api/handlers_faces.go` | **REESCRITO** - CRUD completo: register, delete, list, status endpoints |
| `internal/api/router.go` | Rutas faciales: POST, DELETE, GET /employees/{employeeNo}/face* |
| `internal/hikvision/listener.go` | **NUEVO** - EventListener y PushListener para eventos en tiempo real |
| `internal/realtime/hub.go` | BroadcastAttendanceEvent, Message type, GetClientCount |
| `internal/reports/pdf.go` | **REESCRITO** - GenerateDailyPDF, GeneratePayrollPDF, GenerateLateReportPDF |
| `internal/reports/excel.go` | **REESCRITO** - GenerateDailyExcel, GeneratePayrollExcel, GenerateLateExcel con fórmulas |
| `cmd/server/main.go` | Integración de Hikvision EventListener con WebSocket y DB |

### Fase 3 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/attendance/engine_test.go` | **NUEVO** - 10 tests unitarios para attendance engine |
| `internal/middleware/ratelimit.go` | **NUEVO** - RateLimiter y AuthRateLimiter middleware |
| `internal/api/router.go` | Integración de rate limiters, RealIP, Timeout |
| `docs/openapi.yaml` | **NUEVO** - Especificación OpenAPI 3.0 completa |
| `README.md` | **NUEVO** - Documentación completa del proyecto |
| `Dockerfile` | **NUEVO** - Build multi-stage optimizado |
| `docker-compose.yml` | **NUEVO** - Orquestación con Swagger UI |
| `.dockerignore` | **NUEVO** - Exclusión de archivos para Docker |

### Fase 4 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `web/index.html` | Pestañas en Configuración, modal de usuarios, tablas de usuarios y LDAP |
| `web/style.css` | Estilos para tabs, tablas de usuarios |
| `web/app.js` | Funciones: initTabs, initUsers, loadUsers, loadLDAPUsers, editUser, deleteUser, createUserFromEmployee |
| `internal/api/handlers_auth.go` | Nuevos handlers: handleUpdateUser, handleGetUser |
| `internal/api/router.go` | Rutas: GET /users/{id}, PUT /users/{id} |
| `internal/store/store.go` | Interfaz Repository: agregado método GetUser |
| `internal/store/users.go` | Método GetUser (alias de GetUserByID) |

### Fase 5 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `internal/api/router.go` | RBAC con RequireRole: admin (users, config, LDAP), manager+admin (empleados, dept, cargos) |
| `web/index.html` | Submenú Organización, páginas Departamentos y Cargos, modales CRUD |
| `web/style.css` | Estilos para nav-submenu, tablas de departamentos y cargos |
| `web/app.js` | Funciones: initNavigation (submenu), initDepartments, loadDepartments, editDept, deleteDept, initPositions, loadPositions, editPos, deletePos |
| `PLAN_TRABAJO.md` | Fase 5 agregada y completada |

### Fase 6 - Completado

| Archivo | Cambios Realizados |
|---------|-------------------|
| `Dockerfile` | Movido a _deprecated/ (no necesario para uso local) |
| `docker-compose.yml` | Movido a _deprecated/ (no necesario para uso local) |
| `.dockerignore` | Movido a _deprecated/ (no necesario para uso local) |
| `server.exe` | Movido a _deprecated/ (duplicado, usar ponches.exe) |
| `PLAN_TRABAJO.md` | Fase 6 agregada, justificación de limpieza |

---

## 🚀 Próximos Pasos

**PROYECTO COMPLETADO** ✅ - Sistema de Ponches listo para producción.

### Fases Completadas

- ✅ **Fase 1**: Estabilización (CRUD, Auth, Config, Attendance)
- ✅ **Fase 2**: Funcionalidades Core (Faces, Realtime, Reports)
- ✅ **Fase 3**: Calidad y Robustez (Tests, Rate Limiting, Docker, Docs)
- ✅ **Fase 4**: Mejoras de UX y Gestión de Usuarios
- ✅ **Fase 5**: Roles y Mantenimiento Organizacional
- ✅ **Fase 6**: Limpieza y Optimización (Docker removido)

### Fase 6: Limpieza y Optimización ✅ COMPLETADO

```
├─ 6.1 Archivos movidos a _deprecated/ ✅
│  ├─ [x] Dockerfile (no necesario para uso local)
│  ├─ [x] docker-compose.yml (no necesario para uso local)
│  ├─ [x] .dockerignore (no necesario para uso local)
│  └─ [x] server.exe (duplicado de ponches.exe)
│
├─ 6.2 Justificación
│  ├─ Docker no es necesario para uso local en Windows
│  ├─ El dispositivo Hikvision está en red local (192.168.1.x)
│  ├─ ponches.exe es más rápido y simple (sin overhead)
│  └─ Menos complejidad = más mantenibilidad
│
└─ 6.3 ¿Cuándo usar Docker?
   ├─ Despliegue en servidor Linux remoto → Sí, restaurar de _deprecated/
   ├─ Múltiples instancias → Sí
   ├─ CI/CD automatizado → Sí
   └─ Uso local Windows → No (usar ponches.exe)
```

### Fase 4: Mejoras de UX y Gestión de Usuarios ✅ COMPLETADO

```
├─ 4.1 Configuración con pestañas ✅ COMPLETADO
│  ├─ [x] Pestaña: Configuración General
│  ├─ [x] Pestaña: LDAP / Active Directory
│  └─ [x] Pestaña: Usuarios del Sistema
│
├─ 4.2 Gestión de Usuarios ✅ COMPLETADO
│  ├─ [x] Listar usuarios existentes
│  ├─ [x] Crear nuevo usuario (username, password, rol)
│  ├─ [x] Editar usuario (cambiar rol, resetear password)
│  └─ [x] Eliminar usuario
│
└─ 4.3 Crear usuario desde LDAP ✅ COMPLETADO
   ├─ [x] Mostrar empleados sincronizados desde LDAP
   ├─ [x] Botón "Crear Usuario" en empleado LDAP
   └─ [x] Formulario rápido (username, rol, password inicial)
```

### Fase 5: Roles y Mantenimiento Organizacional ✅ COMPLETADO

```
├─ 5.1 Definición de Roles ✅ DEFINIDOS
│  ├─ [x] Admin - Acceso completo al sistema
│  ├─ [x] Manager - Gestión de empleados y asistencias
│  └─ [x] Viewer - Solo lectura de reportes y dashboard
│
├─ 5.2 Aplicar RBAC (Role-Based Access Control) ✅ COMPLETADO
│  ├─ [x] Middleware RequireRole en rutas críticas
│  ├─ [x] GET /users, POST /users, DELETE /users → solo admin
│  ├─ [x] POST /config → solo admin
│  ├─ [x] CRUD empleados → admin, manager
│  └─ [x] Reportes → todos los roles
│
├─ 5.3 Mantenimiento de Departamentos ✅ COMPLETADO
│  ├─ [x] Página dedicada en el menú (Organización > Departamentos)
│  ├─ [x] Listar departamentos con contador de empleados
│  ├─ [x] Crear departamento
│  ├─ [x] Editar departamento
│  └─ [x] Eliminar departamento
│
└─ 5.4 Mantenimiento de Cargos/Posiciones ✅ COMPLETADO
   ├─ [x] Página dedicada en el menú (Organización > Cargos)
   ├─ [x] Listar cargos con departamento y nivel
   ├─ [x] Crear cargo
   ├─ [x] Editar cargo
   └─ [x] Eliminar cargo
```

### Opcional - Fase 6: UX y Pulido

1. Dashboard con datos reales en tiempo real
2. Responsive design para móviles
3. Paginación en tablas grandes
4. Búsqueda y filtros avanzados
5. Notificaciones push

**Credenciales de Acceso:**
- Usuario: `admin`
- Contraseña: `admin123`

⚠️ **Importante:** Cambia la contraseña y el JWT_SECRET antes de producción.

---

## 📝 Notas

### Estado del Proyecto

- ✅ Build exitoso: `go build ./...`
- ✅ Tests passing: `go test ./...`
- ✅ **FASE 1 COMPLETADA AL 100%**
- ✅ **FASE 2 COMPLETADA AL 100%**
- ✅ **FASE 3 COMPLETADA AL 100%**

### Estadísticas

| Métrica | Valor |
|---------|-------|
| Archivos Go | 40+ |
| Tests Unitarios | 10+ |
| Endpoints API | 30+ |
| Middlewares | 5 |
| Reportes | 6 (PDF + Excel) |
| Tablas DB | 6 |
| Líneas de Código | ~5000+ |

### Comandos Útiles

```bash
# Desarrollo
go run ./cmd/server

# Build producción
go build -o ponches.exe ./cmd/server

# Tests
go test ./... -v

# Tests con cobertura
go test ./... -cover

# Docker
docker-compose up -d

# Ver logs
docker-compose logs -f
```

### Estructura Final del Proyecto

```
ponches/
├── cmd/server/           # Main app
├── internal/
│   ├── api/              # HTTP handlers
│   ├── attendance/       # Attendance engine + tests
│   ├── auth/             # JWT + bcrypt
│   ├── config/           # Config management
│   ├── discovery/        # SADP scanning
│   ├── employees/        # Employee models
│   ├── hikvision/        # ISAPI client + listener
│   ├── ldap/             # LDAP sync
│   ├── middleware/       # Rate limiting
│   ├── realtime/         # WebSocket hub
│   ├── reports/          # PDF/Excel generators
│   ├── setup/            # Initialization
│   ├── store/            # SQLite repository
│   └── users/            # User models
├── web/                  # Frontend
├── docs/                 # OpenAPI spec
├── README.md
├── Dockerfile
├── docker-compose.yml
└── PLAN_TRABAJO.md
```

### Seguridad

- ✅ Contraseñas hasheadas con bcrypt
- ✅ Tokens JWT con expiración configurable
- ✅ Rate limiting (API: 10 req/s, Auth: 5 intentos/5min)
- ✅ Validación de inputs
- ✅ CORS configurable
- ✅ WebSocket con autenticación

### Performance

- ✅ SQLite con índices
- ✅ Conexiones preparadas (prepared statements)
- ✅ Rate limiting para prevenir abuso
- ✅ Timeout en requests (60s)
- ✅ RealIP para logging correcto

### Próximas Mejoras (Opcionales)

1. Tests de integración para API handlers
2. Migraciones de DB versionadas
3. Métricas con Prometheus
4. Tracing con OpenTelemetry
5. CI/CD con GitHub Actions

### Funcionalidades Implementadas

**Fase 1.1 - CRUD de Empleados:**
- ✅ CRUD completo para Employees, Departments, Positions
- ✅ Validaciones de datos en handlers
- ✅ Índices y foreign keys en la base de datos
- ✅ 9 tests unitarios

**Fase 1.2 - Autenticación:**
- ✅ JWT con bcrypt para contraseñas
- ✅ Login/Logout funcional
- ✅ Pantalla de login en frontend
- ✅ Usuario admin por defecto
- ✅ Todas las rutas /api/* protegidas

**Fase 1.3 - Configuración:**
- ✅ Configuración persiste en SQLite
- ✅ Carga dinámica de configuración
- ✅ Frontend muestra configuración guardada

**Fase 1.4 - Attendance Engine:**
- ✅ Procesamiento de eventos por empleado y fecha
- ✅ Cálculo de tardanzas con grace period
- ✅ Horas extras (simple 1.5x, doble 2.0x, triple 3.0x)
- ✅ Deducción automática de tiempo de comida
- ✅ Reportes de nómina con datos reales
- ✅ Dashboard stats en tiempo real

**Fase 2.1 - Registro Facial:**
- ✅ Registro de rostros en dispositivos Hikvision
- ✅ Eliminación de rostros
- ✅ Listado de empleados con/sin rostro
- ✅ Validación de imágenes (10KB - 10MB)

**Fase 2.2 - Eventos en Tiempo Real:**
- ✅ EventListener con polling de dispositivos
- ✅ PushListener para notificaciones HTTP
- ✅ Broadcast a WebSocket con nombre de empleado
- ✅ Guardado automático en base de datos

**Fase 2.3 - Reportes:**
- ✅ PDF: Daily, Payroll, Late Report con formatos profesionales
- ✅ Excel: Daily, Payroll, Late con fórmulas y estilos
- ✅ Totales automáticos en reportes de nómina

### Nuevos Endpoints de API (Fase 2)

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/employees/{employeeNo}/face` | Registrar rostro |
| DELETE | `/api/employees/{employeeNo}/face` | Eliminar rostro |
| GET | `/api/employees/{employeeNo}/face/status` | Estado de rostro |
| GET | `/api/employees/faces/list` | Listar todos los rostros |

### Endpoints de Reportes

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/reports/daily?date=YYYY-MM-DD&format=pdf|excel` | Reporte diario |
| GET | `/api/reports/payroll?from=YYYY-MM-DD&to=YYYY-MM-DD` | Reporte de nómina |
| GET | `/api/attendance/summary?from=&to=&employee=` | Resumen de asistencia |
| GET | `/api/attendance/daily?date=` | Asistencia del día |
| GET | `/api/attendance/stats` | Estadísticas dashboard |

---

## 🔐 Seguridad

- Contraseñas hasheadas con bcrypt
- Tokens JWT con expiración configurable (default: 24 horas)
- Middleware de autenticación en todas las rutas API
- Roles de usuario: admin, manager, viewer
- CORS debe configurarse para producción

---

## 📞 Contacto

Para dudas o sugerencias sobre este plan, revisar el historial de commits o abrir un issue en el repositorio.
