#!/usr/bin/env bash
set -e

### VARIABLES
VERSION="v0.1.0"   # luego lo automatizamos
INSTALL_DIR="/opt/tenubah-agent"
CONFIG_DIR="/etc/tenubah-agent"
BIN="$INSTALL_DIR/tenubah-agent"

if [ -z "$TENUBAH_TOKEN" ]; then
  echo "❌ TENUBAH_TOKEN no definido"
  exit 1
fi

echo "📁 Creando directorios..."
sudo mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"

echo "⬇️ Descargando Tenubah Agent $VERSION..."
sudo curl -sSL \
  https://github.com/TenubahDEV/tenubah-agent/releases/download/$VERSION/tenubah-agent-linux-amd64 \
  -o "$BIN"

sudo chmod +x "$BIN"

echo "📝 Creando configuración..."
sudo tee "$CONFIG_DIR/config.yaml" > /dev/null <<EOF
job_name: "tenubah_agent"
instance_name: ""
pushgateway_url: "https://push.tenubah.com"
token: "$TENUBAH_TOKEN"
interval_seconds: 60
labels:
  customer: "cliente1"
  env: "prod"
EOF

echo "🛠️ Instalando servicio..."
sudo "$BIN" -config "$CONFIG_DIR/config.yaml" install
sudo "$BIN" start

echo "✅ Tenubah Agent instalado y corriendo"
