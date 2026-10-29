import { useState, useEffect } from 'react';
import { SettingsPanel } from './components/SettingsPanel';
import { TTSForm } from './components/TTSForm';
import { AudioHistory } from './components/AudioHistory';
import type { AudioFile } from './components/AudioHistory';
import { ModelsPanel } from './components/ModelsPanel';
import { Disc, Radio, RefreshCw, Layers, Sparkles, Globe } from 'lucide-react';

interface AppConfig {
  tts_engine: string;
  piper_model: string;
  piper_binary: string;
  edge_voice: string;
  kokoro_voice?: string;
  clone_voice_path?: string;
  clone_language?: string;
  sink_name: string;
  mic_name: string;
  play_on_speaker: boolean;
  macros: Record<string, string>;
}

const API_BASE = 'http://localhost:8000';

function App() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [history, setHistory] = useState<AudioFile[]>([]);
  const [selectedFile, setSelectedFile] = useState<AudioFile | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadingConfig, setLoadingConfig] = useState(true);
  const [activeView, setActiveView] = useState<'sintetizar' | 'modelos'>('sintetizar');

  // Carregar dados iniciais
  useEffect(() => {
    fetchConfig();
    fetchHistory();
  }, []);

  const fetchConfig = async () => {
    setLoadingConfig(true);
    try {
      const res = await fetch(`${API_BASE}/api/config`);
      if (res.ok) {
        const data = await res.json();
        setConfig(data);
      }
    } catch (err) {
      console.error("Erro ao carregar configurações:", err);
    } finally {
      setLoadingConfig(false);
    }
  };

  const fetchHistory = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/history`);
      if (res.ok) {
        const data = await res.json();
        setHistory(data);
      }
    } catch (err) {
      console.error("Erro ao carregar histórico:", err);
    }
  };

  const handleSaveConfig = async (newConfig: AppConfig) => {
    try {
      const res = await fetch(`${API_BASE}/api/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newConfig),
      });
      if (res.ok) {
        const data = await res.json();
        setConfig(data);
        alert('Configurações salvas com sucesso!');
      } else {
        alert('Erro ao salvar configurações no servidor.');
      }
    } catch (err) {
      console.error(err);
      alert('Erro de conexão com o servidor.');
    }
  };

  const handleGenerate = async (text: string, options: { rate: string; pitch: string; voice: string; model: string }) => {
    if (!config) return;
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text,
          engine: config.tts_engine,
          voice: options.voice,
          model: options.model,
          rate: options.rate,
          pitch: options.pitch,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        if (data.success) {
          // Recarregar histórico de arquivos
          await fetchHistory();
          
          // Formatar URL com caminho absoluto do servidor da API
          const newAudioFile: AudioFile = {
            filename: data.filename,
            url: `${API_BASE}${data.url}`,
            text: data.text,
            engine: data.engine,
            size: 0, // será recalculado no reload do histórico
            created_at: new Date().toISOString(),
          };
          
          // Selecionar áudio gerado para reproduzir no player
          setSelectedFile(newAudioFile);
        } else {
          alert('Erro ao gerar voz.');
        }
      } else {
        const errMsg = await res.text();
        alert(`Erro do servidor: ${errMsg}`);
      }
    } catch (err) {
      console.error(err);
      alert('Erro na conexão para gerar áudio.');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectHistoryFile = (file: AudioFile) => {
    setSelectedFile({
      ...file,
      url: `${API_BASE}${file.url}`
    });
  };

  return (
    <div className="min-height-screen flex flex-col justify-between p-4 md:p-8 max-w-7xl mx-auto w-full gap-6">
      {/* Header */}
      <header className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-white/5">
        <div className="flex items-center gap-3">
          <div className="p-2.5 bg-purple-600/25 border border-purple-500/20 rounded-2xl flex items-center justify-center animate-glow">
            <Disc className="w-8 h-8 text-purple-400 animate-spin" style={{ animationDuration: '6s' }} />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight bg-gradient-to-r from-white via-purple-300 to-cyan-200 bg-clip-text text-transparent flex items-center gap-2">
              CromyVoice <span className="text-xs bg-cyan-500/20 border border-cyan-500/30 text-cyan-300 px-2 py-0.5 rounded-full font-semibold">Web v1.0</span>
            </h1>
            <p className="text-xs text-slate-400 mt-0.5">Interface Web Inteligente para os Motores Edge-TTS & Piper Local</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button 
            onClick={() => { fetchConfig(); fetchHistory(); }}
            className="btn-secondary p-2 rounded-xl text-slate-400 hover:text-white"
            title="Recarregar Dados"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          
          <div className="flex items-center gap-2 bg-black/45 border border-white/5 px-3 py-1.5 rounded-xl text-xs">
            <Radio className="w-3.5 h-3.5 text-cyan-400" />
            <span className="text-slate-400">Mic Virtual:</span>
            {config ? (
              <span className="text-glow-green font-bold">Ativo ({config.mic_name})</span>
            ) : (
              <span className="text-slate-500 font-bold">Buscando...</span>
            )}
          </div>
        </div>
      </header>

      {/* Tabs Navigation */}
      <div className="flex border-b border-white/5 pb-0 text-sm gap-6">
        <button
          onClick={() => setActiveView('sintetizar')}
          className={`pb-2 px-1 transition relative flex items-center gap-1.5 focus:outline-none border-b-2 font-semibold ${
            activeView === 'sintetizar'
              ? 'text-purple-400 border-purple-500'
              : 'text-slate-400 hover:text-slate-200 border-transparent'
          }`}
        >
          <Sparkles className="w-4 h-4" />
          Sintetizador de Voz
        </button>
        <button
          onClick={() => setActiveView('modelos')}
          className={`pb-2 px-1 transition relative flex items-center gap-1.5 focus:outline-none border-b-2 font-semibold ${
            activeView === 'modelos'
              ? 'text-purple-400 border-purple-500'
              : 'text-slate-400 hover:text-slate-200 border-transparent'
          }`}
        >
          <Globe className="w-4 h-4" />
          Biblioteca de Vozes & Modelos
        </button>
      </div>

      {/* Main View Grid */}
      {activeView === 'sintetizar' ? (
        <main className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch flex-1">
          {/* Formulário de Texto (Esquerda) */}
          <section className="lg:col-span-8 h-full">
            <TTSForm
              engine={config?.tts_engine || 'kokoro'}
              defaultVoice={
                config?.tts_engine === 'kokoro'
                  ? (config?.kokoro_voice || 'pf_dora')
                  : (config?.edge_voice || 'pt-BR-AntonioNeural')
              }
              defaultModel={config?.piper_model || 'pt_BR-faber-medium.onnx'}
              onGenerate={handleGenerate}
              loading={loading}
            />
          </section>

          {/* Configurações (Direita) */}
          <section className="lg:col-span-4 h-full">
            <SettingsPanel
              config={config}
              onSaveConfig={handleSaveConfig}
            />
          </section>
        </main>
      ) : (
        <main className="flex-1 h-full">
          <ModelsPanel
            currentModelPath={config?.piper_model || ''}
            onSelectModel={async (modelPath) => {
              if (config) {
                const newConfig = { ...config, piper_model: modelPath };
                await handleSaveConfig(newConfig);
              }
            }}
            apiBase={API_BASE}
          />
        </main>
      )}

      {/* Histórico e Player */}
      <footer className="w-full">
        <AudioHistory
          files={history}
          onPlay={handleSelectHistoryFile}
          selectedFile={selectedFile}
          loading={loadingConfig}
        />
      </footer>

      {/* Créditos Rodapé */}
      <div className="flex items-center justify-between text-[11px] text-slate-500 border-t border-white/5 pt-4 mt-2">
        <span className="flex items-center gap-1">
          <Layers className="w-3.5 h-3.5 text-slate-600" />
          Powered by edge-tts (Microsoft Cloud) & Piper ONNX Local Engine
        </span>
        <span>CromyVoice © 2026</span>
      </div>
    </div>
  );
}

export default App;
