@echo off
REM Release build for Windows x86_64 only (no CI service involved).
REM Usage: double-click, or run `build-windows.bat` from cmd/PowerShell.
setlocal
cd /d "%~dp0"

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

if not exist dist mkdir dist

echo [release] go build -> dist\assetctl.exe
go build -trimpath -ldflags "-s -w" -o dist\assetctl.exe .\cmd\assetctl
if errorlevel 1 (
  echo [release] BUILD FAILED
  exit /b 1
)

for %%A in (dist\assetctl.exe) do echo [release] built dist\assetctl.exe ^(%%~zA bytes^)
echo [release] self-check:
dist\assetctl.exe version
dist\assetctl.exe skills doctor

endlocal
