# Procedimiento de Backup y Restauración — Ponches

**Última actualización:** 2026-04-21  
**Responsable:** Administrador del Sistema

---

## 📋 Descripción

Este documento describe el procedimiento para realizar backups de la base de datos y restaurarla en caso de emergencia.

**Base de datos:** `ponches.db` (SQLite)  
**Ubicación por defecto:** `C:\GrupoMV\Ponches\ponches.db`

---

## 🔄 Backup Automático (Recomendado)

### Configuración Inicial

1. **Crear carpeta de backups:**
   ```
   C:\GrupoMV\Ponches\backups\
   ```

2. **Configurar Task Scheduler:**
   - Abrir **Task Scheduler** (Programador de Tareas)
   - Clic en **Create Basic Task**
   - Nombre: "Backup Ponches Diario"
   - Trigger: **Daily** a las 18:00 (después del turno)
   - Action: **Start a program**
   - Program: `C:\GrupoMV\Ponches\backup-db.bat`
   - Marcar: "Run with highest privileges"

### Verificación del Backup

1. Ir a `C:\GrupoMV\Ponches\backups\`
2. Verificar que exista un archivo por día: `ponches_YYYYMMDD_HHMM.db`
3. Verificar tamaño del archivo (> 0 bytes)

---

## 📥 Backup Manual

### Opción 1: Usando el script

1. Abrir CMD o PowerShell
2. Navegar a `C:\GrupoMV\Ponches\`
3. Ejecutar:
   ```batch
   backup-db.bat
   ```

### Opción 2: Copia directa

1. Detener el servicio Ponches (Ctrl+C en la consola)
2. Copiar `ponches.db` a una ubicación segura
3. Reiniciar el servicio

---

## 🔁 Restauración de Backup

### ⚠️ Advertencias

- **NUNCA** restaurar un backup mientras el sistema está corriendo
- **SIEMPRE** crear un backup del estado actual antes de restaurar
- La restauración es **DESTRUCTIVA** — se pierden los datos posteriores al backup

### Pasos de Restauración

1. **Detener el servicio:**
   - Si está en consola: Presionar `Ctrl+C`
   - Si es servicio Windows: Detener desde Services.msc

2. **Crear backup del estado actual:**
   ```batch
   copy ponches.db ponches.db.pre-restore.YYYYMMDD
   ```

3. **Restaurar el backup:**
   ```batch
   copy /Y backups\ponches_20260420_1800.db ponches.db
   ```

4. **Verificar la restauración:**
   - Iniciar el sistema: `start-ponches.bat`
   - Abrir `http://localhost:8080`
   - Verificar que los datos aparezcan correctamente

5. **Notificar a los usuarios:**
   - Informar que el sistema fue restaurado
   - Indicar qué datos pueden haberse perdido (eventos posteriores al backup)

---

## 📊 Política de Retención Recomendada

| Tipo | Frecuencia | Retención |
|------|------------|-----------|
| **Diario** | Todos los días 18:00 | 7 días |
| **Semanal** | Domingos | 4 semanas |
| **Mensual** | Último día del mes | 6 meses |

### Script de Limpieza (Opcional)

Crear `cleanup-backups.bat`:

```batch
@echo off
cd /d "%~dp0backups"

:: Eliminar backups de más de 30 días
forfiles /p "%~dp0backups" /s /m ponches_*.db /d -30 /c "cmd /c del @path"

echo Backups antiguos eliminados
```

---

## 🧪 Verificación de Integridad

### Mensual (Recomendado)

1. Copiar un backup reciente a una carpeta de prueba
2. Iniciar Ponches apuntando a esa copia:
   - Editar `.env`: `DB_PATH=C:\temp\prueba-ponches.db`
3. Verificar que:
   - El login funciona
   - Los empleados aparecen
   - Los eventos de asistencia están
4. Restaurar configuración original

---

## 📋 Checklist de Backup

### Diario
- [ ] Verificar que el backup automático se creó
- [ ] Confirmar tamaño > 0 bytes

### Semanal
- [ ] Revisar que hay backups de los últimos 7 días
- [ ] Eliminar backups muy antiguos (>30 días)

### Mensual
- [ ] Verificar integridad de un backup aleatorio
- [ ] Documentar cualquier incidencia

---

## 🆘 Emergencias

### La base de datos está corrupta

1. Detener el servicio inmediatamente
2. Renombrar `ponches.db` a `ponches.db.corrupt`
3. Restaurar el backup más reciente
4. Notificar al equipo técnico

### El servidor falló completamente

1. Instalar Go 1.25 en el servidor de respaldo
2. Clonar/copiar el repositorio de Ponches
3. Copiar el backup más reciente como `ponches.db`
4. Copiar `.env.production` como `.env`
5. Ejecutar `go run ./cmd/server`

---

## 📞 Contactos

| Rol | Nombre | Teléfono |
|-----|--------|----------|
| Administrador | | |
| Soporte Técnico | | |

---

## 📎 Archivos Relacionados

- `backup-db.bat` — Script de backup automático
- `cleanup-backups.bat` — Script de limpieza (opcional)
- `.env.production` — Configuración de producción
