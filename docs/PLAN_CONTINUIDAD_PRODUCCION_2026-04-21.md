# Plan de Continuidad y Puesta en Producción — Ponches

**Fecha:** 2026-04-21  
**Dispositivo objetivo:** `http://10.0.0.100` (Hikvision K1T343EWX confirmado)  
**Estado del proyecto:** Fases 1-6 completadas ✅

---

## 📋 Contexto

El proyecto **Ponches** está completo en sus fases 1-6 (sistema funcional con autenticación, CRUD de empleados, departamentos, cargos, viáticos, permisos, reportes PDF/Excel, integración Hikvision ISAPI, WebSocket en tiempo real, LDAP).

### Hallazgos clave del 2026-04-21

- ✅ Dispositivo real confirmado en `10.0.0.100` (HTTP 200 verificado)
- ✅ El CRUD de dispositivos ya existe en el backend (`handlers_devices.go`) con endpoints para:
  - Listar dispositivos configurados
  - Crear / editar / eliminar dispositivos
  - Marcar dispositivo default
  - Sincronizar empleados (masivo e individual)
  - Ver logs de dispositivo
- ⚠️ El listener de eventos usa puerto fijo `80` — puede necesitar ajuste
- ⚠️ Faltan credenciales ISAPI para pruebas de sincronización

### Estado actual del repositorio

Hay cambios locales sin commit en 14 archivos:
- `.gitignore`, `cmd/seed/main.go`
- `internal/api/handlers_config.go`, `handlers_devices.go`, `handlers_reports.go`, `router.go`
- `internal/attendance/engine_test.go`
- `internal/config/config.go`
- `internal/hikvision/listener.go`, `users.go`
- `internal/store/sqlite.go`, `store.go`
- `web/app.js`, `index.html`, `style.css`

---

## 🎯 Objetivos del Plan

| Objetivo | Descripción | Prioridad |
|----------|-------------|-----------|
| **Fase 7** | Mejoras de continuidad del proyecto | Alta |
| **Fase 8** | Puesta en funcionamiento (producción) | Alta |
| **Fase 9** | CRUD del dispositivo 10.0.0.100 | Alta |

---

## 📝 Fase 7: Mejoras de Continuidad

### 7.1 Listener de Eventos configurable por puerto

**Problema:** `listener.go` usa puerto `80` fijo (línea 48 en `main.go`).

**Solución:** Agregar campo `Port` en `managedDevice` y usarlo al crear el EventListener.

**Archivos a modificar:**
- `internal/hikvision/listener.go` — `NewEventListener` acepta `port int`
- `cmd/server/main.go` — leer puerto de configuración del dispositivo default
- `internal/api/handlers_devices.go` — ya tiene campo `Port` en el struct

**Estado:** Pendiente

---

### 7.2 Logs de dispositivo persistentes

**Problema:** `handleGetDeviceLogs` existe pero no hay schema de tabla `device_logs`.

**Solución:** Agregar tabla `device_logs` con campos:
- `id` (UUID, primary key)
- `device_id` (indexed)
- `operation` (VARCHAR)
- `error_message` (TEXT)
- `level` (VARCHAR: info, warn, error)
- `created_at` (TIMESTAMP)

**Archivos a modificar:**
- `internal/store/sqlite.go` — CREATE TABLE + método `GetDeviceLogs`
- `internal/store/store.go` — interfaz `GetDeviceLogs`

**Estado:** Pendiente

---

### 7.3 Frontend: Página Dispositivos funcional

**Problema:** La navegación tiene ítem "Dispositivos" pero no hay página dedicada en `index.html`.

**Solución:** Agregar `div id="devices-page"` con:
- Lista de dispositivos configurados (tabla con estado online/offline)
- Botón "Escanear red" (llama a `/api/discovery/discover`)
- Modal para agregar/editar dispositivo
- Botones: Sincronizar todos, Sincronizar uno, Ver logs

**Archivos a modificar:**
- `web/index.html` — agregar sección `devices-page`
- `web/app.js` — funciones `initDevices()`, `loadDevices()`, `scanNetwork()`, `syncDevice()`, `viewLogs()`

**Estado:** Pendiente

---

## 🚀 Fase 8: Puesta en Producción

### 8.1 Variables de entorno para producción

**Acciones:**

1. Copiar `.env.example` a `.env.production`
2. Configurar valores reales:

