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

if (-not $env:TENUBAH_TOKEN) {
    Write-Host "❌ TENUBAH_TOKEN no definido"
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
& $BinPath -config (Join-Path $ConfigDir "config.yaml") install

Write-Host "▶️ Iniciando servicio..."
& $BinPath start

########################################
# FINAL
########################################

Write-Host "✅ Tenubah Agent instalado y corriendo"
