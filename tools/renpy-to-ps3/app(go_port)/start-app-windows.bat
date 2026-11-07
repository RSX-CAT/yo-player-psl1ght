@echo off
setlocal
cd /d "%~dp0go"

where go >nul 2>&1
if errorlevel 1 goto :nogo
go version >nul 2>&1
if errorlevel 1 goto :nogo
goto :build

:nogo
echo Go is not installed, or it is not on PATH.
echo.
echo Downloading the Go 1.20.14 installer.
echo This build runs on Windows 7 and Windows 11.
echo.
set "GO_MSI=%TEMP%\go1.20.14.windows-amd64.msi"
certutil -urlcache -split -f "https://dl.google.com/go/go1.20.14.windows-amd64.msi" "%GO_MSI%"
if errorlevel 1 goto :manual
echo.
echo Starting the installer. Approve it if Windows asks.
echo When it finishes, close this window and run start-app-windows.bat again.
echo.
start "" "%GO_MSI%"
exit /b 1

:manual
echo Could not download the installer.
echo Install Go yourself, then run start-app-windows.bat again:
echo   https://go.dev/dl/go1.20.14.windows-amd64.msi
echo Windows 7 must use 1.20.14. Windows 11 can use 1.20.14 or a newer Go.
exit /b 1

:build
echo Building renpy-to-ps3.exe ...
go build -o renpy-to-ps3.exe .\cmd\renpy-to-ps3
if errorlevel 1 exit /b 1

if exist "..\..\ffmpeg.exe" copy /Y "..\..\ffmpeg.exe" ".\ffmpeg.exe" >nul

echo.
echo Starting local UI at http://127.0.0.1:8765/
echo Close this window to quit.
echo.
.\renpy-to-ps3.exe ui
