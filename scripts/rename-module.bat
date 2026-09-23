@echo off
setlocal enabledelayedexpansion

:: Resolve project root
cd /d "%~dp0\.."

if not exist "go.mod" (
    echo Error: go.mod not found.
    exit /b 1
)

:: Get current module name
for /f "tokens=2" %%i in ('findstr /r "^module " go.mod') do (
    set OLD_MODULE=%%i
    goto :found_module
)
:found_module

set NEW_MODULE=%1
if "%NEW_MODULE%"=="" (
    echo Current module: !OLD_MODULE!
    set /p NEW_MODULE="Enter new module name (e.g. github.com/username/my-service): "
)

if "%NEW_MODULE%"=="" (
    echo Error: New module name cannot be empty.
    exit /b 1
)

if "!OLD_MODULE!"=="!NEW_MODULE!" (
    echo Module is already named !NEW_MODULE!. Nothing to do.
    exit /b 0
)

echo.
echo ================================================================================
echo   Renaming Go Module:
echo     Old: !OLD_MODULE!
echo     New: !NEW_MODULE!
echo ================================================================================
echo.

:: Run powershell replace script for cross-platform robustness
powershell -NoProfile -Command ^
    "$old = '!OLD_MODULE!';" ^
    "$new = '!NEW_MODULE!';" ^
    "Get-ChildItem -Path . -Include *.go,*.proto,Makefile,.env.example,README.md -Recurse -File | Where-Object { $_.FullName -notmatch '\\\.git\\|\\bin\\' } | ForEach-Object {" ^
    "    (Get-Content -Path $_.FullName -Raw) -replace [regex]::Escape($old), $new | Set-Content -Path $_.FullName -NoNewline;" ^
    "}"

:: Update go.mod
go mod edit -module !NEW_MODULE!

:: Regenerate
call scripts\protogen.bat
where wire >nul 2>&1 && wire .\bootstrap\...
go mod tidy

echo.
echo ================================================================================
echo   Successfully renamed module to: !NEW_MODULE!
echo ================================================================================
echo.
