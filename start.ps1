$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

if (Test-Path "$scriptDir\gocraft.exe") {
    Write-Host "Starting GoCraft 3D..." -ForegroundColor Cyan
    & "$scriptDir\gocraft.exe"
} elseif (Test-Path "$scriptDir\minecraft.exe") {
    Write-Host "Starting GoCraft 3D..." -ForegroundColor Cyan
    & "$scriptDir\minecraft.exe"
} else {
    Write-Host "Compiling GoCraft 3D..." -ForegroundColor Yellow
    $env:PATH = "C:\Program Files\Go\bin;" + $env:PATH
    $env:CGO_ENABLED = "0"
    go build -ldflags "-s -w" -o gocraft.exe .
    & "$scriptDir\gocraft.exe"
}

