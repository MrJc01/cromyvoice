import os
import sys
import argparse
import time
from pathlib import Path

# Aceite automático de termos do Coqui para automação sem travamento
os.environ["COQUI_TOS_AGREED"] = "1"

# Caminhos padrão dos modelos e clones
HOME = Path.home()
MODELS_DIR = HOME / ".local" / "share" / "myvoice" / "models" / "kokoro"
CLONES_DIR = HOME / ".local" / "share" / "myvoice" / "clones"
CLONES_DIR.mkdir(parents=True, exist_ok=True)

KOKORO_MODEL = MODELS_DIR / "kokoro-v1.0.onnx"
if not KOKORO_MODEL.exists():
    KOKORO_MODEL = MODELS_DIR / "kokoro-v0_19.onnx"

VOICES_BIN = MODELS_DIR / "voices-v1.0.bin"
if not VOICES_BIN.exists():
    VOICES_BIN = MODELS_DIR / "voices.bin"

_kokoro_instance = None
_xtts_instance = None

def get_kokoro():
    global _kokoro_instance
    if _kokoro_instance is None:
        from kokoro_onnx import Kokoro
        if not KOKORO_MODEL.exists() or not VOICES_BIN.exists():
            raise FileNotFoundError(f"Modelos do Kokoro não encontrados em {MODELS_DIR}. Execute install.sh primeiro.")
        _kokoro_instance = Kokoro(str(KOKORO_MODEL), str(VOICES_BIN))
    return _kokoro_instance

def patch_torchaudio():
    import soundfile as sf
    import torch
    import torchaudio

    def _custom_load(filepath, **kwargs):
        data, sr = sf.read(filepath, dtype='float32')
        tensor = torch.from_numpy(data)
        if tensor.ndim == 1:
            tensor = tensor.unsqueeze(0)
        else:
            tensor = tensor.t()
        return tensor, sr

    def _custom_save(filepath, src, sample_rate, **kwargs):
        if isinstance(src, torch.Tensor):
            src = src.detach().cpu().numpy()
        if src.ndim == 2:
            src = src.squeeze()
        sf.write(filepath, src, sample_rate)

    torchaudio.load = _custom_load
    torchaudio.save = _custom_save

def get_xtts():
    global _xtts_instance
    if _xtts_instance is None:
        patch_torchaudio()
        from TTS.api import TTS
        # Usa o modelo XTTS-v2 multilíngue oficial
        _xtts_instance = TTS("tts_models/multilingual/multi-dataset/xtts_v2", gpu=False)
    return _xtts_instance

def list_kokoro_voices():
    # Retorna lista de vozes, priorizando as vozes em Português do Brasil nativas
    return [
        {"id": "pf_dora", "name": "Dora (Feminina - Português Brasileiro)", "lang": "pt-br", "gender": "F"},
        {"id": "pm_alex", "name": "Alex (Masculina - Português Brasileiro)", "lang": "pt-br", "gender": "M"},
        {"id": "pm_santa", "name": "Papai Noel (Masculina Grave - Português)", "lang": "pt-br", "gender": "M"},
        {"id": "af_heart", "name": "Heart (Feminina - Alta Definição / Suave)", "lang": "en-us", "gender": "F"},
        {"id": "af_bella", "name": "Bella (Feminina - Expressiva)", "lang": "en-us", "gender": "F"},
        {"id": "af_sarah", "name": "Sarah (Feminina - Clara / Profissional)", "lang": "en-us", "gender": "F"},
        {"id": "am_adam", "name": "Adam (Masculina - Firme / Narrador)", "lang": "en-us", "gender": "M"},
        {"id": "am_michael", "name": "Michael (Masculina - Conversacional)", "lang": "en-us", "gender": "M"},
        {"id": "bf_isabella", "name": "Isabella (Feminina - Britânica)", "lang": "en-gb", "gender": "F"},
        {"id": "bm_george", "name": "George (Masculina - Britânica)", "lang": "en-gb", "gender": "M"},
        {"id": "ef_dora", "name": "Dora (Feminina - Espanhol)", "lang": "es", "gender": "F"},
        {"id": "em_alex", "name": "Alex (Masculino - Espanhol)", "lang": "es", "gender": "M"}
    ]

def list_clone_voices():
    voices = []
    for ext in ("*.wav", "*.mp3", "*.ogg", "*.flac"):
        for f in CLONES_DIR.glob(ext):
            voices.append({
                "filename": f.name,
                "path": str(f.resolve()),
                "name": f.stem.replace("_", " ").title(),
                "size": f.stat().st_size
            })
    return sorted(voices, key=lambda x: x["name"])

def detect_kokoro_lang(voice: str, user_lang: str = None) -> str:
    if user_lang and user_lang != "auto":
        return user_lang
    if voice.startswith("pf_") or voice.startswith("pm_"):
        return "pt-br"
    if voice.startswith("ef_") or voice.startswith("em_"):
        return "es"
    if voice.startswith("bf_") or voice.startswith("bm_"):
        return "en-gb"
    if voice.startswith("ff_") or voice.startswith("fm_"):
        return "fr-fr"
    if voice.startswith("if_") or voice.startswith("im_"):
        return "it"
    return "en-us"

