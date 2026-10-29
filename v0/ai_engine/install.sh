#!/bin/bash
set -e

ENV_DIR="$HOME/.local/share/myvoice/ai_env"
MODELS_DIR="$HOME/.local/share/myvoice/models/kokoro"
CLONES_DIR="$HOME/.local/share/myvoice/clones"

mkdir -p "$MODELS_DIR" "$CLONES_DIR"

echo "=========================================="
echo "🚀 Configurando Ambiente de IA do CromyVoice"
echo "=========================================="

# 1. Criar virtualenv caso não exista
if [ ! -d "$ENV_DIR" ]; then
    echo "📦 Criando ambiente virtual em $ENV_DIR..."
    python3 -m virtualenv "$ENV_DIR"
else
    echo "✅ Ambiente virtual já existe em $ENV_DIR"
fi

# 2. Instalar dependências essenciais
echo "📦 Instalando pacotes Python (kokoro-onnx, soundfile, fastapi, uvicorn, coqui-tts)..."
"$ENV_DIR/bin/pip" install --no-cache-dir kokoro-onnx soundfile fastapi uvicorn coqui-tts

# 3. Baixar modelo Kokoro-82M ONNX e vozes
KOKORO_MODEL="$MODELS_DIR/kokoro-v0_19.onnx"
VOICES_BIN="$MODELS_DIR/voices.bin"

if [ ! -f "$KOKORO_MODEL" ]; then
    echo "⬇️ Baixando modelo neural Kokoro-82M ONNX (~310MB)..."
    curl -L --progress-bar "https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files/kokoro-v0_19.onnx" -o "$KOKORO_MODEL"
else
    echo "✅ Modelo Kokoro-82M já baixado em $KOKORO_MODEL"
fi

if [ ! -f "$VOICES_BIN" ]; then
    echo "⬇️ Baixando catálogo de vozes do Kokoro (~20MB)..."
    curl -L --progress-bar "https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files/voices.bin" -o "$VOICES_BIN"
else
    echo "✅ Catálogo de vozes já baixado em $VOICES_BIN"
fi

echo "=========================================="
echo "🎉 Ambiente de IA configurado com sucesso!"
echo "=========================================="
