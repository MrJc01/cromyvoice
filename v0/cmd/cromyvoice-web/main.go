package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/j/myvoice/pkg/bootstrap"
	"github.com/j/myvoice/pkg/config"
)

var (
	cfg       *config.AppConfig
	staticDir string
	audioDir  string
)

type GenerateRequest struct {
	Text   string `json:"text"`
	Engine string `json:"engine"`
	Voice  string `json:"voice"`
	Model  string `json:"model"`
	Rate   string `json:"rate"`  // Ex: "+10%" ou "-5%"
	Pitch  string `json:"pitch"` // Ex: "+2Hz" ou "-5Hz"
}

type GenerateResponse struct {
	Success  bool   `json:"success"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Engine   string `json:"engine"`
	Text     string `json:"text"`
	Date     string `json:"date"`
}

type AudioFile struct {
	Filename  string    `json:"filename"`
	URL       string    `json:"url"`
	Text      string    `json:"text"`
	Engine    string    `json:"engine"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	// 1. Inicializar ambiente do MyVoice (garantir piper e modelos)
	appDir, err := bootstrap.EnsureEnvironment()
	if err != nil {
		fmt.Printf("⚠️ Erro ao configurar ambiente local: %v\n", err)
	}

	// 2. Carregar configuração
	cfg = config.LoadConfig(appDir)

	// 3. Configurar caminhos estáticos
	execPath, err := os.Executable()
	var baseDir string
	if err != nil {
		baseDir = "."
	} else {
		baseDir = filepath.Dir(execPath)
		// Se estiver rodando com "go run", usa o diretório de trabalho atual
		if strings.Contains(baseDir, "/tmp/go-build") {
			wd, err := os.Getwd()
			if err == nil {
				baseDir = wd
			}
		}
	}

	staticDir = filepath.Join(baseDir, "frontend", "dist")
	audioDir = filepath.Join(baseDir, "static", "audios")

	// Criar diretório de áudios se não existir
	os.MkdirAll(audioDir, 0755)

	// 4. Configurar rotas
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/generate", handleGenerate)
	http.HandleFunc("/api/history", handleHistory)
	http.HandleFunc("/api/audio/", serveAudio)
	http.HandleFunc("/api/models", handleModels)
	http.HandleFunc("/api/models/download", handleModelDownload)
	http.HandleFunc("/api/voices/kokoro", handleKokoroVoices)
	http.HandleFunc("/api/clone/voices", handleCloneVoices)
	http.HandleFunc("/api/clone/upload", handleCloneUpload)

	// Servir o frontend React compilado
	fs := http.FileServer(http.Dir(staticDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Se a rota for uma rota de API, ela já foi capturada.
		// Caso contrário, se for um arquivo que não existe, serve o index.html (SPA routing fallback)
		path := filepath.Join(staticDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) && !strings.HasPrefix(r.URL.Path, "/api") {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	// 5. Iniciar o servidor
	port := "8000"
	fmt.Printf("\n🚀 Servidor CromyVoice Web rodando em http://localhost:%s\n", port)
	fmt.Printf("📂 Servindo Frontend de: %s\n", staticDir)
	fmt.Printf("📂 Salvando Áudios em: %s\n", audioDir)
	
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor: %v\n", err)
	}
}

// Handler para obter ou atualizar a configuração
func handleConfig(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
		return
	}

	if r.Method == http.MethodPost {
		var newCfg config.AppConfig
		err := json.NewDecoder(r.Body).Decode(&newCfg)
		if err != nil {
			http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Atualizar campos relevantes da configuração atual
		cfg.TTSEngine = newCfg.TTSEngine
		if newCfg.PiperModel != "" {
			cfg.PiperModel = newCfg.PiperModel
		}
		if newCfg.PiperBinary != "" {
			cfg.PiperBinary = newCfg.PiperBinary
		}
		if newCfg.EdgeVoice != "" {
			cfg.EdgeVoice = newCfg.EdgeVoice
		}
		if newCfg.KokoroVoice != "" {
			cfg.KokoroVoice = newCfg.KokoroVoice
		}
		if newCfg.CloneVoicePath != "" {
			cfg.CloneVoicePath = newCfg.CloneVoicePath
		}
		if newCfg.CloneLanguage != "" {
			cfg.CloneLanguage = newCfg.CloneLanguage
		}
		if newCfg.SinkName != "" {
			cfg.SinkName = newCfg.SinkName
		}
		if newCfg.MicName != "" {
			cfg.MicName = newCfg.MicName
		}
		cfg.PlayOnSpeaker = newCfg.PlayOnSpeaker
		if newCfg.Macros != nil {
			cfg.Macros = newCfg.Macros
		}

		err = cfg.Save()
		if err != nil {
			http.Error(w, "Erro ao salvar configuração: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
		return
	}

	http.Error(w, "Método não suportado", http.StatusMethodNotAllowed)
}

// Handler para gerar áudio
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não suportado", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, "O texto não pode estar vazio", http.StatusBadRequest)
		return
	}

	// Definir motor, voz e modelo padrão se omitidos
	engine := req.Engine
	if engine == "" {
		engine = cfg.TTSEngine
	}
	voice := req.Voice
	if voice == "" {
		if engine == "kokoro" {
			voice = cfg.KokoroVoice
			if voice == "" {
				voice = "pf_dora"
			}
		} else {
			voice = cfg.EdgeVoice
		}
	}
	model := req.Model
	if model == "" {
		model = cfg.PiperModel
	}

	// Criar nome único para o arquivo
	timestamp := time.Now().UnixNano()
	var ext string
	if engine == "piper" || engine == "kokoro" || engine == "xtts" {
		ext = ".wav"
	} else {
		ext = ".mp3"
	}
	filename := fmt.Sprintf("tts_%d%s", timestamp, ext)
	destFilePath := filepath.Join(audioDir, filename)

	// Gerar áudio
	err = generateAudioFile(req.Text, destFilePath, engine, voice, model, req.Rate, req.Pitch)
	if err != nil {
		http.Error(w, "Erro ao sintetizar áudio: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Opcional: Criar arquivo meta contendo o texto original para o histórico
	metaFilename := filename + ".txt"
	_ = os.WriteFile(filepath.Join(audioDir, metaFilename), []byte(req.Text), 0644)

	res := GenerateResponse{
		Success:  true,
		URL:      "/api/audio/" + filename,
		Filename: filename,
		Engine:   engine,
		Text:     req.Text,
		Date:     time.Now().Format("02/01/2006 15:04"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// Handler para listar histórico de áudios
func handleHistory(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	files, err := os.ReadDir(audioDir)
	if err != nil {
		http.Error(w, "Erro ao ler histórico: "+err.Error(), http.StatusInternalServerError)
		return
	}

	audioFiles := []AudioFile{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		if !strings.HasSuffix(name, ".mp3") && !strings.HasSuffix(name, ".wav") {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		// Ler texto original associado
		txtBytes, _ := os.ReadFile(filepath.Join(audioDir, name+".txt"))
		text := string(txtBytes)
		if text == "" {
			text = "Sintetizado via Web App"
		}

		engine := "edge-tts"
		if strings.HasSuffix(name, ".wav") {
			engine = "piper"
		}

		audioFiles = append(audioFiles, AudioFile{
			Filename:  name,
			URL:       "/api/audio/" + name,
			Text:      text,
			Engine:    engine,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	// Ordenar por mais recentes primeiro
	sort.Slice(audioFiles, func(i, j int) bool {
		return audioFiles[i].CreatedAt.After(audioFiles[j].CreatedAt)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(audioFiles)
}

// Handler para servir áudio gerado
func serveAudio(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/audio/")
	// Prevenir Directory Traversal
	filename = filepath.Base(filename)
	path := filepath.Join(audioDir, filename)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "Áudio não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	http.ServeFile(w, r, path)
}

// Auxiliar para gerar o arquivo de áudio utilizando subprocessos
func generateAudioFile(text, destPath, engine, voice, model, rate, pitch string) error {
	absDestPath, err := filepath.Abs(destPath)
	if err != nil {
		return err
	}

	if engine == "edge-tts" {
		args := []string{"--voice", voice, "--text", text}
		if rate != "" && rate != "+0%" && rate != "0%" {
			args = append(args, "--rate", rate)
		}
		if pitch != "" && pitch != "+0Hz" && pitch != "0Hz" {
			args = append(args, "--pitch", pitch)
		}
		args = append(args, "--write-media", absDestPath)

		cmd := exec.Command("edge-tts", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("edge-tts: %v (output: %s)", err, string(out))
		}
		return nil
	}

	if engine == "piper" {
		// Determinar caminho correto do binário do piper
		piperBinary := cfg.PiperBinary
		if piperBinary == "" || piperBinary == "piper" {
			// fallback para o path descoberto pelo bootstrap
			home, _ := os.UserHomeDir()
			piperBinary = filepath.Join(home, ".local", "share", "myvoice", "piper", "piper")
		}
		
		libPath := filepath.Dir(piperBinary)
		
		// O Piper lê do stdin e cospe em --output_file
		cmd := exec.Command(piperBinary, "--model", model, "--output_file", absDestPath)
		cmd.Dir = libPath
		cmd.Env = append(os.Environ(),
			"ESPEAK_DATA_PATH="+filepath.Join(libPath, "espeak-ng-data"),
			"LD_LIBRARY_PATH="+libPath,
		)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		go func() {
			defer stdin.Close()
			_, _ = io.WriteString(stdin, text+"\n")
		}()

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("piper: %v (output: %s)", err, string(out))
		}
		return nil
	}

	if engine == "kokoro" || engine == "xtts" {
		pythonBin := cfg.AIEnginePython
		if pythonBin == "" {
			home, _ := os.UserHomeDir()
			pythonBin = filepath.Join(home, ".local", "share", "myvoice", "ai_env", "bin", "python3")
		}

		// Localizar o service.py
		execPath, _ := os.Executable()
		baseDir := filepath.Dir(execPath)
		if strings.Contains(baseDir, "/tmp/go-build") {
			baseDir, _ = os.Getwd()
		}
		serviceScript := filepath.Join(baseDir, "ai_engine", "service.py")

		args := []string{serviceScript, "--engine", engine, "--text", text, "--out", absDestPath}

		if engine == "kokoro" {
			kokoroVoice := voice
			if kokoroVoice == "" || strings.HasPrefix(kokoroVoice, "pt-BR-") || strings.HasPrefix(kokoroVoice, "en-US-") || kokoroVoice == cfg.EdgeVoice {
				kokoroVoice = cfg.KokoroVoice
				if kokoroVoice == "" {
					kokoroVoice = "pf_dora"
				}
			}
			args = append(args, "--voice", kokoroVoice)
		} else if engine == "xtts" {
			speakerWav := cfg.CloneVoicePath
			if speakerWav == "" {
				return fmt.Errorf("nenhuma voz de referência configurada para clonagem")
			}
			lang := cfg.CloneLanguage
			if lang == "" {
				lang = "pt"
			}
			args = append(args, "--speaker-wav", speakerWav, "--lang", lang)
		}

		cmd := exec.Command(pythonBin, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %v (output: %s)", engine, err, string(out))
		}
		return nil
	}

	return fmt.Errorf("motor desconhecido: %s", engine)
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

type ModelDownloadRequest struct {
	Name    string `json:"name"`
	Quality string `json:"quality"`
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Método não suportado", http.StatusMethodNotAllowed)
		return
	}

	installed, err := getInstalledModels()
	if err != nil {
		http.Error(w, "Erro ao listar modelos instalados: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Curadoria de modelos disponíveis em pt-BR
	available := []map[string]string{
		{"name": "faber", "quality": "medium", "description": "Faber (Masculino - Padrão)", "size": "60MB", "filename": "pt_BR-faber-medium.onnx"},
		{"name": "faber", "quality": "low", "description": "Faber (Masculino - Rápido)", "size": "15MB", "filename": "pt_BR-faber-low.onnx"},
		{"name": "isabela", "quality": "medium", "description": "Isabela (Feminino - Natural)", "size": "55MB", "filename": "pt_BR-isabela-medium.onnx"},
		{"name": "isabela", "quality": "low", "description": "Isabela (Feminino - Rápido)", "size": "15MB", "filename": "pt_BR-isabela-low.onnx"},
		{"name": "edresson", "quality": "low", "description": "Edresson (Masculino - Clássico)", "size": "15MB", "filename": "pt_BR-edresson-low.onnx"},
	}

	response := map[string]interface{}{
		"installed": installed,
		"available": available,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleModelDownload(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não suportado", http.StatusMethodNotAllowed)
		return
	}

	var req ModelDownloadRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Quality == "" {
		http.Error(w, "Parâmetros 'name' e 'quality' são obrigatórios", http.StatusBadRequest)
		return
	}

	err = downloadModelFiles(req.Name, req.Quality)
	if err != nil {
		http.Error(w, "Erro ao baixar modelo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Modelo pt_BR-%s-%s baixado com sucesso!", req.Name, req.Quality),
		"filename": fmt.Sprintf("pt_BR-%s-%s.onnx", req.Name, req.Quality),
	})
}

func getInstalledModels() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(home, ".local", "share", "myvoice")
	
	files, err := os.ReadDir(appDir)
	if err != nil {
		// Se a pasta não existir por algum motivo, criamos
		os.MkdirAll(appDir, 0755)
		return []string{}, nil
	}

	var models []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".onnx") {
			models = append(models, f.Name())
		}
	}
	return models, nil
}

func downloadModelFiles(voiceName, quality string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	appDir := filepath.Join(home, ".local", "share", "myvoice")
	os.MkdirAll(appDir, 0755)

	fileName := fmt.Sprintf("pt_BR-%s-%s.onnx", voiceName, quality)
	urlPath := fmt.Sprintf("https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/pt/pt_BR/%s/%s/%s", voiceName, quality, fileName)

	destPath := filepath.Join(appDir, fileName)
	destJsonPath := destPath + ".json"
	urlJsonPath := urlPath + ".json"

	// 1. Baixar ONNX
	err = downloadHTTPFile(urlPath, destPath)
	if err != nil {
		return fmt.Errorf("falha ao baixar ONNX: %w", err)
	}

	// 2. Baixar ONNX.json
	err = downloadHTTPFile(urlJsonPath, destJsonPath)
	if err != nil {
		// remover arquivo onnx incompleto
		os.Remove(destPath)
		return fmt.Errorf("falha ao baixar especificação JSON: %w", err)
	}

	return nil
}

func downloadHTTPFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status HTTP inexperado: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func handleKokoroVoices(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	voices := []map[string]string{
		{"id": "pf_dora", "name": "Dora (Feminina - Português Brasileiro Nativo)", "lang": "pt-br", "gender": "F"},
		{"id": "pm_alex", "name": "Alex (Masculina - Português Brasileiro Nativo)", "lang": "pt-br", "gender": "M"},
		{"id": "pm_santa", "name": "Papai Noel (Masculina Grave - Português)", "lang": "pt-br", "gender": "M"},
		{"id": "af_heart", "name": "Heart (Feminina - Alta Definição / Suave)", "lang": "en-us", "gender": "F"},
		{"id": "af_bella", "name": "Bella (Feminina - Expressiva)", "lang": "en-us", "gender": "F"},
		{"id": "af_sarah", "name": "Sarah (Feminina - Clara / Profissional)", "lang": "en-us", "gender": "F"},
		{"id": "am_adam", "name": "Adam (Masculina - Firme / Narrador)", "lang": "en-us", "gender": "M"},
		{"id": "am_michael", "name": "Michael (Masculina - Conversacional)", "lang": "en-us", "gender": "M"},
		{"id": "bf_isabella", "name": "Isabella (Feminina - Britânica)", "lang": "en-gb", "gender": "F"},
		{"id": "bm_george", "name": "George (Masculina - Britânica)", "lang": "en-gb", "gender": "M"},
		{"id": "ef_dora", "name": "Dora (Feminina - Espanhol)", "lang": "es", "gender": "F"},
		{"id": "em_alex", "name": "Alex (Masculino - Espanhol)", "lang": "es", "gender": "M"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voices)
}

func handleCloneVoices(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	home, _ := os.UserHomeDir()
	clonesDir := filepath.Join(home, ".local", "share", "myvoice", "clones")
	os.MkdirAll(clonesDir, 0755)

	entries, err := os.ReadDir(clonesDir)
	if err != nil {
		http.Error(w, "Erro ao listar vozes clonadas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var voices []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".wav" || ext == ".mp3" || ext == ".ogg" || ext == ".flac" {
			info, _ := entry.Info()
			cleanName := strings.TrimSuffix(entry.Name(), ext)
			cleanName = strings.ReplaceAll(cleanName, "_", " ")
			cleanName = strings.Title(cleanName)

			voices = append(voices, map[string]interface{}{
				"filename": entry.Name(),
				"path":     filepath.Join(clonesDir, entry.Name()),
				"name":     cleanName,
				"size":     info.Size(),
			})
		}
	}

	if voices == nil {
		voices = make([]map[string]interface{}, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voices)
}

func handleCloneUpload(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não suportado", http.StatusMethodNotAllowed)
		return
	}

	// Limitar tamanho a 25MB
	r.ParseMultipartForm(25 << 20)

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Erro ao receber arquivo de áudio: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	home, _ := os.UserHomeDir()
	clonesDir := filepath.Join(home, ".local", "share", "myvoice", "clones")
	os.MkdirAll(clonesDir, 0755)

	destPath := filepath.Join(clonesDir, header.Filename)
	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Erro ao salvar arquivo no disco: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Erro ao gravar áudio: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Atualizar configuração atual para usar este clone automaticamente
	cfg.CloneVoicePath = destPath
	_ = cfg.Save()

	response := map[string]interface{}{
		"success":  true,
		"filename": header.Filename,
		"path":     destPath,
		"message":  "Áudio de referência de voz importado com sucesso!",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}


