$ErrorActionPreference = "Stop"

if (-not $env:TENUBAH_TOKEN) {
    Write-Host "❌ TENUBAH_TOKEN no definido"
    exit 1
}

$Version = "v0.1.0"
$InstallDir = "C:\Program Files\TenubahAgent"
$ConfigDir  = "C:\ProgramData\TenubahAgent"
$BinPath    = Join-Path $InstallDir "tenubah-agent.exe"

Write-Host "📁 Creando directorios..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigDir  | Out-Null

Write-Host "⬇️ Descargando Tenubah Agent $Version..."
Invoke-WebRequest `
  https://github.com/TenubahDEV/tenubah-agent/releases/download/$Version/tenubah-agent-windows-amd64.exe `
  -OutFile $BinPath

@"
job_name: "tenubah_agent"
instance_name: ""
pushgateway_url: "https://push.tenubah.com"
token: "$env:TENUBAH_TOKEN"
interval_seconds: 60
labels:
  customer: "cliente1"
  env: "prod"
"@ | Out-File (Join-Path $ConfigDir "config.yaml") -Encoding ascii

Write-Host "🛠️ Instalando servicio..."
& $BinPath -config (Join-Path $ConfigDir "config.yaml") install
& $BinPath start

Write-Host "✅ Tenubah Agent instalado y corriendo"
