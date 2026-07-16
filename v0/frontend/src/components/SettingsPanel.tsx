import React, { useState, useEffect } from 'react';
import { Settings, Save, Volume2, VolumeX, Radio, HardDrive, ListPlus } from 'lucide-react';

interface AppConfig {
  tts_engine: string;
  piper_model: string;
  piper_binary: string;
  edge_voice: string;
  sink_name: string;
  mic_name: string;
  play_on_speaker: boolean;
  macros: Record<string, string>;
}

interface SettingsPanelProps {
  config: AppConfig | null;
  onSaveConfig: (newConfig: AppConfig) => Promise<void>;
}

const EDGE_VOICES = [
  { value: 'pt-BR-FranciscaNeural', label: 'Francisca (Feminina - Suave)' },
  { value: 'pt-BR-AntonioNeural', label: 'Antonio (Masculino - Natural)' },
  { value: 'pt-BR-ThalitaNeural', label: 'Thalita (Feminina - Conversacional)' },
  { value: 'pt-BR-ValerioNeural', label: 'Valerio (Masculino - Formal)' },
];

export const SettingsPanel: React.FC<SettingsPanelProps> = ({ config, onSaveConfig }) => {
  const [localConfig, setLocalConfig] = useState<AppConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState<'audio' | 'macros'>('audio');

  useEffect(() => {
    if (config) {
      setLocalConfig({ ...config });
    }
  }, [config]);

  if (!localConfig) {
    return (
      <div className="glass-panel p-6 flex items-center justify-center h-full min-h-[400px]">
        <div className="text-center text-slate-400">Carregando configurações...</div>
      </div>
    );
  }

  const handleChange = (field: keyof AppConfig, value: any) => {
    setLocalConfig(prev => {
      if (!prev) return null;
      return { ...prev, [field]: value };
    });
  };

  const handleMacroChange = (key: string, value: string) => {
    setLocalConfig(prev => {
      if (!prev) return null;
      return {
        ...prev,
        macros: {
          ...prev.macros,
          [key]: value
        }
      };
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await onSaveConfig(localConfig);
    } catch (err) {
      alert('Erro ao salvar as configurações.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="glass-panel p-6 h-full flex flex-col">
      <div className="flex items-center justify-between mb-6 pb-4 border-b border-white/5">
        <h2 className="text-xl font-bold flex items-center gap-2 glow-text-primary">
          <Settings className="w-5 h-5 text-purple-400" />
          Painel de Controle
        </h2>
        <div className="flex bg-black/40 rounded-lg p-1 text-xs">
          <button
            onClick={() => setActiveTab('audio')}
            className={`px-3 py-1.5 rounded-md transition ${activeTab === 'audio' ? 'bg-purple-600/30 text-purple-300 font-semibold border border-purple-500/20' : 'text-slate-400 hover:text-slate-200'}`}
          >
            Voz & Sistema
          </button>
          <button
            onClick={() => setActiveTab('macros')}
            className={`px-3 py-1.5 rounded-md transition ${activeTab === 'macros' ? 'bg-purple-600/30 text-purple-300 font-semibold border border-purple-500/20' : 'text-slate-400 hover:text-slate-200'}`}
          >
            Atalhos (Macros)
          </button>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="flex-1 flex flex-col justify-between">
        {activeTab === 'audio' ? (
          <div className="space-y-4 overflow-y-auto max-h-[480px] pr-2">
            {/* TTs Engine */}
            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-semibold text-slate-300">Motor de Síntese Padrão</label>
              <select
                value={localConfig.tts_engine}
                onChange={e => handleChange('tts_engine', e.target.value)}
                className="w-full"
              >
                <option value="edge-tts">Microsoft Edge TTS (Nuvem Neural - Grátis)</option>
                <option value="piper">Piper TTS (Local ONNX - Ultra Rápido)</option>
              </select>
            </div>

            {/* Edge Voice */}
            {localConfig.tts_engine === 'edge-tts' && (
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-slate-300">Voz do Microsoft Edge</label>
                <select
                  value={localConfig.edge_voice}
                  onChange={e => handleChange('edge_voice', e.target.value)}
                  className="w-full"
                >
                  {EDGE_VOICES.map(v => (
                    <option key={v.value} value={v.value}>{v.label}</option>
                  ))}
                </select>
              </div>
            )}

            {/* Piper Model */}
            {localConfig.tts_engine === 'piper' && (
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-semibold text-slate-300 flex items-center gap-1">
                  <HardDrive className="w-4 h-4 text-cyan-400" />
                  Modelo Piper (.onnx)
                </label>
                <input
                  type="text"
                  value={localConfig.piper_model}
                  onChange={e => handleChange('piper_model', e.target.value)}
                  placeholder="/caminho/do/modelo.onnx"
                  className="w-full text-xs font-mono"
                />
                <span className="text-[10px] text-slate-500">
                  Modelo ativo padrão: pt_BR-faber-medium.onnx
                </span>
              </div>
            )}

            {/* Speakers / Mic Loop */}
            <div className="p-3 bg-black/30 rounded-xl border border-white/5 space-y-3">
              <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
                <Radio className="w-3.5 h-3.5 text-pink-400" /> Configuração PulseAudio / Cabos
              </h3>

              <div className="grid grid-cols-2 gap-3">
                <div className="flex flex-col gap-1">
                  <label className="text-[11px] text-slate-400 font-medium">Nome do Sink Virtual</label>
                  <input
                    type="text"
                    value={localConfig.sink_name}
                    onChange={e => handleChange('sink_name', e.target.value)}
                    className="w-full text-xs font-mono py-1 px-2"
                  />
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[11px] text-slate-400 font-medium">Nome do Dispositivo</label>
                  <input
                    type="text"
                    value={localConfig.mic_name}
                    onChange={e => handleChange('mic_name', e.target.value)}
                    className="w-full text-xs py-1 px-2"
                  />
                </div>
              </div>

              {/* Toggle Play on Speaker */}
              <div className="flex items-center justify-between pt-2 border-t border-white/5">
                <div className="flex items-center gap-2">
                  {localConfig.play_on_speaker ? (
                    <Volume2 className="w-4 h-4 text-cyan-400" />
                  ) : (
                    <VolumeX className="w-4 h-4 text-slate-500" />
                  )}
                  <div className="flex flex-col">
                    <span className="text-xs font-semibold text-slate-300">Retorno no Alto-falante</span>
                    <span className="text-[10px] text-slate-500">Ouvir o áudio gerado no desktop</span>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => handleChange('play_on_speaker', !localConfig.play_on_speaker)}
                  className={`relative inline-flex h-5 w-10 items-center rounded-full transition-colors duration-200 focus:outline-none ${
                    localConfig.play_on_speaker ? 'bg-cyan-500' : 'bg-slate-700'
                  }`}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform duration-200 ${
                      localConfig.play_on_speaker ? 'translate-x-5.5' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-3 overflow-y-auto max-h-[480px] pr-2">
            <div className="flex items-center gap-1.5 text-xs text-slate-400 pb-2 border-b border-white/5">
              <ListPlus className="w-4 h-4 text-purple-400" />
              <span>Configure textos curtos rápidos para disparar com ALT + [0-9]</span>
            </div>
            
            {Array.from({ length: 10 }).map((_, idx) => {
              const num = idx === 9 ? '0' : `${idx + 1}`;
              return (
                <div key={num} className="flex items-center gap-2 bg-black/20 p-2 rounded-lg border border-white/5">
                  <div className="w-14 text-center font-bold text-xs text-purple-400 bg-purple-500/10 border border-purple-500/20 py-1.5 rounded">
                    ALT + {num}
                  </div>
                  <input
                    type="text"
                    value={localConfig.macros[num] || ''}
                    onChange={e => handleMacroChange(num, e.target.value)}
                    placeholder="Sem macro cadastrada..."
                    className="flex-1 text-xs py-1.5 px-3 bg-black/40 border-none"
                  />
                </div>
              );
            })}
          </div>
        )}

        <div className="mt-6 pt-4 border-t border-white/5">
          <button
            type="submit"
            disabled={saving}
            className="w-full btn-primary py-2.5 text-sm"
          >
            {saving ? (
              <>Salvando...</>
            ) : (
              <>
                <Save className="w-4 h-4" />
                Salvar Configurações
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  );
};
