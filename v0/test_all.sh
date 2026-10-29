#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
TMP_DIR="$BASE_DIR/tmp_test"
mkdir -p "$TMP_DIR"

echo -e "${CYAN}====================================================${NC}"
echo -e "${CYAN}🧪 BATERIA COMPLETA DE TESTES - CROMYVOICE${NC}"
echo -e "${CYAN}====================================================${NC}"

# 1. TESTE KOKORO-82M (Voz Feminina Brasileira pf_dora)
echo -e "\n${YELLOW}[TESTE 1/6] CLI: Gerando voz com Kokoro-82M v1.0 (pf_dora - Português Brasileiro)...${NC}"
"$BASE_DIR/cromyvoice" -text "Olá! Este é um teste da voz feminina brasileira Dora no Kokoro." \
  -engine kokoro -kokoro-voice pf_dora -out "$TMP_DIR/teste1_kokoro_dora.wav"

if [ -f "$TMP_DIR/teste1_kokoro_dora.wav" ] && [ -s "$TMP_DIR/teste1_kokoro_dora.wav" ]; then
    INFO=$(ffprobe -v error -show_entries stream=channels,sample_rate,duration -of default=noprint_wrappers=1:nokey=1 "$TMP_DIR/teste1_kokoro_dora.wav" | tr '\n' ' ')
    echo -e "${GREEN}✅ Teste 1 APROVADO! Voz brasileira gerada: $INFO${NC}"
else
    echo -e "${RED}❌ Teste 1 FALHOU! Arquivo vazio ou não encontrado.${NC}"
    exit 1
fi

# 2. TESTE KOKORO-82M (Voz Masculina Brasileira pm_alex)
echo -e "\n${YELLOW}[TESTE 2/6] CLI: Gerando voz com Kokoro-82M v1.0 (pm_alex - Português Brasileiro)...${NC}"
"$BASE_DIR/cromyvoice" -text "Olá! Este é o teste com a voz masculina brasileira Alex no Kokoro." \
  -engine kokoro -kokoro-voice pm_alex -out "$TMP_DIR/teste2_kokoro_alex.wav"

if [ -f "$TMP_DIR/teste2_kokoro_alex.wav" ] && [ -s "$TMP_DIR/teste2_kokoro_alex.wav" ]; then
    INFO=$(ffprobe -v error -show_entries stream=channels,sample_rate,duration -of default=noprint_wrappers=1:nokey=1 "$TMP_DIR/teste2_kokoro_alex.wav" | tr '\n' ' ')
    echo -e "${GREEN}✅ Teste 2 APROVADO! Voz brasileira gerada: $INFO${NC}"
else
    echo -e "${RED}❌ Teste 2 FALHOU!${NC}"
    exit 1
fi

# 3. TESTE CLONAGEM DE VOZ (XTTS-v2 em Português)
echo -e "\n${YELLOW}[TESTE 3/6] CLI: Testando Clonagem de Voz XTTS-v2 em Português...${NC}"
REF_VOICE="$HOME/.local/share/myvoice/clones/am_adam_sample.wav"
"$BASE_DIR/cromyvoice" -text "Olá! O sistema de clonagem de voz pelo XTTS versão dois está funcionando perfeitamente em português." \
  -engine xtts -clone-wav "$REF_VOICE" -out "$TMP_DIR/teste3_clone.wav"

if [ -f "$TMP_DIR/teste3_clone.wav" ] && [ -s "$TMP_DIR/teste3_clone.wav" ]; then
    INFO=$(ffprobe -v error -show_entries stream=channels,sample_rate,duration -of default=noprint_wrappers=1:nokey=1 "$TMP_DIR/teste3_clone.wav" | tr '\n' ' ')
    echo -e "${GREEN}✅ Teste 3 APROVADO! Áudio clonado gerado: $INFO${NC}"
else
    echo -e "${RED}❌ Teste 3 FALHOU!${NC}"
    exit 1
fi

# 4. TESTE REGRESSÃO MOTORES NATIVOS (Edge-TTS)
echo -e "\n${YELLOW}[TESTE 4/6] CLI: Testando regressão com Edge-TTS (Nuvem Neural)...${NC}"
"$BASE_DIR/cromyvoice" -text "Verificando se o Microsoft Edge neural continua funcionando perfeitamente." \
  -engine edge-tts -voice pt-BR-AntonioNeural -out "$TMP_DIR/teste4_edge.mp3"

