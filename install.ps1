#Requires -Version 5.1
<#
.SYNOPSIS
    Assern Windows Install Script - Installs Assern inside WSL

.DESCRIPTION
    This script verifies WSL2 is configured, then installs Assern inside your
    WSL Linux distribution using the standard install.sh script.

.PARAMETER Nightly
    Install the latest nightly build instead of stable release.

.PARAMETER Version
    Install a specific version (e.g., "v1.2.3").

.EXAMPLE
    irm https://raw.githubusercontent.com/valksor/go-assern/master/install.ps1 | iex

.LINK
    https://github.com/valksor/go-assern
#>

[CmdletBinding()]
param(
    [switch]$Nightly,
    [string]$Version
)

if ($PSVersionTable.PSVersion.Major -lt 5 -or
    ($PSVersionTable.PSVersion.Major -eq 5 -and $PSVersionTable.PSVersion.Minor -lt 1)) {
    Write-Host "[ERROR] PowerShell 5.1+ required" -ForegroundColor Red
    exit 1
}

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$InstallScriptUrl = "https://raw.githubusercontent.com/valksor/go-assern/master/install.sh"
$DocsUrl = "https://valksor.com/docs/assern"
$MinWindowsBuild = 19041

function Write-ColorOutput {
    param([string]$Message, [string]$Type = "Info")
    switch ($Type) {
        "Info"    { Write-Host "[INFO] " -ForegroundColor Blue -NoNewline; Write-Host $Message }
        "Success" { Write-Host "[OK] " -ForegroundColor Green -NoNewline; Write-Host $Message }
        "Warning" { Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline; Write-Host $Message }
        "Error"   { Write-Host "[ERROR] " -ForegroundColor Red -NoNewline; Write-Host $Message }
    }
}

function Show-Banner {
    Write-Host ""
    Write-Host "     _                          " -ForegroundColor Cyan
    Write-Host "    / \   ___ ___  ___ _ __ _ __  " -ForegroundColor Cyan
    Write-Host "   / _ \ / __/ __|/ _ \ '__| '_ \ " -ForegroundColor Cyan
    Write-Host "  / ___ \\__ \__ \  __/ |  | | | |" -ForegroundColor Cyan
    Write-Host " /_/   \_\___/___/\___|_|  |_| |_|" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  MCP Aggregator - Windows Installer"
    Write-Host ""
}

function Test-WindowsBuild {
    $build = [System.Environment]::OSVersion.Version.Build
    if ($build -lt $MinWindowsBuild) {
        Write-ColorOutput "Windows build $build too old. Need $MinWindowsBuild+" -Type "Error"
        exit 1
    }
    Write-ColorOutput "Windows build $build (OK)" -Type "Success"
}

function Test-WSLInstalled {
    $wslPath = Get-Command wsl.exe -ErrorAction SilentlyContinue
    if (-not $wslPath) {
        Write-ColorOutput "WSL not installed. Run: wsl --install" -Type "Error"
        exit 1
    }
    Write-ColorOutput "WSL installed" -Type "Success"
}

function Test-WSLDistribution {
    $distros = wsl --list --quiet 2>&1
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($distros)) {
        Write-ColorOutput "No WSL distribution. Run: wsl --install -d Ubuntu" -Type "Error"
        exit 1
    }
    $firstDistro = ($distros -split "`n" | Where-Object { $_ -and $_.Trim() } | Select-Object -First 1).Trim()
    $firstDistro = $firstDistro -replace '\s*\(Default\)', '' -replace '[^\w-]', ''
    Write-ColorOutput "WSL distribution: $firstDistro" -Type "Success"
    return $firstDistro
}

function Install-Assern {
    param([string]$Distro)
    $installArgs = ""
    if ($Nightly) { $installArgs = " -s -- --nightly" }
    elseif ($Version) { $installArgs = " -s -- -v $Version" }

    $installCommand = "curl -fsSL $InstallScriptUrl | bash$installArgs"
    Write-ColorOutput "Installing Assern inside WSL..." -Type "Info"
    Write-Host "Running: $installCommand" -ForegroundColor DarkGray
    wsl -e bash -c $installCommand
    if ($LASTEXITCODE -ne 0) {
        Write-ColorOutput "Installation failed" -Type "Error"
        exit 1
    }
}

function Test-Installation {
    Write-ColorOutput "Verifying..." -Type "Info"
    $result = wsl -e assern version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host $result
        Write-ColorOutput "Success!" -Type "Success"
    } else {
        Write-ColorOutput "May need WSL restart: wsl --shutdown" -Type "Warning"
    }
}

function Show-NextSteps {
    Write-Host ""
    Write-Host "Next: Open WSL (wsl or ubuntu), then run 'assern --help'"
    Write-Host "Docs: $DocsUrl"
    Write-Host ""
}

function Main {
    Show-Banner
    Write-ColorOutput "Checking prerequisites..." -Type "Info"
    Test-WindowsBuild
    Test-WSLInstalled
    $distro = Test-WSLDistribution
    Install-Assern -Distro $distro
    Test-Installation
    Show-NextSteps
}

Main
