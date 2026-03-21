package tts

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/j/myvoice/pkg/config"
	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

var (
	GlobalPiperStdin io.WriteCloser
	piperCmd         *exec.Cmd

	pulseClient    *pulse.Client
	virtualStream  *pulse.PlaybackStream
	speakerStream  *pulse.PlaybackStream
)

// ContinuousBuffer protege a goroutine vital do PulseAudio injetando Zeros (Silêncio) quando não há bytes.
// Evitando Timeouts e o "Panic: Send on Closed Channel" causado pelo Daemon matando client que trava.
type ContinuousBuffer struct {
	mu        sync.Mutex
	data      []byte
	lastWrite time.Time
	playing   bool
}

func (b *ContinuousBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data = append(b.data, p...)
	b.lastWrite = time.Now()
	// Não liga a b.playing aqui. A represa D'água se auto-regula no Read.
	return len(p), nil
}

func (b *ContinuousBuffer) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Se secou tudo, fecha a comporta e devolve zero para não dar timeout no Kernel.
	if len(b.data) == 0 {
		b.playing = false
		for i := range p { p[i] = 0 }
		return len(p), nil
	}

	// JITTER BUFFER AUTO-HEALING (WATERMARK):
	// A placa de som pede aúdio de 4 em 4KB. O Piper entrega aos trancos.
	// Represamos a água até bater 16KB de onda contínua.
	if !b.playing {
		// Se não bateu 16KB e o produtor de IA ainda estiver injetando texto forte (<50ms delay)
		if len(b.data) < 16384 && time.Since(b.lastWrite) < 50*time.Millisecond {
			for i := range p { p[i] = 0 }
			return len(p), nil // Entrega silencio segurando a onda
		}
		// Atingiu o volume seguro de pressão ou o produtor de IA parou (bateu ponto final)! Abre comporta.
		b.playing = true
	}

	n = copy(p, b.data)
	b.data = b.data[n:]
	
	// Limpa frame residual final com zeros garantindo buffer limpo de alocação antiga
	if n < len(p) {
		for i := n; i < len(p); i++ { p[i] = 0 }
	}

	return len(p), nil
}

func getPulseClient() (*pulse.Client, error) {
	if pulseClient != nil {
		return pulseClient, nil
	}
	c, err := pulse.NewClient(pulse.ClientApplicationName("MyVoice Bot Integrator"))
	if err != nil {
		return nil, err
	}
	pulseClient = c
	return c, nil
}

