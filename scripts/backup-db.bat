@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

cd /d "%~dp0"

:: ==========================================
:: BACKUP AUTOMÁTICO DE BASE DE DATOS
:: Ponches - Control de Asistencia
:: ==========================================

:: Crear carpeta backups si no existe
if not exist "backups" mkdir backups

:: Generar nombre con fecha YYYYMMDD_HHMM
set DATE=%DATE:~-4%%DATE:~3,2%%DATE:~0,2%
set TIME=%TIME:~0,2%%TIME:~3,2%
set TIME=%TIME: =0%

:: Copiar base de datos
copy /Y "ponches.db" "backups\ponches_%DATE%_%TIME%.db" >nul

if %ERRORLEVEL% EQU 0 (
    echo ==========================================
    echo BACKUP COMPLETADO EXITOSAMENTE
    echo ==========================================
    echo Archivo: backups\ponches_%DATE%_%TIME%.db
    echo Fecha: %DATE%
    echo Hora: %TIME%
    echo ==========================================
) else (
    echo ERROR: No se pudo crear el backup
    echo Verifique que ponches.db no esté en uso
)
