@echo off
setlocal enabledelayedexpansion

:: Install Wails
echo Installing Wails framework...
go install github.com/wailsapp/wails/v2/cmd/wails@latest
if %ERRORLEVEL% NEQ 0 (
    echo Failed to install Wails framework
    exit /b 1
)
echo Wails installed successfully!

:: Create base directory (equivalent to LOCALAPPDATA\Kube)
set "BASE_DIR=%LOCALAPPDATA%\Kube"
mkdir "%BASE_DIR%" 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Failed to create base directory: %BASE_DIR%
    exit /b 1
)

:: Create models subdirectory
set "MODELS_DIR=%BASE_DIR%\models"
mkdir "%MODELS_DIR%" 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Failed to create models directory: %MODELS_DIR%
    exit /b 1
)

:: Get script directory to locate f_model
set "SCRIPT_DIR=%~dp0"
set "SOURCE_FILE=%SCRIPT_DIR%f_model\test.xml"

:: Copy test.xml file to models directory
copy "%SOURCE_FILE%" "%MODELS_DIR%\" >nul
if %ERRORLEVEL% NEQ 0 (
    echo Failed to copy %SOURCE_FILE% to %MODELS_DIR%
    exit /b 1
)

echo Installation completed successfully!
echo Base directory: %BASE_DIR%
echo Models directory: %MODELS_DIR%

endlocal
