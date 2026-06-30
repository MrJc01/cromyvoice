package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/j/myvoice/pkg/audio"
	"github.com/j/myvoice/pkg/bootstrap"
	"github.com/j/myvoice/pkg/config"
	"github.com/j/myvoice/pkg/tts"
)

var (
	MicIsActive bool
	MacroCache  map[string]*tts.MacroAudio
)

func main() {
	appDir, err := bootstrap.EnsureEnvironment()
	if err != nil {
		fmt.Println("⚠️ O processo Autônomo falhou (você dependerá unicamente da versão Web Cloud):", err)
	}

	cfg := config.LoadConfig(appDir)

	textFlag := flag.String("text", "", "Text to synthesize")
	outFlag := flag.String("out", "", "Output audio file path")
	engineFlag := flag.String("engine", "", "TTS Engine (edge-tts or piper)")
	voiceFlag := flag.String("voice", "", "Edge-tts voice name")
	modelFlag := flag.String("model", "", "Piper model file path")
	flag.Parse()

	if *textFlag != "" && *outFlag != "" {
		if *engineFlag != "" {
			cfg.TTSEngine = *engineFlag
		}
		if *voiceFlag != "" {
			cfg.EdgeVoice = *voiceFlag
		}
		if *modelFlag != "" {
			cfg.PiperModel = *modelFlag
		}

		err := generateToFile(*textFlag, *outFlag, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Erro ao gerar áudio: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Áudio gerado com sucesso em: %s\n", *outFlag)
		os.Exit(0)
	}

	audio.DestroyVirtualMic(cfg.SinkName)
	MicIsActive = false

	hydrateMacros(cfg)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n🧹 Fechando forçado pela SESSÃO EXTERNA. Limpando OS...")
		if MicIsActive {
			tts.StopContinuousPipeline()
			audio.DestroyVirtualMic(cfg.SinkName)
		}
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		micStatus := "🔴 OFFLINE"
		if MicIsActive {
			micStatus = "🟢 ONLINE (" + cfg.MicName + ")"
		}

		speakerStatus := "🔇 Desligado"
		if cfg.PlayOnSpeaker {
			speakerStatus = "🔊 LIGADO"
		}

		devStatus := "OFF"
		if cfg.DevMode {
			devStatus = "ON"
		}

		fmt.Println("\n======================================")
		fmt.Println("🎙️   MyVoice - Painel Principal")
		fmt.Println("======================================")
		fmt.Printf(" Status do Mic Virtual : %s\n", micStatus)
		fmt.Printf(" Retorno Alto-falante  : %s\n", speakerStatus)
		fmt.Println("--------------------------------------")
		fmt.Println("1) 🔌 Ligar / Desligar Microfone Virtual")
		fmt.Println("2) 🎧 Ligar / Desligar Retorno (Sair som no Auto-Falante)")
		fmt.Println("3) 💬 Conversa Normal (Gera e Toca Textos somente ao dar Enter)")
		fmt.Println("4) ⚡ Conversa Assíncrona LIVE (Gera a voz IMEDIATAMENTE enquanto você digita!)")
		fmt.Println("5) ⌨️  Respostas Rápidas (Macros de Áudio ALT+Numero)")
		fmt.Println("6) ⚙️  Configurações Avançadas (Mudar Voz/Modelo)")
		fmt.Printf("7) 🐛 Modo Desenvolvedor (Logs em Tempo Real): [%s]\n", devStatus)
		fmt.Println("8) ❌ Fechar App")
		fmt.Print("\nEscolha [1-8]: ")

		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			if MicIsActive {
				tts.StopContinuousPipeline()
				audio.DestroyVirtualMic(cfg.SinkName)
				MicIsActive = false
				fmt.Println("✅ Microfone desligado com sucesso.")
			} else {
				fmt.Println("⏳ Registrando cabo virtual no SO...")
				err := audio.CreateVirtualMic(cfg.SinkName, cfg.MicName)
				if err != nil {
					fmt.Printf("❌ Falha de Cabo Virtual: %v\n", err)
				} else {
					fmt.Println("⏳ Hidratando Inteligência Artificial (Carregando ONNX Engine)...")
					errPiper := tts.StartContinuousPipeline(cfg)
					if errPiper != nil {
						fmt.Printf("⚠️  Pipeline Nativo de Ultra-Performance falhou (%v). Nuvem Continuará ativa.\n", errPiper)
					}
					MicIsActive = true
					fmt.Println("✅ Microfone ONLINE e Plugado no OS! Escolha no dispositivo de Call.")
				}
			}
		case "2":
			cfg.PlayOnSpeaker = !cfg.PlayOnSpeaker
			cfg.Save()
			if cfg.PlayOnSpeaker {
				fmt.Println("✅ RETORNO LIGADO: Você também ouvirá as mensagens no seu desktop simultaneamente (Zero Delay).")
			} else {
				fmt.Println("✅ RETORNO DESLIGADO: Apenas as pessoas da call ouvirão a IA.")
			}
		case "3":
			if !MicIsActive {
				fmt.Println("\n⚠️ [AVISO] O Microfone da call está OFFLINE.")
			}
			enterChatLoop(cfg, scanner)
		case "4":
			if !MicIsActive {
				fmt.Println("\n⚠️ [AVISO] O Microfone da call está OFFLINE.")
			}
			asyncChatLoop(cfg, scanner)
		case "5":
			openMacrosMenu(cfg, scanner)
		case "6":
			openSettings(cfg, scanner)
		case "7":
			cfg.DevMode = !cfg.DevMode
			if cfg.DevMode {
				fmt.Println("✅ MODO DEV LIGADO: Syscalls CGO e falhas da IA serão mostradas cruas.")
			} else {
				fmt.Println("✅ MODO DEV DESLIGADO: Painéis agora estarão limpos.")
			}
		case "8":
			if MicIsActive {
				tts.StopContinuousPipeline()
				audio.DestroyVirtualMic(cfg.SinkName)
			}
			fmt.Println("👋 Desconectando e encerrando subsistemas...")
			return
		default:
			fmt.Println("⚠️  Opção não identificada.")
		}
	}
}