```env
# Servidor
SERVER_PORT=8080

# Base de datos
DB_PATH=C:\GrupoMV\Ponches\ponches.db

# JWT (CAMBIAR EN PRODUCCIÓN)
JWT_SECRET=tu-secreto-aleatorio-aqui-generado-con-openssl
JWT_EXPIRATION_HOURS=24

# Dispositivo Hikvision
HIKVISION_IP=10.0.0.100
HIKVISION_USERNAME=____________
HIKVISION_PASSWORD=____________

# Turnos
DEFAULT_SHIFT_START=08:00
DEFAULT_SHIFT_END=17:00
LUNCH_BREAK_MINUTES=60
GRACE_PERIOD_MINUTES=5

# Horas extras
OVERTIME_MULTIPLIER_SIMPLE=1.5
OVERTIME_MULTIPLIER_DOUBLE=2.0
OVERTIME_MULTIPLIER_TRIPLE=3.0
OVERTIME_THRESHOLD_HOURS=8.0

# LDAP (opcional)
LDAP_HOST=
LDAP_PORT=389
LDAP_BIND_DN=
LDAP_BIND_PASS=

# Empresa
COMPANY_NAME=Grupo MV
COMPANY_RNC=
```

**Estado:** Pendiente — requiere credenciales del dispositivo

---

### 8.2 Script de inicio automático (Windows)

**Acciones:**

1. Crear `start-ponches.bat` en `C:\GrupoMV\Ponches\`:

```batch
@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo Iniciando Ponches...
ponches.exe
pause
```

2. Configurar Task Scheduler para inicio automático con Windows:
   - Trigger: "At startup"
   - Action: `start-ponches.bat`
   - Run with highest privileges

**Estado:** Pendiente

---

### 8.3 Backup automático de base de datos

**Acciones:**

1. Crear carpeta `backups/` en el directorio de la aplicación

2. Crear script `backup-db.bat`:

```batch
@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

cd /d "%~dp0"

:: Crear carpeta backups si no existe
if not exist "backups" mkdir backups

:: Generar nombre con fecha YYYYMMDD
set DATE=%DATE:~-4%%DATE:~3,2%%DATE:~0,2%
set TIME=%TIME:~0,2%%TIME:~3,2%
set TIME=%TIME: =0%

:: Copiar base de datos
copy /Y "ponches.db" "backups\ponches_%DATE%_%TIME%.db" >nul

echo Backup completado: backups\ponches_%DATE%_%TIME%.db
```

3. Agendar tarea diaria en Task Scheduler

**Estado:** Pendiente

---

### 8.4 Documentación de operación

**Archivos a crear:**

1. `docs/GUIA_OPERACION.md` — pasos para operadores diarios
2. `docs/PROCEDIMIENTO_BACKUP.md` — cómo restaurar backups
3. `docs/CREDENCIALES.md` — plantilla para guardar credenciales (NO incluir datos reales)

**Estado:** Pendiente

---

## 🔧 Fase 9: Integración Dispositivo 10.0.0.100

### 9.1 Pruebas de conectividad ISAPI

**Pasos:**

1. Iniciar backend: `go run ./cmd/server`
2. Login en `http://localhost:8080` con admin/admin123
3. Navegar a Configuración → Dispositivos
4. Agregar dispositivo:
   - Nombre: "K1T343EWX Principal"
   - IP: `10.0.0.100`
   - Puerto: `80` (o `8000` si ISAPI no responde en 80)
   - Username: `___` (pendiente)
   - Password: `___` (pendiente)
5. Marcar como default
6. Verificar estado online en lista

**Criterio de aceptación:** Dispositivo aparece como "online" en la lista

**Estado:** Pendiente — requiere credenciales

---

### 9.2 Prueba de sincronización individual

**Pasos:**

1. Obtener un `employeeNo` válido de la DB (ej: de la tabla `employees`)
2. Ejecutar:
   ```bash
   POST /api/devices/configured/{id}/sync-one/{employeeNo}
   ```
3. Verificar respuesta exitosa:
   ```json
   {
     "message": "Empleado Juan Perez registrado en K1T343EWX Principal",
     "employeeNo": "00001",
     "device": "10.0.0.100"
   }
   ```
4. Verificar en dispositivo físico que el empleado aparece

**Criterio de aceptación:** Empleado aparece en la lista de usuarios del dispositivo

**Estado:** Pendiente — requiere credenciales + employeeNo válido

---

### 9.3 Prueba de sincronización masiva

**Pasos:**

1. Asegurar que hay empleados activos en la DB
2. Ejecutar:
   ```bash
   POST /api/devices/configured/{id}/sync
   ```
3. Verificar respuesta:
   ```json
   {
     "message": "Successfully synced employees to device",
     "count": 5,
     "device": "10.0.0.100"
   }
   ```
4. Validar en dispositivo que los usuarios aparecen

**Criterio de aceptación:** Múltiples empleados sincronizados correctamente

**Estado:** Pendiente — requiere credenciales

---

