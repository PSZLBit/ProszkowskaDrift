@echo off
setlocal
cd /d "%~dp0"
REM Install the repository-local Git hooks on Windows.
git config --local core.hooksPath .githooks
if errorlevel 1 exit /b 1
echo Local Git hooks installed from .githooks.
echo Verify with: git config --local --get core.hooksPath