func StartContinuousPipeline(cfg *config.AppConfig) error {
	if cfg.TTSEngine != "piper" { return nil }
	
	// Força diretório nativo para a Engine encontrar a pasta invisível 'espeak-ng-data' do Dicionário Pt-BR!
	libPath := filepath.Dir(cfg.PiperBinary)

	// Inject STDBUF bypasser para Stream fluido + Pipes destravados pra Log Dev
	piperCmd = exec.Command("stdbuf", "-i0", "-o0", "-e0", cfg.PiperBinary, "--model", cfg.PiperModel, "--output_raw")
	piperCmd.Dir = libPath
	
	// Força reconhecimento global da Biblioteca Dinâmica E do Espeak Data Path C++
	piperCmd.Env = append(os.Environ(), 
		"ESPEAK_DATA_PATH="+filepath.Join(libPath, "espeak-ng-data"),
		"LD_LIBRARY_PATH="+libPath,
	)

	stdin, err := piperCmd.StdinPipe()
	if err != nil { return err }
	GlobalPiperStdin = stdin

	stdout, err := piperCmd.StdoutPipe()
	if err != nil { return err }

	piperStderr, err := piperCmd.StderrPipe()
	if err == nil {
		go func() {
			scanner := bufio.NewScanner(piperStderr)
			for scanner.Scan() {
				if cfg.DevMode {
					fmt.Printf("\r\n\033[31m[DEV PIPER ERRO]\033[0m %s\r\n", scanner.Text())
				}
			}
		}()
	}

	c, err := getPulseClient()
	if err != nil { return fmt.Errorf("pulse local socket client: %v", err) }

	sinkName := strings.TrimPrefix(cfg.SinkName, "pulse/")
	virtualSink, errSink := c.SinkByID(sinkName)
	// Removemos o return error aqui para permitir uso APENAS com Alto-Falante se o Mic estiver OFFLINE.

	sampleRate := 22050

	if cfg.PlayOnSpeaker {
		virtBuf := &ContinuousBuffer{}
		spkBuf := &ContinuousBuffer{}

		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					if errSink == nil {
						virtBuf.Write(buf[:n])
					}
					spkBuf.Write(buf[:n])
					if cfg.DevMode {
						fmt.Printf("\r\n\033[36m[DEV PIPER SUCESSO]\033[0m C++ Atirou %d bytes na RAM.\r\n", n)
					}
				}
				if err != nil { break }
			}
		}()

		if errSink == nil {
			virtualStream, _ = c.NewPlayback(pulse.NewReader(virtBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(sampleRate), pulse.PlaybackMono, pulse.PlaybackSink(virtualSink))
		}
		speakerStream, _ = c.NewPlayback(pulse.NewReader(spkBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(sampleRate), pulse.PlaybackMono) 
	} else {
		virtBuf := &ContinuousBuffer{}
		
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					if errSink == nil {
						virtBuf.Write(buf[:n])
					}
					if cfg.DevMode {
						fmt.Printf("\r\n\033[36m[DEV PIPER SUCESSO]\033[0m C++ Atirou %d bytes na RAM.\r\n", n)
					}
				}
				if err != nil { break }
			}
		}()

		if errSink == nil {
			virtualStream, _ = c.NewPlayback(pulse.NewReader(virtBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(sampleRate), pulse.PlaybackMono, pulse.PlaybackSink(virtualSink))
		}
	}

	if virtualStream == nil && speakerStream == nil {
		return fmt.Errorf("Nenhuma saída de áudio válida. Ligue o Virtual Mic ou o Retorno Alto-falante.")
	}

	if err := piperCmd.Start(); err != nil {
		return fmt.Errorf("falha ao ligar a IA local: %v", err)
	}

	if virtualStream != nil { go virtualStream.Start() }
	if speakerStream != nil { go speakerStream.Start() }
    
	return nil
}

func StopContinuousPipeline() {
	if virtualStream != nil {
		virtualStream.Close()
		virtualStream = nil
	}
	if speakerStream != nil {
		speakerStream.Close()
		speakerStream = nil
	}
	if GlobalPiperStdin != nil {
		GlobalPiperStdin.Close() 
		GlobalPiperStdin = nil
	}
	if piperCmd != nil && piperCmd.Process != nil {
		piperCmd.Process.Kill()
	}
}

func GenerateAndPlay(text string, audioDevice string, cfg *config.AppConfig) error {
	if cfg.TTSEngine == "edge-tts" {
		return playUsingEdge(text, audioDevice, cfg) 
	}

	if GlobalPiperStdin == nil {
		if err := StartContinuousPipeline(cfg); err != nil {
			return fmt.Errorf("fluxo corrompido (Auto-Wake Local IA falhou): %v", err)
		}
	}

	if GlobalPiperStdin != nil {
		_, err := io.WriteString(GlobalPiperStdin, text+"\n")
		return err
	}

	return fmt.Errorf("fluxo corrompido: Piper ausente/nulo na Fila Principal.")
}

