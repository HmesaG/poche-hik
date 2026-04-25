# Hallazgos de Continuidad - 2026-04-21

## Objetivo de esta nota

Guardar el contexto operativo de la sesion para retomarlo despues sin volver a reconstruir el estado del proyecto ni el objetivo de pruebas con dispositivo real.

## Dispositivo objetivo para pruebas reales

- URL informada por el usuario: `http://10.0.0.100/#/login`
- Verificacion realizada el 2026-04-21:
  - `GET http://10.0.0.100/` respondio `HTTP 200`
  - La conectividad fue confirmada fuera del sandbox local
- Implicacion:
  - El dispositivo existe y es alcanzable por red desde esta maquina
  - La siguiente fase debe hacerse contra `10.0.0.100`, no con una IP de ejemplo

## Hallazgos tecnicos ya ubicados en el codigo

### 1. Gestion de dispositivos configurados

Archivo principal: `internal/api/handlers_devices.go`

- Los dispositivos administrados se guardan en configuracion persistente bajo la clave `managed_devices`
- Al marcar uno como default, el backend actualiza tambien:
  - `hikvision_ip`
  - `hikvision_username`
  - `hikvision_password`
- Esa actualizacion ocurre en `applyDefaultDevice(...)`

### 2. Endpoints existentes para trabajar con el dispositivo

Definidos en `internal/api/router.go`:

- `GET /api/devices/configured`
- `POST /api/devices/configured`
- `PUT /api/devices/configured/{id}`
- `DELETE /api/devices/configured/{id}`
- `POST /api/devices/configured/{id}/default`
- `POST /api/devices/configured/{id}/sync`
- `POST /api/devices/configured/{id}/sync-one/{employeeNo}`
- `GET /api/devices/configured/{id}/logs`

### 3. Listener en tiempo real

Archivo: `cmd/server/main.go`

- El listener se inicia si existen `cfg.HikvisionIP` y `cfg.HikvisionUsername`
- Usa `hikvision.NewEventListener(cfg.HikvisionIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)`
- Importante: ahora mismo el listener esta fijado a puerto `80`
- Si el equipo real usa otro puerto ISAPI, habra que ajustar ese punto

### 4. Cliente de validacion del dispositivo

Archivo: `internal/hikvision/device.go`

- `GetDeviceInfo()` consume `/ISAPI/System/deviceInfo`
- Esta llamada se usa para chequeos rapidos de conectividad/estado en la lista de dispositivos

## Estado actual del repo al cerrar esta sesion

Hay cambios locales sin commit. No tocarlos ni revertirlos al retomar sin revisar contexto.

Archivos modificados observados:

- `.gitignore`
- `cmd/seed/main.go`
- `internal/api/handlers_config.go`
- `internal/api/handlers_devices.go`
- `internal/api/handlers_reports.go`
- `internal/api/router.go`
- `internal/attendance/engine_test.go`
- `internal/config/config.go`
- `internal/hikvision/listener.go`
- `internal/hikvision/users.go`
- `internal/store/sqlite.go`
- `internal/store/store.go`
- `web/app.js`
- `web/index.html`
- `web/style.css`

## Siguiente ruta recomendada al retomar

1. Levantar el backend local.
2. Iniciar sesion en la app con el usuario de trabajo.
3. Registrar `10.0.0.100` como dispositivo configurado con sus credenciales reales.
4. Marcarlo como dispositivo por defecto.
5. Validar que `GET /api/devices/configured` lo muestre en linea.
6. Probar `sync-one` con un empleado real de prueba antes de lanzar sincronizacion masiva.
7. Revisar logs del dispositivo desde `GET /api/devices/configured/{id}/logs`.
8. Confirmar si el listener de eventos en tiempo real recibe eventos reales desde ese equipo.

## Datos que aun faltan para la prueba completa

- Usuario del dispositivo
- Contrasena del dispositivo
- Confirmacion de si el ISAPI responde en puerto `80` o en otro puerto
- Un `employeeNo` valido para prueba de `sync-one`

## Riesgos conocidos

- El frontend del equipo responde por HTTP, pero eso no garantiza todavia autenticacion ISAPI correcta
- Si el dispositivo usa credenciales distintas entre web UI e ISAPI, la prueba puede fallar aunque la pagina cargue
- El listener actual usa puerto fijo `80`; si el equipo real no expone eventos ahi, no habra eventos en tiempo real aunque el dispositivo exista

## Decision operativa tomada

Para continuar, el objetivo oficial de pruebas queda fijado en:

- `http://10.0.0.100/#/login`

Y cualquier validacion de integracion Hikvision pendiente debe comprobarse contra ese dispositivo.
