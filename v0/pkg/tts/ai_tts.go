package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/j/myvoice/pkg/config"
)

// AIEngineRequest representa uma requisição para o AI Engine Python
type AIEngineRequest struct {
	Text       string  `json:"text"`
	Voice      string  `json:"voice,omitempty"`
	Speed      float64 `json:"speed,omitempty"`
	SpeakerWav string  `json:"speaker_wav,omitempty"`
	Language   string  `json:"language,omitempty"`
	OutPath    string  `json:"out_path"`
}

type AIEngineResponse struct {
	Success bool   `json:"success"`
	OutPath string `json:"out_path"`
}

// getServiceScript retorna o caminho do service.py relativo ao binário/projeto
func getServiceScript() string {
	// Tenta localizar relativo ao executável
	execPath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(execPath), "ai_engine", "service.py")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Fallback: relativo ao diretório de trabalho atual
	wd, _ := os.Getwd()
	candidate := filepath.Join(wd, "ai_engine", "service.py")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// Último fallback: caminho absoluto conhecido
	return filepath.Join(os.Getenv("HOME"), "Documentos", "GitHub", "cromyvoice", "v0", "ai_engine", "service.py")
}

// SynthesizeViaCLI executa o service.py diretamente via CLI (sem servidor)
func SynthesizeViaCLI(text string, engine string, outFile string, cfg *config.AppConfig) error {
	pythonBin := cfg.AIEnginePython
	serviceScript := getServiceScript()

	args := []string{serviceScript, "--engine", engine, "--text", text, "--out", outFile}

	switch engine {
	case "kokoro":
		args = append(args, "--voice", cfg.KokoroVoice)
	case "xtts":
		if cfg.CloneVoicePath == "" {
			return fmt.Errorf("nenhuma voz de referência configurada para clonagem (clone_voice_path vazio)")
		}
		args = append(args, "--speaker-wav", cfg.CloneVoicePath, "--lang", cfg.CloneLanguage)
	default:
		return fmt.Errorf("motor de IA desconhecido: %s", engine)
	}

	if cfg.DevMode {
		fmt.Printf("\r\n\033[36m[DEV AI-ENGINE CLI]\033[0m %s %v\r\n", pythonBin, args)
	}

	cmd := exec.Command(pythonBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ai_engine CLI falhou: %v (output: %s)", err, string(output))
	}

	if cfg.DevMode {
		fmt.Printf("\r\n\033[36m[DEV AI-ENGINE CLI]\033[0m Saída: %s\r\n", string(output))
	}

	return nil
}

// SynthesizeViaHTTP envia requisição para o servidor FastAPI do AI Engine
func SynthesizeViaHTTP(text string, engine string, outFile string, cfg *config.AppConfig) error {
	var endpoint string
	req := AIEngineRequest{
		Text:    text,
		OutPath: outFile,
	}

	switch engine {
	case "kokoro":
		endpoint = cfg.AIEngineURL + "/tts/kokoro"
		req.Voice = cfg.KokoroVoice
		req.Speed = 1.0
	case "xtts":
		endpoint = cfg.AIEngineURL + "/tts/clone"
		req.SpeakerWav = cfg.CloneVoicePath
		req.Language = cfg.CloneLanguage
	default:
		return fmt.Errorf("motor de IA desconhecido: %s", engine)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if cfg.DevMode {
		fmt.Printf("\r\n\033[36m[DEV AI-ENGINE HTTP]\033[0m POST %s body=%s\r\n", endpoint, string(body))
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ai_engine HTTP falhou (servidor offline?): %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("ai_engine HTTP retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	var result AIEngineResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("resposta inválida do ai_engine: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("ai_engine reportou falha")
	}

	if cfg.DevMode {
		fmt.Printf("\r\n\033[36m[DEV AI-ENGINE HTTP]\033[0m Áudio salvo em: %s\r\n", result.OutPath)
	}

	return nil
}

// IsAIEngineServerRunning verifica se o servidor FastAPI está rodando
func IsAIEngineServerRunning(cfg *config.AppConfig) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(cfg.AIEngineURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// SynthesizeAI tenta usar o servidor HTTP se disponível, senão faz fallback para CLI
func SynthesizeAI(text string, engine string, outFile string, cfg *config.AppConfig) error {
	if IsAIEngineServerRunning(cfg) {
		return SynthesizeViaHTTP(text, engine, outFile, cfg)
	}
	return SynthesizeViaCLI(text, engine, outFile, cfg)
}
