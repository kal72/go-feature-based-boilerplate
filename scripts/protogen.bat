@echo off
setlocal enabledelayedexpansion

:: ─── Configuration ────────────────────────────────────────────────────────────
set PROTO_DIR=api\proto
set GEN_GO_DIR=gen\pb
set GEN_SWAGGER_DIR=gen\openapi

:: Resolve project root (scripts\ lives one level below root)
cd /d "%~dp0\.."

:: ─── Dependency check & auto-install ──────────────────────────────────────────
echo ==^> Checking protoc plugins...

where protoc >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: protoc not found. Install via https://grpc.io/docs/protoc-installation/
    exit /b 1
)

where protoc-gen-go >nul 2>&1
if %errorlevel% neq 0 (
    echo     Installing protoc-gen-go...
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
)

where protoc-gen-go-grpc >nul 2>&1
if %errorlevel% neq 0 (
    echo     Installing protoc-gen-go-grpc...
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
)

where protoc-gen-grpc-gateway >nul 2>&1
if %errorlevel% neq 0 (
    echo     Installing protoc-gen-grpc-gateway...
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
)

where protoc-gen-openapiv2 >nul 2>&1
if %errorlevel% neq 0 (
    echo     Installing protoc-gen-openapiv2...
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
)

:: ─── Prepare output directories ──────────────────────────────────────────────
echo ==^> Cleaning previous generated files...
if exist "%GEN_GO_DIR%" rmdir /s /q "%GEN_GO_DIR%"
if exist "%GEN_SWAGGER_DIR%" rmdir /s /q "%GEN_SWAGGER_DIR%"
mkdir "%GEN_GO_DIR%"
mkdir "%GEN_SWAGGER_DIR%"

:: ─── Collect proto files ──────────────────────────────────────────────────────
set PROTO_FILES=
for /r "%PROTO_DIR%" %%f in (*.proto) do (
    set PROTO_FILES=!PROTO_FILES! "%%f"
)

if "!PROTO_FILES!"=="" (
    echo WARNING: No .proto files found in %PROTO_DIR%
    exit /b 0
)

echo ==^> Generating Go code + gRPC + Gateway + Swagger...

:: ─── Run protoc ───────────────────────────────────────────────────────────────
for /f "delims=" %%i in ('go env GOPATH') do set GOPATH=%%i
set GATEWAY_PROTO=%GOPATH%\pkg\mod\github.com\grpc-ecosystem\grpc-gateway\v2@v2.29.0

protoc ^
  --proto_path=%PROTO_DIR%\plugins ^
  --proto_path=%PROTO_DIR% ^
  --proto_path=%GATEWAY_PROTO% ^
  --go_out=%GEN_GO_DIR% ^
  --go_opt=paths=source_relative ^
  --go-grpc_out=%GEN_GO_DIR% ^
  --go-grpc_opt=paths=source_relative ^
  --grpc-gateway_out=%GEN_GO_DIR% ^
  --grpc-gateway_opt=paths=source_relative ^
  --openapiv2_out=%GEN_SWAGGER_DIR% ^
  --openapiv2_opt=logtostderr=true ^
  --openapiv2_opt=allow_merge=true ^
  --openapiv2_opt=merge_file_name=api ^
  !PROTO_FILES!

if %errorlevel% neq 0 (
    echo ERROR: protoc failed
    exit /b 1
)

:: ─── Generate openapi.go wrapper ─────────────────────────────────────────────
(
echo package openapi
echo.
echo import _ "embed"
echo.
echo // Spec contains the raw JSON bytes of the generated OpenAPI / Swagger spec.
echo //go:embed api.swagger.json
echo var Spec []byte
) > "%GEN_SWAGGER_DIR%\openapi.go"

:: ─── Summary ──────────────────────────────────────────────────────────────────
echo.
echo ==^> Done!
echo     Go code  -^> %GEN_GO_DIR%\
echo     Swagger  -^> %GEN_SWAGGER_DIR%\api.swagger.json
echo.
echo     Generated files:
for /r "%GEN_GO_DIR%" %%f in (*.go) do echo       %%f
for /r "%GEN_SWAGGER_DIR%" %%f in (*.json) do echo       %%f
for /r "%GEN_SWAGGER_DIR%" %%f in (openapi.go) do echo       %%f

endlocal