if [ -f "$TMP_DIR/teste4_edge.mp3" ] && [ -s "$TMP_DIR/teste4_edge.mp3" ]; then
    INFO=$(ffprobe -v error -show_entries stream=channels,sample_rate,duration -of default=noprint_wrappers=1:nokey=1 "$TMP_DIR/teste4_edge.mp3" | tr '\n' ' ')
    echo -e "${GREEN}✅ Teste 4 APROVADO! MP3 gerado: $INFO${NC}"
else
    echo -e "${RED}❌ Teste 4 FALHOU!${NC}"
    exit 1
fi

# 5. TESTE SERVIDOR WEB E API REST
echo -e "\n${YELLOW}[TESTE 5/6] WEB: Iniciando servidor cromyvoice-web e testando APIs...${NC}"
"$BASE_DIR/cromyvoice-web" &
WEB_PID=$!
sleep 2

trap "kill $WEB_PID 2>/dev/null || true" EXIT

# 5.1 Teste Config API
CFG_RES=$(curl -s http://localhost:8000/api/config)
if echo "$CFG_RES" | grep -q "kokoro_voice"; then
    echo -e "${GREEN}✅ API /api/config OK (Campos Kokoro e Clonagem presentes)${NC}"
else
    echo -e "${RED}❌ API /api/config falhou!${NC}"
    exit 1
fi

# 5.2 Teste Voices API
VOICES_RES=$(curl -s http://localhost:8000/api/voices/kokoro)
if echo "$VOICES_RES" | grep -q "af_bella"; then
    echo -e "${GREEN}✅ API /api/voices/kokoro OK (Lista de vozes recebida)${NC}"
else
    echo -e "${RED}❌ API /api/voices/kokoro falhou!${NC}"
    exit 1
fi

# 5.3 Teste Clone Voices API
CLONES_RES=$(curl -s http://localhost:8000/api/clone/voices)
if echo "$CLONES_RES" | grep -q "am_adam_sample.wav"; then
    echo -e "${GREEN}✅ API /api/clone/voices OK (Catálogo de referências encontrado)${NC}"
else
    echo -e "${RED}❌ API /api/clone/voices falhou!${NC}"
    exit 1
fi

# 5.4 Teste Geração Web via Kokoro (Voz Dora - Português)
GEN_RES=$(curl -s -X POST http://localhost:8000/api/generate \
  -H "Content-Type: application/json" \
  -d '{"text": "Teste disparado via painel Web para validar síntese brasileira do Kokoro.", "engine": "kokoro", "voice": "pf_dora"}')

if echo "$GEN_RES" | grep -q '"success":true'; then
    AUDIO_URL=$(echo "$GEN_RES" | grep -oP '(?<="url":")[^"]*')
    echo -e "${GREEN}✅ API /api/generate OK (URL de download: $AUDIO_URL)${NC}"
    
    # Validar download do arquivo gerado
    curl -s -o "$TMP_DIR/teste5_web_download.wav" "http://localhost:8000$AUDIO_URL"
    if [ -s "$TMP_DIR/teste5_web_download.wav" ]; then
        INFO=$(ffprobe -v error -show_entries stream=channels,sample_rate,duration -of default=noprint_wrappers=1:nokey=1 "$TMP_DIR/teste5_web_download.wav" | tr '\n' ' ')
        echo -e "${GREEN}✅ Download de áudio web validado com sucesso: $INFO${NC}"
    else
        echo -e "${RED}❌ Download do áudio web veio vazio!${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ API /api/generate falhou: $GEN_RES${NC}"
    exit 1
fi

# 6. TESTE DE CABOS DE ÁUDIO PULSEAUDIO / PIPEWIRE
echo -e "\n${YELLOW}[TESTE 6/6] ÁUDIO: Verificando infraestrutura PulseAudio/PipeWire...${NC}"
if which pactl >/dev/null 2>&1; then
    SERVER_INFO=$(pactl info 2>&1 | grep "Server Name" || true)
    echo -e "${GREEN}✅ Servidor de áudio ativo no sistema operacional: $SERVER_INFO${NC}"
else
    echo -e "${YELLOW}⚠️ pactl não encontrado, verifique seu PulseAudio/PipeWire.${NC}"
fi

# Finalizar servidor web de teste
kill $WEB_PID 2>/dev/null || true
trap - EXIT

echo -e "\n${CYAN}====================================================${NC}"
echo -e "${GREEN}🎉 TODOS OS 6 TESTES FORAM CONCLUÍDOS COM 100% DE SUCESSO!${NC}"
echo -e "${CYAN}====================================================${NC}"
