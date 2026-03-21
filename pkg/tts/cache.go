package tts

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/j/myvoice/pkg/config"
	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

type MacroAudio struct {
	PCM        []byte
	SampleRate int
	Stereo     bool
}

// PreGenerateMacro chama um processo rápido e isolado (sem pulse local na pipeline original) para capturar o PCM final
func PreGenerateMacro(text string, cfg *config.AppConfig) (*MacroAudio, error) {
	if cfg.TTSEngine == "edge-tts" {
		return edgeToPCM(text, cfg)
	}
	return piperToPCM(text, cfg)
}

func edgeToPCM(text string, cfg *config.AppConfig) (*MacroAudio, error) {
	ttsCmd := exec.Command("edge-tts", "--voice", cfg.EdgeVoice, "--text", text, "--write-media", "-")
	mp3Stream, err := ttsCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := ttsCmd.Start(); err != nil {
		return nil, err
	}

	mp3Data, err := io.ReadAll(mp3Stream)
	if err != nil {
		ttsCmd.Process.Kill()
		return nil, err
	}
	ttsCmd.Wait()

	if len(mp3Data) < 100 {
		return nil, fmt.Errorf("mp3 vazio atirado da nuvem")
	}

	decoder, err := mp3.NewDecoder(bytes.NewReader(mp3Data))
	if err != nil {
		return nil, err
	}

	pcmData, err := io.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return &MacroAudio{
		PCM:        pcmData,
		SampleRate: decoder.SampleRate(),
		Stereo:     true, // MP3 Decoder do pacote hajimehoshi converte para Stereo por padrão
	}, nil
}

func piperToPCM(text string, cfg *config.AppConfig) (*MacroAudio, error) {
	libPath := filepath.Dir(cfg.PiperBinary)
	piperCmd := exec.Command(cfg.PiperBinary, "--model", cfg.PiperModel, "--output_raw")
	piperCmd.Dir = libPath
	piperCmd.Env = append(os.Environ(),
		"ESPEAK_DATA_PATH="+filepath.Join(libPath, "espeak-ng-data"),
		"LD_LIBRARY_PATH="+libPath,
	)

	stdin, err := piperCmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := piperCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := piperCmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, text+"\n")
	}()

	pcmData, err := io.ReadAll(stdout)
	piperCmd.Wait()

	if err != nil {
		return nil, err
	}

	return &MacroAudio{
		PCM:        pcmData,
		SampleRate: 22050,
		Stereo:     false,
	}, nil
}

// PlayMacroAudio injeta instataneamente o PCM carregado em buffer bruto pra os alto-falantes e bypassa delays de síntese.
func PlayMacroAudio(audio *MacroAudio, cfg *config.AppConfig) error {
	c, err := getPulseClient()
	if err != nil {
		return err
	}

	sinkName := strings.TrimPrefix(cfg.SinkName, "pulse/")
	virtualSink, errSink := c.SinkByID(sinkName)

	virtBuf := &ContinuousBuffer{}
	spkBuf := &ContinuousBuffer{}

	if errSink == nil {
		virtBuf.Write(audio.PCM)
	}
	if cfg.PlayOnSpeaker {
		spkBuf.Write(audio.PCM)
	}

	var channels pulse.PlaybackOption
	if audio.Stereo {
		channels = pulse.PlaybackStereo
	} else {
		channels = pulse.PlaybackMono
	}

	var virStream, spkStream *pulse.PlaybackStream

	// Força o start com o cache pronto
	if errSink == nil {
		virStream, _ = c.NewPlayback(pulse.NewReader(virtBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(audio.SampleRate), channels, pulse.PlaybackSink(virtualSink))
	}
	if cfg.PlayOnSpeaker {
		spkStream, _ = c.NewPlayback(pulse.NewReader(spkBuf, proto.FormatInt16LE), pulse.PlaybackSampleRate(audio.SampleRate), channels)
	}

	if virStream == nil && spkStream == nil {
		return fmt.Errorf("Nenhuma saída de áudio habilitada (Active).")
	}

	if virStream != nil {
		virStream.Start()
	}
	if spkStream != nil {
		spkStream.Start()
	}

	// Calcula fisicamente os segundos necessários para Pulse tocar a Stream baseando nos bytes int16 reais
	bytesPerSample := 2
	numChans := 1
	if audio.Stereo {
		numChans = 2
	}
	bytesPerSec := audio.SampleRate * numChans * bytesPerSample
	playDuration := time.Duration(len(audio.PCM)) * time.Second / time.Duration(bytesPerSec)

	time.Sleep(playDuration + 150*time.Millisecond) // Sobra de +150ms para Pulse Hardware Flush / Jitter Latency

	if virStream != nil {
		virStream.Close()
	}
	if spkStream != nil {
		spkStream.Close()
	}

	return nil
}