func playUsingEdge(text string, audioDevice string, cfg *config.AppConfig) error {
	ttsCmd := exec.Command("edge-tts", "--voice", cfg.EdgeVoice, "--text", text, "--write-media", "-")
	
	mp3Stream, err := ttsCmd.StdoutPipe()
	if err != nil { return err }

	if cfg.DevMode {
		fmt.Printf("\r\n\033[36m[DEV EDGE INFO]\033[0m Requisitando áudio para: '%s'\r\n", text)
	}

	if err := ttsCmd.Start(); err != nil {
		return fmt.Errorf("cloud host ou executável 'edge-tts' falhou: %v", err)
	}

	// Bloqueante in-memory seguro: Lê 100% o payload MP3 da internet
	// antes de forçar parse. Isso previne o 'seek |0: illegal seek',
	// pois prové um arquivo na RAM que suporta Seeker de Metadata (ID3).
	mp3Data, err := io.ReadAll(mp3Stream)
	if err != nil {
		ttsCmd.Process.Kill()
		return fmt.Errorf("rede instável, quebra de pacote TCP: %v", err)
	}

	ttsCmd.Wait()

	if len(mp3Data) < 100 {
		return fmt.Errorf("edge-tts atirou MP3 vazio. Verifique sua conexão web.")
	}

	if cfg.DevMode {
		fmt.Printf("\r\n\033[36m[DEV EDGE SUCESSO]\033[0m Áudio em RAM (%d bytes).\r\n", len(mp3Data))
	}

	decoder, err := mp3.NewDecoder(bytes.NewReader(mp3Data))
	if err != nil {
		return fmt.Errorf("MP3 decoder panics: Erro na conversão para WAVE (%v)", err)
	}

	c, err := getPulseClient()
	if err != nil { return err }

	sinkName := strings.TrimPrefix(cfg.SinkName, "pulse/")
	virtualSink, errSink := c.SinkByID(sinkName)
	
	sampleRate := decoder.SampleRate()
	
	var virStream, spkStream *pulse.PlaybackStream

	virtBuf := &ContinuousBuffer{}
	spkBuf := &ContinuousBuffer{}
	
	var decodeDone sync.WaitGroup
	decodeDone.Add(1)

	if cfg.PlayOnSpeaker {
		go func() {
			defer decodeDone.Done()
			buf := make([]byte, 4096)
			for {
				n, err := decoder.Read(buf)
				if n > 0 {
					if errSink == nil {
						virtBuf.Write(buf[:n])
					}
					spkBuf.Write(buf[:n])
				}
				if err != nil { break }
			}
		}()

		if errSink == nil {
			virStream, _ = c.NewPlayback(pulse.NewReader(virtBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(sampleRate), pulse.PlaybackStereo, pulse.PlaybackSink(virtualSink))
		}
		spkStream, _ = c.NewPlayback(pulse.NewReader(spkBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(sampleRate), pulse.PlaybackStereo)
	} else {
		go func() {
			defer decodeDone.Done()
			buf := make([]byte, 4096)
			for {
				n, err := decoder.Read(buf)
				if n > 0 {
					if errSink == nil {
						virtBuf.Write(buf[:n])
					}
				}
				if err != nil { break }
			}
		}()

		if errSink == nil {
			virStream, _ = c.NewPlayback(pulse.NewReader(virtBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(sampleRate), pulse.PlaybackStereo, pulse.PlaybackSink(virtualSink))
		}
	}
	
	if virStream == nil && spkStream == nil {
		return fmt.Errorf("Nenhuma saída ativa para reproduzir áudio.")
	}

	if virStream != nil { virStream.Start() }
	if spkStream != nil { spkStream.Start() }

	ttsCmd.Wait()
	
	// Previne que a thread continue antes de descarregar todo o mp3 nos buffers
	decodeDone.Wait()
	
	// Calcula fisicamente os segundos necessários para Pulse tocar a Stream MP3
	bytesPerSample := 2
	numChans := 2 // Decode do MP3 do Edge é Stereo convert
	bytesPerSec := decoder.SampleRate() * numChans * bytesPerSample
	playDuration := time.Duration(decoder.Length()) * time.Second / time.Duration(bytesPerSec)

	time.Sleep(playDuration + 150*time.Millisecond)

	if virStream != nil { virStream.Close() }
	if spkStream != nil { spkStream.Close() }

	return nil
}
