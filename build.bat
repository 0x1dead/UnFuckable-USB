@echo off
REM UnFuckable USB Build Script for Windows
REM Author: 0x1dead

set VERSION=1.0.3
set APP_NAME=unfuckable-usb
set OUTPUT_DIR=dist

echo ================================================================
echo            UnFuckable USB - Build Script v%VERSION%
echo         Making your data impossible to fuck with
echo ================================================================
echo.

REM Create output directory
if not exist %OUTPUT_DIR% mkdir %OUTPUT_DIR%

REM Get dependencies
echo [*] Getting dependencies...
go mod tidy

REM Build flags
set LDFLAGS=-s -w -X main.AppVersion=%VERSION%

REM Compile Windows resources (icon)
set ICON_COMPILED=0

if exist icons\icon.ico (
    echo [*] Found icons\icon.ico, compiling Windows resources...
    
    REM Try rsrc first
    rsrc -ico icons\icon.ico -o rsrc.syso 2>nul
    if exist rsrc.syso (
        echo     OK Compiled rsrc.syso
        set ICON_COMPILED=1
        goto build
    )
    
    REM Try goversioninfo
    goversioninfo -icon=icons\icon.ico 2>nul
    if exist resource.syso (
        echo     OK Compiled resource.syso
        set ICON_COMPILED=1
        goto build
    )
    
    echo     [!] No resource compiler found
    echo     [!] Install: go install github.com/akavel/rsrc@latest
    echo     [!] Building without icon...
) else (
    echo [*] No icons\icon.ico found, building without icon
)

:build

REM Build for Windows AMD64
echo [*] Building for Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o %OUTPUT_DIR%\%APP_NAME%-windows-amd64.exe .
if %ERRORLEVEL% EQU 0 (
    echo     OK %OUTPUT_DIR%\%APP_NAME%-windows-amd64.exe
) else (
    echo     FAILED Windows amd64 build
)

REM Build for Windows ARM64
echo [*] Building for Windows (arm64)...
set GOARCH=arm64
go build -ldflags="%LDFLAGS%" -o %OUTPUT_DIR%\%APP_NAME%-windows-arm64.exe .
if %ERRORLEVEL% EQU 0 (
    echo     OK %OUTPUT_DIR%\%APP_NAME%-windows-arm64.exe
) else (
    echo     FAILED Windows arm64 build
)

REM Cleanup
if exist rsrc.syso del /q rsrc.syso
if exist resource.syso del /q resource.syso

echo.
echo [*] Build complete!
echo.

if %ICON_COMPILED% EQU 0 (
    echo [!] NOTE: Built without icon. To add icon:
    echo     go install github.com/akavel/rsrc@latest
    echo     Then run build.bat again
    echo.
)

REM Show files
echo Files:
dir /b %OUTPUT_DIR%\%APP_NAME%*

echo.
echo Done!
pause