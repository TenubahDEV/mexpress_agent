#!/usr/bin/env bash
set -euo pipefail

########################################
# CONFIGURACIÓN GENERAL
########################################

CLIENT="dev"
BASE_URL="https://install.tenubah.com/${CLIENT}"

INSTALL_DIR="/opt/tenubah-agent"
CONFIG_DIR="/etc/tenubah-agent"
BIN="${INSTALL_DIR}/tenubah-agent"
SIG="${BIN}.sig"

########################################
# VALIDACIONES
########################################

if [ -z "${TENUBAH_TOKEN:-}" ]; then
  echo "❌ TENUBAH_TOKEN no definido"
  exit 1
fi

########################################
# INSTALACIÓN
########################################

echo "📁 Creando directorios..."
sudo mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"

echo "⬇️ Descargando Tenubah Agent (latest aprobado)..."

sudo curl -fL \
  "${BASE_URL}/bin/tenubah-agent-linux-amd64" \
  -o "$BIN"

sudo curl -fL \
  "${BASE_URL}/bin/tenubah-agent-linux-amd64.sig" \
  -o "$SIG"

sudo chmod +x "$BIN"

########################################
# (Opcional pero recomendado)
# Validar que el binario sea ELF
########################################

if ! file "$BIN" | grep -q "ELF"; then
  echo "❌ El archivo descargado no es un binario ELF válido"
  exit 1
fi

########################################
# CONFIGURACIÓN
########################################

echo "📝 Creando configuración..."

sudo tee "$CONFIG_DIR/config.yaml" > /dev/null <<EOF
job_name: "tenubah_agent"
instance_name: ""
pushgateway_url: "https://push.tenubah.com"
token: "$TENUBAH_TOKEN"
interval_seconds: 60

auto_update:
  enabled: true
  check_interval_hours: 24

labels:
  customer: "${CLIENT}"
  env: "prod"
EOF

########################################
# SERVICIO
########################################

echo "🛠️ Instalando servicio..."
sudo "$BIN" -config "$CONFIG_DIR/config.yaml" install

echo "▶️ Iniciando servicio..."
sudo "$BIN" start

########################################
# FINAL
########################################

echo "✅ Tenubah Agent instalado y corriendo"