func enterChatLoop(cfg *config.AppConfig, scanner *bufio.Scanner) {
	fmt.Println("\n===========================================")
	fmt.Println("💬 [MODO DE TEXTO NORMAL]")
	fmt.Println("💡 O botão ENTER trava envia o áudio inteiro.")
	fmt.Println("🚀 DICA: Pressione ALT + Número(1-9,0) para disparar Respostas Rápidas instantâneas!")
	fmt.Println("⛔ Pressione a tecla ESC para sair do chat e ir para o Painel.")
	fmt.Println("===========================================")

	jobs := make(chan string, 100)
    
	go func() {
		for text := range jobs {
			if strings.TrimSpace(text) != "" {
				err := tts.GenerateAndPlay(text, "pulse/"+cfg.SinkName, cfg)
				if err != nil {
					fmt.Printf("\r\n❌ Erro ao processar voz: %v\r\n", err)
				}
			}
		}
	}()

	fmt.Print("\r\n[Normal] -> ")

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Sua distro não suporta Kernel Raw TTY:", err)
		return
	}

	var buf []rune
	reader := bufio.NewReader(os.Stdin)

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		if r == 3 { 
			term.Restore(int(os.Stdin.Fd()), oldState)
			close(jobs)
			fmt.Println("\r\n🧹 Fechando forçado pela SESSÃO EXTERNA. Limpando OS...")
			if MicIsActive {
				tts.StopContinuousPipeline()
				audio.DestroyVirtualMic(cfg.SinkName)
			}
			os.Exit(0)
		}

		if r == 27 { 
			time.Sleep(15 * time.Millisecond)
			if reader.Buffered() > 0 {
				nextR, _, _ := reader.ReadRune()
				if nextR >= '0' && nextR <= '9' {
					macroTxt := cfg.Macros[string(nextR)]
					if macroTxt != "" {
						fmt.Printf("\r\n🎯 [ATALHO ALT+%c Disparado]: %s\r\n", nextR, macroTxt)
						if cachedAudio, exists := MacroCache[string(nextR)]; exists && cachedAudio != nil {
							go func() {
								err := tts.PlayMacroAudio(cachedAudio, cfg)
								if err != nil {
									fmt.Printf("\r\n❌ Erro no Atalho (Local): %v\r\n[Normal] -> ", err)
								}
							}()
						} else {
							jobs <- macroTxt
						}
					}
					fmt.Print("\r\n[Normal] -> " + string(buf))
					continue
				}
			}
			break
		}

		if r == '\r' || r == '\n' {
			if len(buf) > 0 {
				jobs <- strings.TrimSpace(string(buf))
				buf = buf[:0]
			}
			fmt.Print("\r\n[Normal] -> ")
			continue
		}

		if r == 127 || r == 8 {
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Print("\b \b")
			}
			continue
		}

		buf = append(buf, r)
		fmt.Print(string(r))
	}

	term.Restore(int(os.Stdin.Fd()), oldState)
	close(jobs)
	fmt.Println("\nSaindo da sessão ativa...")
}

