# Guía de Operación Diaria — Ponches

**Última actualización:** 2026-04-21  
**Versión:** 1.0

---

## 📋 ¿Qué es Ponches?

Ponches es el sistema de control de asistencia de Grupo MV. Permite:
- Registrar entradas y salidas de empleados mediante dispositivos biométricos Hikvision
- Calcular horas extras, tardanzas y asistencias
- Generar reportes de nómina en PDF y Excel
- Gestionar departamentos, cargos y viáticos

---

## 🚀 Inicio del Sistema

### Opción 1: Inicio Manual

1. Abrir el Explorador de Archivos
2. Navegar a `C:\GrupoMV\Ponches\`
3. Doble clic en `start-ponches.bat`
4. Esperar a que aparezca el mensaje "Server starting on port 8080"
5. Abrir navegador en `http://localhost:8080` o `http://<IP-del-servidor>:8080`

### Opción 2: Inicio Automático

Si está configurado en el Task Scheduler, el sistema inicia automáticamente al encender el servidor.

---

## 🔐 Acceso al Sistema

**URL:** `http://localhost:8080` (local) o `http://<IP-servidor>:8080` (red)

**Credenciales por defecto:**
- Usuario: `admin`
- Contraseña: `admin123`

⚠️ **Importante:** Cambiar la contraseña de admin en la primera sesión.

---

## 📱 Roles de Usuario

| Rol | Permisos |
|-----|----------|
| **Admin** | Acceso completo: usuarios, configuración, empleados, reportes |
| **Manager** | Gestionar empleados, ver asistencias, generar reportes |
| **Viewer** | Solo lectura de empleados y reportes |

---

## 🏭 Operación Diaria

### 1. Verificar que el sistema esté funcionando

- Abrir `http://localhost:8080`
- Verificar que aparezca el dashboard
- Revisar que el contador de empleados activos sea correcto

### 2. Monitorear eventos en tiempo real

- Los eventos de ponche aparecen automáticamente en la sección **Asistencia**
- Cada vez que un empleado poncha, se muestra una notificación toast

### 3. Registrar un nuevo empleado

1. Ir a **Empleados** → **Nuevo Empleado**
2. Llenar datos obligatorios:
   - Número de empleado (único)
   - Nombre y apellido
   - Departamento y cargo (opcionales)
3. Guardar

### 4. Sincronizar empleado con dispositivo

1. Ir a **Empleados**
2. Buscar el empleado
3. Clic en **Acceso** (icono de tarjeta)
4. Seleccionar dispositivo o usar el predeterminado
5. Esperar confirmación

### 5. Generar reporte de asistencia

1. Ir a **Reportes**
2. Seleccionar tipo de reporte:
   - **Diario:** Asistencias del día
   - **Nómina:** Horas trabajadas, extras, tardanzas
   - **Tardanzas:** Reporte específico de llegadas tarde
   - **Excel:** Exportación masiva
3. Seleccionar rango de fechas
4. Clic en **Generar**

---

## 🔧 Sección Dispositivos

### Agregar un dispositivo

1. Ir a **Configuración** → **Dispositivos**
2. Clic en **+ Nuevo Dispositivo**
3. Llenar:
   - Nombre: Ej: "K1T343EWX Principal"
   - IP: Ej: `10.0.0.100`
   - Puerto: `80` (o el que use el dispositivo)
   - Usuario y contraseña ISAPI
4. Marcar "Dispositivo predeterminado" si es el principal
5. Guardar

### Sincronizar todos los empleados

1. Ir a **Dispositivos**
2. Clic en **Sincronizar Empleados**
3. Esperar confirmación (puede tardar varios segundos)

### Ver logs de errores

1. Ir a **Dispositivos**
2. Bajar a "Centro de Salud y Sync"
3. Revisar tabla de logs
4. Los errores aparecen en rojo

---

## ⚠️ Solución de Problemas Comunes

### El sistema no inicia

1. Verificar que el puerto 8080 no esté en uso
2. Ejecutar `start-ponches.bat` como administrador
3. Revisar que `ponches.db` no esté bloqueado

### Los empleados no aparecen en el dispositivo

1. Verificar que el dispositivo esté online en **Dispositivos**
2. Confirmar credenciales ISAPI correctas
3. Reintentar sincronización individual

### No llegan eventos en tiempo real

1. Verificar que el dispositivo tenga conexión de red
2. Revisar logs en **Dispositivos** → **Centro de Salud**
3. Reiniciar el servicio

### Olvidé la contraseña de admin

1. Detener el servicio (Ctrl+C en la consola)
2. Contactar al desarrollador para reset

---

## 📞 Soporte

Para problemas técnicos o preguntas, contactar al equipo de desarrollo.

---

## 📄 Archivos Importantes

| Archivo | Función |
|---------|---------|
| `ponches.db` | Base de datos principal |
| `.env` | Configuración del sistema |
| `start-ponches.bat` | Script de inicio |
| `backup-db.bat` | Script de backup |
| `backups/` | Carpeta de backups automáticos |
