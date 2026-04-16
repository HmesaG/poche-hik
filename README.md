# Ponches - Sistema de Control de Asistencia Hikvision

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Sistema completo de control de asistencia integrado con dispositivos Hikvision.

## 🚀 Características

- ✅ **Autenticación JWT** con roles (admin, manager, viewer)
- ✅ **CRUD completo** de empleados, departamentos y posiciones
- ✅ **Registro facial** en dispositivos Hikvision
- ✅ **Eventos en tiempo real** vía WebSocket
- ✅ **Cálculo de asistencia** con horas extras y tardanzas
- ✅ **Reportes** en PDF y Excel (diarios, nómina, tardanzas)
- ✅ **Integración LDAP/Active Directory**
- ✅ **Configuración persistente** en SQLite
- ✅ **Rate limiting** para protección de API

## 📋 Requisitos

- Go 1.24 o superior
- SQLite (embebido)
- Dispositivo Hikvision (opcional, para funcionalidad completa)

## 🔧 Instalación

### Desde código fuente

```bash
# Clonar repositorio
git clone https://github.com/tu-usuario/ponches.git
cd ponches

# Descargar dependencias
go mod download

# Compilar
go build -o ponches.exe ./cmd/server

# Ejecutar
./ponches.exe
```

### Usando Docker (próximamente)

```bash
docker-compose up -d
```

## ⚙️ Configuración

Copiar `.env.example` a `.env` y editar:

```bash
cp .env.example .env
```

### Variables principales

```env
# Servidor
SERVER_PORT=8080
DB_PATH=./ponches.db
LOG_LEVEL=info

# Dispositivo Hikvision
HIKVISION_IP=192.168.1.64
HIKVISION_USERNAME=admin
HIKVISION_PASSWORD=tu-password

# Reglas de asistencia
DEFAULT_SHIFT_START=08:00
DEFAULT_SHIFT_END=17:00
LUNCH_BREAK_MINUTES=60
GRACE_PERIOD_MINUTES=5

# JWT
JWT_SECRET=cambia-esta-clave-secreta-en-produccion
JWT_EXPIRATION_HOURS=24
```

## 🔐 Credenciales por defecto

```
Usuario: admin
Contraseña: admin123
```

⚠️ **IMPORTANTE:** Cambia la contraseña después del primer inicio de sesión.

## 📡 API Endpoints

### Autenticación

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/public/auth/login` | Iniciar sesión |
| POST | `/api/auth/logout` | Cerrar sesión |
| GET | `/api/auth/me` | Usuario actual |

### Empleados

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/employees` | Listar empleados |
| POST | `/api/employees` | Crear empleado |
| GET | `/api/employees/{id}` | Obtener empleado |
| PUT | `/api/employees/{id}` | Actualizar empleado |
| DELETE | `/api/employees/{id}` | Eliminar empleado |

### Asistencia

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/attendance/summary` | Resumen de asistencia |
| GET | `/api/attendance/daily` | Asistencia del día |
| GET | `/api/attendance/stats` | Estadísticas dashboard |

### Reportes

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/reports/daily?date=YYYY-MM-DD&format=pdf\|excel` | Reporte diario |
| GET | `/api/reports/payroll?from=YYYY-MM-DD&to=YYYY-MM-DD` | Pre-nómina |

### Configuración

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/config` | Obtener configuración |
| POST | `/api/config` | Guardar configuración |

## 📚 Documentación API

La documentación completa OpenAPI/Swagger está disponible en:

```
/docs/openapi.yaml
```

Puedes visualizarla usando:
- [Swagger Editor](https://editor.swagger.io/)
- [Stoplight Studio](https://stoplight.io/studio/)

## 🏗️ Arquitectura

```
ponches/
├── cmd/
│   └── server/          # Punto de entrada principal
├── internal/
│   ├── api/             # Handlers y router HTTP
│   ├── attendance/      # Motor de asistencia
│   ├── auth/            # JWT y password hashing
│   ├── config/          # Configuración
│   ├── discovery/       # Descubrimiento SADP
│   ├── employees/       # Modelos de empleados
│   ├── hikvision/       # Cliente ISAPI Hikvision
│   ├── ldap/            # Sincronización LDAP
│   ├── middleware/      # Middleware (rate limiting)
│   ├── realtime/        # WebSocket Hub
│   ├── reports/         # Generación PDF/Excel
│   ├── setup/           # Inicialización
│   ├── store/           # Persistencia SQLite
│   └── users/           # Modelos de usuarios
├── web/                 # Frontend
└── docs/                # Documentación
```

## 🧪 Testing

```bash
# Ejecutar tests
go test ./...

# Con cobertura
go test -cover ./...

# Tests específicos
go test ./internal/store/...
go test ./internal/attendance/...
```

## 🔒 Seguridad

- Contraseñas hasheadas con **bcrypt**
- Tokens **JWT** con expiración configurable
- **Rate limiting** para prevenir abuso
- **CORS** configurable
- Validación de inputs en todos los endpoints

## 📊 Base de Datos

El sistema usa **SQLite** con las siguientes tablas:

- `users` - Usuarios del sistema
- `employees` - Empleados
- `departments` - Departamentos
- `positions` - Posiciones/Cargos
- `attendance_events` - Eventos de asistencia
- `app_config` - Configuración de la aplicación

## 🔄 Flujo de Eventos en Tiempo Real

1. Dispositivo Hikvision registra evento
2. EventListener hace polling cada 5 segundos
3. Evento se guarda en SQLite
4. WebSocket broadcast a clientes conectados
5. Frontend actualiza dashboard en tiempo real

## 📝 Licencia

MIT License - ver [LICENSE](LICENSE) para detalles.

## 🤝 Contribuir

1. Fork el repositorio
2. Crea una rama (`git checkout -b feature/mi-feature`)
3. Commit (`git commit -m 'Añadir mi feature'`)
4. Push (`git push origin feature/mi-feature`)
5. Pull Request

## 📞 Soporte

Para issues o preguntas, abrir un issue en GitHub.

---

**Desarrollado con ❤️ usando Go**

# poche-hik