func asyncChatLoop(cfg *config.AppConfig, scanner *bufio.Scanner) {
	fmt.Println("\n===========================================")
	fmt.Println("⚡ [MODO DE LIVE TYPING / STREAMING RAW]")
	fmt.Println("💡 O áudio é paralelizado e a tela nunca trava.")
	fmt.Println("🚀 DICA: Pressione ALT + Número(1-9,0) para disparar Respostas Rápidas instantâneas no meio da sua digitação sem perder suas letras!")

	fmt.Print("\nDefina a quebra de geração (Nº de palavras ex: 3, 5  |  OU pontuação ex: , ou .): ")
	if !scanner.Scan() {
		return
	}
	triggerInput := strings.TrimSpace(scanner.Text())
	
	isWordChunk := true
	wordLimit := 3
	charTrigger := rune(0)

	if val, err := strconv.Atoi(triggerInput); err == nil && val > 0 {
		isWordChunk = true
		wordLimit = val
	} else if len(triggerInput) > 0 {
		isWordChunk = false
		charTrigger = []rune(triggerInput)[0]
	}

	if isWordChunk {
		fmt.Printf("✅ A IA vai gerar voz na exata mesma hora que você apertar ESPAÇO no seu %dº agrupamento.\n", wordLimit)
	} else {
		fmt.Printf("✅ A IA vai gerar voz instantaneamente no exato momento que você apertar '%c' sem depender de Enter.\n", charTrigger)
	}

	jobs := make(chan string, 100)
    
	go func() {
		for text := range jobs {
			if strings.TrimSpace(text) != "" {
				err := tts.GenerateAndPlay(text, "pulse/"+cfg.SinkName, cfg)
				if err != nil {
					fmt.Printf("\r\n❌ Erro local (Engine Crashed): %v\r\n", err)
				}
			}
		}
	}()

	fmt.Println("⛔ Pressione a tecla 'ESC' do seu teclado para cancelar e voltar à home.")
	fmt.Println("===========================================")
	fmt.Print("\r\n[Live Async] -> ")

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Sua distro não suporta Kernel Raw TTY:", err)
		return
	}

	var buf []rune
	wordCount := 0
	reader := bufio.NewReader(os.Stdin)

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		if r == 3 { 
			term.Restore(int(os.Stdin.Fd()), oldState)
			close(jobs)
			fmt.Println("\r\n🧹 Ctrl+C pressionado em RAW (Modo Deus). Destruindo microfones e saindo do app...")
			if MicIsActive {
				tts.StopContinuousPipeline()
				audio.DestroyVirtualMic(cfg.SinkName)
			}
			os.Exit(0)
		}

		if r == 27 { 
			time.Sleep(15 * time.Millisecond) 
			if reader.Buffered() > 0 {
				nextR, _, _ := reader.ReadRune()
				if nextR >= '0' && nextR <= '9' {
					macroTxt := cfg.Macros[string(nextR)]
					if macroTxt != "" {
						fmt.Printf("\r\n🎯 [ATALHO ALT+%c Disparado]: %s\r\n", nextR, macroTxt)
						if cachedAudio, exists := MacroCache[string(nextR)]; exists && cachedAudio != nil {
							go func() {
								err := tts.PlayMacroAudio(cachedAudio, cfg)
								if err != nil {
									fmt.Printf("\r\n❌ Erro no Atalho (Local): %v\r\n[Live Async] -> ", err)
								}
							}()
						} else {
							jobs <- macroTxt
						}
					}
					fmt.Print("\r\n[Live Async] -> " + string(buf))
					continue
				}
			}
			break
		}

		if r == '\r' || r == '\n' { 
			if len(buf) > 0 {
				jobs <- strings.TrimSpace(string(buf))
				buf = buf[:0]
				wordCount = 0
			}
			fmt.Print("\r\n[Live Async] -> ")
			continue
		}

		if r == 127 || r == 8 { 
			if len(buf) > 0 {
				if buf[len(buf)-1] == ' ' {
					wordCount--
				}
				buf = buf[:len(buf)-1]
				fmt.Print("\b \b") 
			}
			continue
		}

		buf = append(buf, r)
		fmt.Print(string(r))

		triggerReady := false

		if isWordChunk {
			if r == ' ' { 
				wordCount++
				if wordCount >= wordLimit {
					triggerReady = true
				}
			}
		} else {
			if r == charTrigger {
				triggerReady = true
			}
		}

		if triggerReady {
			jobs <- strings.TrimSpace(string(buf))
			buf = buf[:0]
			wordCount = 0
		}
	}

	term.Restore(int(os.Stdin.Fd()), oldState)
	close(jobs)
	fmt.Println("\n✔️ Voltando ao Shell seguro da GUI (Line Mode Restaurado). Aguarde...")
}