### 9.4 Prueba de eventos en tiempo real

**Pasos:**

1. Verificar que `EventListener` está corriendo en `main.go`
2. Ponchar en el dispositivo físico (tarjeta o reconocimiento facial)
3. Verificar que el evento aparece en WebSocket frontend (notificación toast)
4. Verificar que se guarda en tabla `attendance_events`

**Criterio de aceptación:** Evento se recibe en < 10 segundos y se persiste en DB

**Estado:** Pendiente — requiere dispositivo configurado

---

## 📊 Resumen de Archivos a Modificar

| Archivo | Cambio | Fase | Prioridad |
|---------|--------|------|-----------|
| `web/index.html` | Agregar `devices-page` completo | 7.3 | Alta |
| `web/app.js` | Funciones de gestión de dispositivos | 7.3 | Alta |
| `internal/store/sqlite.go` | Tabla `device_logs` | 7.2 | Media |
| `internal/store/store.go` | Interfaz `GetDeviceLogs` | 7.2 | Media |
| `internal/hikvision/listener.go` | Soporte de puerto configurable | 7.1 | Media |
| `cmd/server/main.go` | Iniciar listener con puerto del config | 7.1 | Media |
| `.env.production` | Variables para producción | 8.1 | Alta |
| `start-ponches.bat` | Script de inicio | 8.2 | Alta |
| `backup-db.bat` | Script de backup | 8.3 | Media |
| `docs/GUIA_OPERACION.md` | Documentación de operación | 8.4 | Alta |
| `docs/PROCEDIMIENTO_BACKUP.md` | Procedimiento de backup | 8.4 | Media |
| `docs/CREDENCIALES.md` | Plantilla de credenciales | 8.4 | Media |

---

## ✅ Verificación End-to-End

### Criterios de aceptación del proyecto:

- [ ] Backend inicia sin errores con `ponches.exe`
- [ ] Login funciona con usuario `admin` / `admin123`
- [ ] Página Dispositivos muestra lista vacía o con dispositivos
- [ ] Se puede agregar `10.0.0.100` con credenciales
- [ ] Estado online/offline se actualiza correctamente
- [ ] Sincronización individual de empleado funciona
- [ ] Sincronización masiva funciona (al menos 1 empleado)
- [ ] Eventos en tiempo real se reciben al ponchar
- [ ] Backup de DB se puede crear y restaurar
- [ ] Documentación está disponible para operadores

---

## 🔐 Datos Pendientes

Para completar las pruebas se necesita:

| Dato | Valor | Estado |
|------|-------|--------|
| Usuario ISAPI del dispositivo | `________________` | Pendiente |
| Contraseña ISAPI del dispositivo | `________________` | Pendiente |
| Puerto ISAPI (default 80) | `________________` | Pendiente |
| EmployeeNo para prueba | `________________` | Pendiente |

---

## 📅 Cronograma Estimado

| Fase | Tareas | Duración | Dependencias |
|------|--------|----------|--------------|
| **7.1** | Listener configurable | 1 hora | Ninguna |
| **7.2** | Logs persistentes | 1 hora | Ninguna |
| **7.3** | Frontend Dispositivos | 2-3 horas | Ninguna |
| **8.1** | Variables entorno | 30 min | Ninguna |
| **8.2** | Script inicio | 30 min | Ninguna |
| **8.3** | Script backup | 30 min | Ninguna |
| **8.4** | Documentación | 1 hora | Ninguna |
| **9.1-9.4** | Integración dispositivo | 2-4 horas | Credenciales |

**Total estimado:** 8-12 horas de trabajo

---

## 📌 Notas Importantes

1. **No usar Docker:** El proyecto usa `ponches.exe` directo en Windows (ver Fase 6 de `PLAN_TRABAJO.md`)

2. **Backup antes de cambios:** Copiar `ponches.db` antes de cualquier migración o cambio mayor

3. **JWT_SECRET:** Cambiar en producción — el default `ponches-secret-key-change-in-production` es inseguro

4. **Listener:** Si el dispositivo no envía eventos por polling (puerto 80), habilitar PushListener en puerto configurable (ej: 8081)

5. **Credenciales:** Nunca commitear `.env.production` con credenciales reales al repositorio

---

## 🔗 Referencias

- [PLAN_TRABAJO.md](../PLAN_TRABAJO.md) — Plan original del proyecto
- [HALLAZGOS_CONTINUIDAD_2026-04-21.md](./HALLAZGOS_CONTINUIDAD_2026-04-21.md) — Hallazgos de la sesión anterior
- [README.md](../README.md) — Documentación general del proyecto
- [docs/openapi.yaml](./openapi.yaml) — Especificación de la API
