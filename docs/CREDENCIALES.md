# Plantilla de Credenciales — Ponches

**⚠️ ADVERTENCIA CRÍTICA:** Este archivo es una **PLANTILLA**. **NUNCA** guardar credenciales reales en este documento.

---

## 📋 Propósito

Este documento sirve como plantilla para que el administrador del sistema registre las credenciales críticas de la operación de Ponches.

**Ubicación recomendada:** Imprimir y guardar en caja fuerte, o usar un gestor de contraseñas como Bitwarden/1Password.

---

## 🔐 Credenciales del Sistema

### Usuario Admin Principal

| Campo | Valor |
|-------|-------|
| Usuario | `admin` |
| Contraseña | ________________________________ |
| Email asociado | ________________________________ |
| Fecha de creación | ___/___/_____ |
| Última rotación | ___/___/_____ |

---

## 🔌 Dispositivos Hikvision

### Dispositivo Principal (10.0.0.100)

| Campo | Valor |
|-------|-------|
| IP | `10.0.0.100` |
| Modelo | ________________________________ |
| Número de Serie | ________________________________ |
| Usuario ISAPI | ________________________________ |
| Contraseña ISAPI | ________________________________ |
| Puerto | `80` / `8000` / Otro: ______ |
| Ubicación física | ________________________________ |
| Fecha de instalación | ___/___/_____ |

### Dispositivo Secundario (si aplica)

| Campo | Valor |
|-------|-------|
| IP | ________________________________ |
| Modelo | ________________________________ |
| Número de Serie | ________________________________ |
| Usuario ISAPI | ________________________________ |
| Contraseña ISAPI | ________________________________ |
| Puerto | ______ |
| Ubicación física | ________________________________ |

---

## 🗄️ Base de Datos

| Campo | Valor |
|-------|-------|
| Ruta | `C:\GrupoMV\Ponches\ponches.db` |
| Tipo | SQLite (embebido) |
| Backup automático | Sí / No |
| Hora de backup | ______:______ |
| Ubicación backups | `C:\GrupoMV\Ponches\backups\` |

---

## 🔑 JWT Secret (Producción)

**Generar con:** `openssl rand -hex 32`

```
Secret: _________________________________________________________________
        _________________________________________________________________
        
Fecha de generación: ___/___/_____
Generado por: ____________________
```

---

## 🌐 LDAP / Active Directory (si aplica)

| Campo | Valor |
|-------|-------|
| Host | ________________________________ |
| Puerto | `389` / `636` (SSL) |
| Base DN | ________________________________ |
| Bind DN | ________________________________ |
| Bind Password | ________________________________ |
| Usuario de servicio | ________________________________ |

---

## 🖥️ Servidor

| Campo | Valor |
|-------|-------|
| Hostname | ________________________________ |
| IP del servidor | ________________________________ |
| Sistema Operativo | Windows Server ______ |
| Usuario administrador | ________________________________ |
| Contraseña | ________________________________ |
| Acceso remoto | RDP / TeamViewer / Otro: ______ |

---

## 📞 Contactos de Emergencia

| Rol | Nombre | Teléfono | Email |
|-----|--------|----------|-------|
| Administrador de Sistema | | | |
| Soporte Técnico | | | |
| Proveedor Hikvision | | | |
| Responsable Nómina | | | |

---

## 📝 Historial de Cambios

| Fecha | Cambio | Realizado por |
|-------|--------|---------------|
| | | |
| | | |
| | | |

---

## 🔒 Instrucciones de Seguridad

1. **Nunca** almacenar este archivo completado en la computadora
2. **Imprimir** y guardar en caja fuerte o lugar seguro
3. **Actualizar** después de cada cambio de credenciales
4. **Rotar** contraseñas cada 90 días mínimo
5. **Limitar acceso** solo al personal autorizado

---

**Fecha de última actualización:** ___/___/_____  
**Responsable:** ________________________________
