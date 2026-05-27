$ErrorActionPreference = "Stop"

########################################
# CONFIGURACIÓN GENERAL
########################################

$Client     = "mexpress"
$BaseUrl    = "https://install.tenubah.com/$Client"

$InstallDir = "C:\Program Files\TenubahAgent"
$ConfigDir  = "C:\ProgramData\TenubahAgent"
$BinPath    = Join-Path $InstallDir "tenubah-agent.exe"
$SigPath    = "$BinPath.sig"

########################################
# VALIDACIONES
########################################

if (-not $env:TENUBAH_TOKEN -and (-not $env:TENUBAH_USER -or -not $env:TENUBAH_PASSWORD)) {
    Write-Host "❌ Debes definir TENUBAH_TOKEN o bien ambos TENUBAH_USER y TENUBAH_PASSWORD"
    exit 1
}

########################################
# INSTALACIÓN
########################################

Write-Host "📁 Creando directorios..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigDir  | Out-Null

Write-Host "⬇️ Descargando Tenubah Agent (latest aprobado)..."

Invoke-WebRequest `
  "$BaseUrl/bin/tenubah-agent-windows-amd64.exe" `
  -OutFile $BinPath `
  -UseBasicParsing

Invoke-WebRequest `
  "$BaseUrl/bin/tenubah-agent-windows-amd64.exe.sig" `
  -OutFile $SigPath `
  -UseBasicParsing

########################################
# VALIDACIÓN BÁSICA DEL BINARIO
########################################

$FileInfo = Get-Item $BinPath
if ($FileInfo.Length -lt 1MB) {
    Write-Host "❌ El binario descargado no es válido (tamaño incorrecto)"
    exit 1
}

########################################
# CONFIGURACIÓN
########################################

Write-Host "📝 Creando configuración..."

@"
job_name: "mexpress_agent"
instance_name: ""
pushgateway_url: "https://push.mexpress.tenubah.com"
token: "$($env:TENUBAH_TOKEN)"
username: "$($env:TENUBAH_USER)"
password: "$($env:TENUBAH_PASSWORD)"
interval_seconds: 60

auto_update:
  enabled: true
  check_interval_hours: 24

labels:
  customer: "$Client"
  env: "prod"
"@ | Out-File (Join-Path $ConfigDir "config.yaml") -Encoding ascii -Force

########################################
# SERVICIO
########################################

Write-Host "🛠️ Instalando servicio..."
& $BinPath -config (Join-Path $ConfigDir "config.yaml") install 2>&1

Write-Host "▶️ Iniciando servicio..."
& $BinPath start 2>&1

########################################
# FINAL
########################################

Write-Host "✅ Tenubah Agent instalado y corriendo"
