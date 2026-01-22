# MarchProxy OBS Plugin Installer for Windows

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ScriptName = "marchproxy-stream.lua"
$OBSScriptsDir = "$env:APPDATA\obs-studio\scripts"

Write-Host "MarchProxy OBS Plugin Installer"
Write-Host "================================"
Write-Host ""

# Check if OBS is installed
$OBSInstalled = Test-Path "C:\Program Files\obs-studio" -or Test-Path "$env:LOCALAPPDATA\Programs\obs-studio"
if (-not $OBSInstalled) {
    Write-Host "Warning: OBS Studio doesn't appear to be installed." -ForegroundColor Yellow
    Write-Host "The script will be copied anyway, but you'll need OBS to use it."
    Write-Host ""
}

# Create scripts directory if it doesn't exist
if (-not (Test-Path $OBSScriptsDir)) {
    Write-Host "Creating OBS scripts directory..."
    New-Item -ItemType Directory -Path $OBSScriptsDir -Force | Out-Null
}

# Copy the script
Write-Host "Installing $ScriptName to $OBSScriptsDir..."
Copy-Item -Path "$ScriptDir\$ScriptName" -Destination $OBSScriptsDir -Force

# Verify installation
if (Test-Path "$OBSScriptsDir\$ScriptName") {
    Write-Host ""
    Write-Host "Installation successful!" -ForegroundColor Green
    Write-Host ""
    Write-Host "To enable the plugin:"
    Write-Host "1. Open OBS Studio"
    Write-Host "2. Go to Tools -> Scripts"
    Write-Host "3. Click '+' and select '$ScriptName'"
    Write-Host "4. Configure your MarchProxy settings"
    Write-Host ""
} else {
    Write-Host "Error: Installation failed" -ForegroundColor Red
    exit 1
}
