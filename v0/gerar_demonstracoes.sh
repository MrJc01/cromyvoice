#!/bin/bash
set -e

TEXT="No fim, o único antídoto real para o risco de governança e o colapso da manutenção é a soberania técnica. Você precisa estar preparado para assumir o controle. Isso significa ter a capacidade interna de entender, auditar e, se necessário, manter o código que você usa. A dependência passiva é uma sentença de morte lenta."
OUT_DIR="/home/j/Documentos/GitHub/cromyvoice/v0/audios_demonstracao"
mkdir -p "$OUT_DIR"

echo "🎙️ 1. Gerando com Kokoro v1.0 - Dora (Feminina Brasil)..."
./cromyvoice -text "$TEXT" -engine kokoro -kokoro-voice pf_dora -out "$OUT_DIR/1_kokoro_dora_brasil.wav"

echo "🎙️ 2. Gerando com Kokoro v1.0 - Alex (Masculino Brasil)..."
./cromyvoice -text "$TEXT" -engine kokoro -kokoro-voice pm_alex -out "$OUT_DIR/2_kokoro_alex_brasil.wav"

echo "🎙️ 3. Gerando com Edge-TTS - Antonio (Masculino Brasil)..."
./cromyvoice -text "$TEXT" -engine edge-tts -voice pt-BR-AntonioNeural -out "$OUT_DIR/3_edgetts_antonio_brasil.mp3"

echo "🎙️ 4. Gerando com Edge-TTS - Francisca (Feminina Brasil)..."
./cromyvoice -text "$TEXT" -engine edge-tts -voice pt-BR-FranciscaNeural -out "$OUT_DIR/4_edgetts_francisca_brasil.mp3"

echo "🎙️ 5. Gerando com XTTS-v2 - Clonagem em Português..."
./cromyvoice -text "$TEXT" -engine xtts -clone-wav "$HOME/.local/share/myvoice/clones/am_adam_sample.wav" -out "$OUT_DIR/5_xtts_clonagem_brasil.wav"

echo "✅ Todos os áudios foram gerados com sucesso!"
ls -lh "$OUT_DIR"
