@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Building OlicanaPlot and Plugins
echo ========================================

set ROOT_DIR=%~dp0
cd /d "%ROOT_DIR%"

echo.
echo Cleaning up running processes...
taskkill /F /IM OlicanaPlot.exe /T >nul 2>&1
taskkill /F /IM csv.exe /T >nul 2>&1
taskkill /F /IM model_selector.exe /T >nul 2>&1
taskkill /F /IM random_walk_generator.exe /T >nul 2>&1
taskkill /F /IM synthetic_data_generator.exe /T >nul 2>&1
echo Done.

echo.
echo [1/5] Building Main Application...
call wails3 build
if %errorlevel% neq 0 (
    echo Error building main application.
    exit /b %errorlevel%
)

echo.
echo [2/5] Building Random Walk Generator (C++ Plugin)...
cd /d "%ROOT_DIR%plugins\random_walk_generator"
if exist build.bat (
    call build.bat
) else (
    echo Warning: random_walk_generator\build.bat not found.
)

echo.
echo [3/5] Building CSV IPC (Go Plugin)...
cd /d "%ROOT_DIR%plugins\csv"
if exist build.bat (
    call build.bat
) else (
    echo Warning: csv\build.bat not found.
)

echo.
echo [4/5] Building Synthetic Data Generator (Wails Plugin)...
cd /d "%ROOT_DIR%plugins\synthetic_data_generator"
call wails3 build
if %errorlevel% neq 0 (
    echo Warning: Error building synthetic_data_generator.
)

echo.
echo [5/5] Building Model Selector (Go IPC Plugin)...
cd /d "%ROOT_DIR%plugins\model_selector"
go build -o model_selector.exe main.go
if %errorlevel% neq 0 (
    echo Warning: Error building model_selector.
)

echo.
echo ========================================
echo Build Complete
echo ========================================
cd /d "%ROOT_DIR%"