func openMacrosMenu(cfg *config.AppConfig, scanner *bufio.Scanner) {
	for {
		fmt.Println("\n======================================")
		fmt.Println("🚀  Configuração de Respostas Rápidas (Macros)")
		fmt.Println("💡 No Modo Normal ou Async, aperte ALT + NÚMERO (1-9,0) para disparar atalhos.")
		fmt.Println("--------------------------------------")
		for i := 1; i <= 9; i++ {
			k := fmt.Sprintf("%d", i)
			fmt.Printf("%s) [%s]\n", k, cfg.Macros[k])
		}
		fmt.Printf("0) [%s]\n", cfg.Macros["0"])
		fmt.Println("--------------------------------------")
		fmt.Println("V) 💾 Salvar e Voltar")

		fmt.Print("\nEscolha qual Atalho editar [1-9, 0 ou V]: ")
		if !scanner.Scan() { return }
		opt := strings.ToUpper(strings.TrimSpace(scanner.Text()))

		if opt == "V" {
			cfg.Save()
			hydrateMacros(cfg)
			return
		}

		if (opt >= "0" && opt <= "9") {
			fmt.Printf("Digite a fala do Atalho ALT + %s (Enter vazio para apagar): ", opt)
			if scanner.Scan() {
				cfg.Macros[opt] = strings.TrimSpace(scanner.Text())
			}
		} else {
			fmt.Println("⚠️ Opção Inexistente.")
		}
	}
}

func openSettings(cfg *config.AppConfig, scanner *bufio.Scanner) {
	for {
		fmt.Println("\n--- ⚙️  Setup de Voz Avançado ---")
		fmt.Printf("1) Engine de TTS  : [%s]\n", cfg.TTSEngine)
		fmt.Printf("2) Voz do Edge    : [%s]\n", cfg.EdgeVoice)
		fmt.Printf("3) Modelo Piper   : [%s]\n", cfg.PiperModel)
		fmt.Printf("4) Nome do Mic    : [%s]\n", cfg.MicName)
		fmt.Println("5) 💾 Salvar e Voltar ao Início")
		fmt.Print("\nO que vai alterar? [1-5]: ")

		if !scanner.Scan() {
			return
		}
		opt := strings.TrimSpace(scanner.Text())

		switch opt {
		case "1":
			if cfg.TTSEngine == "edge-tts" {
				cfg.TTSEngine = "piper"
			} else {
				cfg.TTSEngine = "edge-tts"
			}
			fmt.Printf("✅ Motor definido para: %s\n", cfg.TTSEngine)
		case "2":
			fmt.Print("Nova Voz (ex: pt-BR-AntonioNeural, pt-BR-FranciscaNeural): ")
			if scanner.Scan() {
				val := strings.TrimSpace(scanner.Text())
				if val != "" {
					cfg.EdgeVoice = val
				}
			}
		case "3":
			fmt.Print("Arquivo do Modelo (ex: pt_BR-faber-medium.onnx): ")
			if scanner.Scan() {
				val := strings.TrimSpace(scanner.Text())
				if val != "" {
					cfg.PiperModel = val
				}
			}
		case "4":
			fmt.Print("Novo Nome de Tela (ex: Fake_Bot, Mic_IA, Bot_RH): ")
			if scanner.Scan() {
				val := strings.TrimSpace(scanner.Text())
				if val != "" {
					cfg.MicName = val
				}
			}
		case "5":
			if err := cfg.Save(); err != nil {
				fmt.Printf("❌ Erro em disco: %v\n", err)
			}
			hydrateMacros(cfg)
			return
		default:
			fmt.Println("⚠️ Opção Inexistente.")
		}
	}
}

func hydrateMacros(cfg *config.AppConfig) {
	fmt.Println("\n⏳ Gerando cache de áudio para atalhos rápido (Zero-Delay)...")
	MacroCache = make(map[string]*tts.MacroAudio)
	for k, text := range cfg.Macros {
		if strings.TrimSpace(text) != "" {
			fmt.Printf("   🎙️  Sintetizando Atalho [%s]...\n", k)
			audioData, err := tts.PreGenerateMacro(text, cfg)
			if err == nil {
				MacroCache[k] = audioData
			} else {
				if cfg.DevMode {
					fmt.Printf("   ⚠️  Falha no atalho [%s]: %v\n", k, err)
				}
			}
		}
	}
	fmt.Println("✅ Cache de atalhos carregado e pronto para uso instantâneo!")
}

func generateToFile(text, outPath string, cfg *config.AppConfig) error {
	if cfg.TTSEngine == "edge-tts" {
		cmd := exec.Command("edge-tts", "--voice", cfg.EdgeVoice, "--text", text, "--write-media", outPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("edge-tts error: %v, output: %s", err, string(out))
		}
		return nil
	} else if cfg.TTSEngine == "piper" {
		libPath := filepath.Dir(cfg.PiperBinary)
		cmd := exec.Command(cfg.PiperBinary, "--model", cfg.PiperModel, "--output_file", outPath)
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
			return fmt.Errorf("piper error: %v, output: %s", err, string(out))
		}
		return nil
	}
	return fmt.Errorf("unknown tts engine: %s", cfg.TTSEngine)
}