def synthesize_kokoro(text: str, voice: str = "pf_dora", speed: float = 1.0, lang: str = None, out_path: str = None):
    import soundfile as sf
    kokoro = get_kokoro()
    final_lang = detect_kokoro_lang(voice, lang)
    samples, sample_rate = kokoro.create(text, voice=voice, speed=speed, lang=final_lang)
    if out_path:
        sf.write(out_path, samples, sample_rate)
    return samples, sample_rate

def synthesize_xtts(text: str, speaker_wav: str, language: str = "pt", out_path: str = None):
    import soundfile as sf
    if not os.path.exists(speaker_wav):
        raise FileNotFoundError(f"Arquivo de referência de voz não encontrado: {speaker_wav}")
    xtts = get_xtts()
    if out_path is None:
        out_path = "/tmp/xtts_out.wav"
    xtts.tts_to_file(text=text, speaker_wav=speaker_wav, language=language, file_path=out_path, temperature=0.35, repetition_penalty=2.5, length_penalty=1.0)
    return out_path

# ==================== MODO SERVIDOR (FASTAPI) ====================
def start_server(host="127.0.0.1", port=8001):
    import uvicorn
    from fastapi import FastAPI, HTTPException, UploadFile, File, Form
    from fastapi.responses import FileResponse, JSONResponse
    from fastapi.middleware.cors import CORSMiddleware
    from pydantic import BaseModel
    import shutil

    app = FastAPI(title="CromyVoice AI Engine", version="1.0")

    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    class KokoroReq(BaseModel):
        text: str
        voice: str = "af_bella"
        speed: float = 1.0
        out_path: str = ""

    class XTTSReq(BaseModel):
        text: str
        speaker_wav: str
        language: str = "pt"
        out_path: str = ""

    @app.get("/health")
    def health():
        return {"status": "ok", "kokoro_ready": KOKORO_MODEL.exists(), "clones_dir": str(CLONES_DIR)}

    @app.get("/voices/kokoro")
    def get_voices():
        return list_kokoro_voices()

    @app.get("/clone/voices")
    def get_clones():
        return list_clone_voices()

    @app.post("/clone/upload")
    async def upload_clone(file: UploadFile = File(...)):
        dest = CLONES_DIR / file.filename
        with open(dest, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)
        return {"success": True, "filename": file.filename, "path": str(dest.resolve())}

    @app.post("/tts/kokoro")
    def api_kokoro(req: KokoroReq):
        try:
            out_file = req.out_path
            if not out_file:
                out_file = f"/tmp/kokoro_{int(time.time()*1000)}.wav"
            synthesize_kokoro(req.text, req.voice, req.speed, out_path=out_file)
            return {"success": True, "out_path": out_file}
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    @app.post("/tts/clone")
    def api_clone(req: XTTSReq):
        try:
            out_file = req.out_path
            if not out_file:
                out_file = f"/tmp/xtts_{int(time.time()*1000)}.wav"
            synthesize_xtts(req.text, req.speaker_wav, req.language, out_file)
            return {"success": True, "out_path": out_file}
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    uvicorn.run(app, host=host, port=port)

# ==================== MODO CLI ====================
def main():
    parser = argparse.ArgumentParser(description="CromyVoice AI Engine CLI & Server")
    parser.add_argument("--server", action="store_true", help="Inicia o servidor HTTP FastAPI")
    parser.add_argument("--host", default="127.0.0.1", help="Host do servidor")
    parser.add_argument("--port", type=int, default=8001, help="Porta do servidor")

    parser.add_argument("--engine", choices=["kokoro", "xtts"], help="Motor de síntese")
    parser.add_argument("--text", help="Texto a ser sintetizado")
    parser.add_argument("--voice", default="pf_dora", help="Voz para Kokoro (padrão: pf_dora)")
    parser.add_argument("--speaker-wav", help="Caminho do arquivo de áudio de referência para XTTS")
    parser.add_argument("--lang", default=None, help="Idioma para síntese (ex: pt-br, pt, en-us)")
    parser.add_argument("--speed", type=float, default=1.0, help="Velocidade de reprodução")
    parser.add_argument("--out", help="Arquivo WAV de saída")
    parser.add_argument("--list-voices", choices=["kokoro", "clones"], help="Lista vozes disponíveis")

    args = parser.parse_args()

    if args.server:
        start_server(args.host, args.port)
        return

    if args.list_voices == "kokoro":
        import json
        print(json.dumps(list_kokoro_voices(), indent=2, ensure_ascii=False))
        return

    if args.list_voices == "clones":
        import json
        print(json.dumps(list_clone_voices(), indent=2, ensure_ascii=False))
        return

    if not args.engine or not args.text or not args.out:
        parser.print_help()
        sys.exit(1)

    if args.engine == "kokoro":
        print(f"🎙️ [Kokoro v1.0] Sintetizando com voz '{args.voice}'...")
        synthesize_kokoro(args.text, args.voice, args.speed, lang=args.lang, out_path=args.out)
        print(f"✅ Áudio salvo em: {args.out}")

    elif args.engine == "xtts":
        if not args.speaker_wav:
            print("❌ Erro: --speaker-wav é obrigatório para clonagem de voz.")
            sys.exit(1)
        print(f"🧬 [XTTS-v2] Clonando voz de '{args.speaker_wav}' em idioma '{args.lang}'...")
        synthesize_xtts(args.text, args.speaker_wav, args.lang, args.out)
        print(f"✅ Áudio clonado salvo em: {args.out}")

if __name__ == "__main__":
    main()
