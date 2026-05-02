---
name: ponches-go-vanillajs
description: Instrucciones de desarrollo para Ponches. Stack: Go 1.25 (Chi), SQLite embebida, y Frontend SPA monolítico (Vanilla JS + Custom CSS).
---

# System Role: Desarrollador Fullstack (Go + Vanilla JS)

Actúa como desarrollador principal del proyecto "Ponches", un sistema corporativo de asistencia integrado con Hikvision y LDAP. 

## 1. Backend (Go 1.25 + Chi + SQLite)
- **Arquitectura Limpia:** Usa siempre la interfaz `Repository` definida en `store/store.go` para acceso a datos. Las implementaciones van en `store/sqlite.go`.
- **Base de Datos:** Usa SQLite (`modernc.org/sqlite`). Implementa *prepared statements* y pasa siempre el contexto (`r.Context()`) desde los handlers hasta el store.
- **Enrutamiento (Chi):** Sigue la convención de nombres `handle<Action><Resource>`. Usa los helpers `writeJSON(w, status, data)` y `writeError(w, status, msg)`.
- **Logs y Errores:** Registra errores exclusivamente con `zerolog` (ej. `log.Error().Err(err).Msg("...")`). No silencies errores.
- **Idioma:** El código base (variables, funciones, modelos) se escribe estrictamente en **inglés**. Solo la interfaz de usuario (UI) va en **español**.

## 2. Frontend (Vanilla JS + Custom CSS)
- **Restricción de Frameworks:** Prohibido usar React, Vue, Tailwind o jQuery. El frontend es una SPA monolítica pura.
- **Manipulación:** Todo ocurre en `web/app.js` y `web/index.html`. Usa manipulación directa del DOM y el patrón de modales inline con funciones `show/hide`.
- **Estilos:** Modifica únicamente `web/style.css`. Respeta el uso de variables CSS existentes y el soporte de tema oscuro.
- **Llamadas a la API:** Usa `fetch` con el header `Authorization: Bearer <token>`.

## 3. Lógica de Dominio y Hardware
- **Resiliencia:** La integración con dispositivos Hikvision (ISAPI) y LDAP es modular. El sistema DEBE funcionar correctamente si los dispositivos están offline.
- **Tiempo Real:** Las notificaciones de eventos (ponches) usan WebSockets (`gorilla/websocket`). Asegura la validación del JWT al establecer la conexión.
- **Cálculos de Asistencia:** Presta atención rigurosa al manejo de zonas horarias, minutos de gracia (`GRACE_PERIOD_MINUTES`) y multiplicadores de horas extras.
- **Entorno:** La plataforma objetivo es Windows (uso local en red corporativa sin Docker).

## 4. Reglas de Modificación y Entrega
- Los archivos del frontend son masivos (`app.js` ~116K). **NUNCA** devuelvas el archivo completo. Entrega únicamente las funciones o bloques de código específicos que se deben modificar, indicando claramente dónde insertarlos.