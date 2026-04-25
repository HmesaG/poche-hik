@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==========================================
echo         INICIANDO PONCHES
echo         Control de Asistencia
echo ==========================================
echo.
echo Fecha: %DATE%
echo Hora: %TIME%
echo.
echo Iniciando servidor...
ponches.exe
echo.
echo El servidor se ha detenido.
pause
