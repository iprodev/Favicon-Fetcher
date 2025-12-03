@echo off
REM Build script for Windows

echo ========================================
echo Building Favicon Fetcher (Windows)
echo Full format support: PNG, WebP, AVIF
echo ========================================

REM Disable CGO for static binary
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

REM Clean previous builds
if exist favicon-server.exe del favicon-server.exe
if exist bin\favicon-server.exe del bin\favicon-server.exe

REM Build
echo.
echo Compiling...
go build -v -ldflags="-s -w" -o favicon-server.exe ./cmd/server

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo Build successful!
    echo ========================================
    echo.
    echo Binary: favicon-server.exe
    echo Formats: PNG, WebP, AVIF
    echo.
    echo To run:
    echo   favicon-server.exe
    echo.
    echo To run with custom settings:
    echo   favicon-server.exe -port 8080 -log-level debug
    echo.
) else (
    echo.
    echo ========================================
    echo Build FAILED!
    echo ========================================
    echo.
    exit /b 1
)
