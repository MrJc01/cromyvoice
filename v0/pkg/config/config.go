package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	TTSEngine     string `json:"tts_engine"`
	PiperModel    string `json:"piper_model"`
	PiperBinary   string `json:"piper_binary"`
	EdgeVoice     string `json:"edge_voice"`
	SinkName      string `json:"sink_name"`
	MicName       string `json:"mic_name"`
	PlayOnSpeaker bool   `json:"play_on_speaker"`
	Macros        map[string]string `json:"macros"`
	DevMode       bool `json:"-"`

	// Kokoro-82M (TTS Neural ONNX de alta fidelidade)
	KokoroVoice   string `json:"kokoro_voice"`

	// XTTS-v2 (Clonagem de Voz)
	CloneVoicePath string `json:"clone_voice_path"`
	CloneLanguage  string `json:"clone_language"`

	// AI Engine (Micro-serviço Python local)
	AIEngineURL   string `json:"ai_engine_url"`
	AIEnginePython string `json:"ai_engine_python"`

	savePath string `json:"-"`
}

func LoadConfig(appDir string) *AppConfig {
	cfg := &AppConfig{
		TTSEngine:      "piper", 
		PiperModel:     filepath.Join(appDir, "pt_BR-faber-medium.onnx"),
		PiperBinary:    filepath.Join(appDir, "piper", "piper"),
		EdgeVoice:      "pt-BR-AntonioNeural",
		SinkName:       "MyVoiceSink",
		MicName:        "Microfone_Bot",
		PlayOnSpeaker:  false,
		Macros:         make(map[string]string),
		KokoroVoice:    "pf_dora",
		CloneVoicePath: "",
		CloneLanguage:  "pt",
		AIEngineURL:    "http://127.0.0.1:8001",
		AIEnginePython: filepath.Join(appDir, "ai_env", "bin", "python3"),
	}

	cfg.savePath = filepath.Join(appDir, "config.json")

	data, err := os.ReadFile(cfg.savePath)
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// Hot-fix auto-cura caso venha do config da V2 legado:
	if cfg.PiperBinary == "" || cfg.PiperBinary == "piper" {
		cfg.PiperBinary = filepath.Join(appDir, "piper", "piper")
	}
	if cfg.PiperModel == "pt_BR-faber-medium.onnx" {
		cfg.PiperModel = filepath.Join(appDir, "pt_BR-faber-medium.onnx")
	}

	return cfg
}

func (c *AppConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.savePath, data, 0644)
}
