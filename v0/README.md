# 🎙️ MyVoice (v0)

O **MyVoice** é uma aplicação autônoma de linha de comando desenvolvida em Go que atua como um Microfone Virtual inteligente no Linux. Ele injeta respostas de voz geradas por inteligência artificial (Text-to-Speech) de forma nativa e paralela, podendo ser captado em chamadas do Google Meet, Discord, Microsoft Teams, entre outros.

---

## 🚀 Principais Funcionalidades

- **Microfone Virtual Nativamente Acoplado**: Utiliza `pulse` para criar e injetar áudio em um *Sink* do PulseAudio / PipeWire. O áudio do app entra diretamente como um microfone real do sistema operacional.
- **Dois Motores de Text-to-Speech**:
  - `edge-tts`: Geração de voz de alta qualidade suportada pela nuvem neural da Microsoft Edge.
  - `piper` (Modo Nível Nativo Ultra-Performance): Execução local e veloz via modelos ONNX, assegurando zero delay na síntese.
- **Cache de Atalhos em Memória (Macros)**: Cadastre áudios recorrentes e chame-os instantaneamente com os comandos `ALT + 0 a 9`. O áudio já fica pré-sintetizado durante a inicialização na RAM (`MacroCache`).
- **Conversa Assíncrona LIVE (Live Async / Streaming Raw)**: Geração paralela da fala baseada na cadência da escrita. Defina palavras e a IA interpretará blocos instantaneamente *antes mesmo de você pressionar Enter*, garantindo comunicação fluida na chamada.
- **Retorno no Alto-Falante**: Monitore exatamente o som que está sendo emitido na chamada em seu próprio canal de áudio desktop.
- **Modo Desenvolvedor (DevMode)**: Depuração crua, ideal para monitorar syscalls do CGO e gargalos nas engines.

## ⚙️ Pré-Requisitos e Ambiente

- **Sistema Operacional**: Linux (Testado em ambientes Linux Desktop).
- **Servidor de Áudio**: `PulseAudio` ou `PipeWire` rodando daemon `pipewire-pulse`. É necessário o `pactl` no sistema para a criação do sink.
- **Linguagem**: Go 1.25.0+.
- **Compilador C**: Extensões do subsistema podem necessitar de `cgo`.

## 🛠️ Instalação e Execução

Clone o repositório ou navegue até a pasta raiz:

```bash
cd v0
```

Para rodar a aplicação em modo interativo direto pelo terminal:

```bash
go run cmd/myvoice/main.go
```

Ou recompile para uso direto no seu `$PATH`:

```bash
go build -o myvoice_app cmd/myvoice/main.go
./myvoice_app
```

## 📂 Arquitetura do Projeto

O código está modularizado para assegurar manutenibilidade:

- `cmd/myvoice/`: Ponto de entrada (CLI interativa, TTY raw mode handlers).
- `pkg/audio/`: Funções encapsuladas para a injeção em APIs do sistema de som (PulseAudio). Criação e Destruição via `pacmd`/`pactl`.
- `pkg/bootstrap/`: Validação de infraestrutura e `health_check` antes do runtime.
- `pkg/config/`: Estrutura de persistência no *config.json* (`Macros`, `piper_model`, `edge_voice`, etc).
- `pkg/tts/`: Orquestrador dos modelos Cloud e Nativo. Processos de fila/geração e persistência em canais (Goroutines).

---

> Desenvolvido com foco na integração ininterrupta de IA como ferramenta de comunicação assíncrona.
