import React, { useState } from 'react';
import { Play, Sparkles, Sliders, Volume2, Type } from 'lucide-react';

interface TTSFormProps {
  engine: string;
  defaultVoice: string;
  defaultModel: string;
  onGenerate: (text: string, options: { rate: string; pitch: string; voice: string; model: string }) => Promise<void>;
  loading: boolean;
}

export const TTSForm: React.FC<TTSFormProps> = ({ engine, defaultVoice, defaultModel, onGenerate, loading }) => {
  const [text, setText] = useState('');
  const [rate, setRate] = useState(0); // -50 a +50%
  const [pitch, setPitch] = useState(0); // -20 a +20Hz

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!text.trim() || loading) return;

    // Formatar parâmetros para a API
    const rateStr = rate >= 0 ? `+${rate}%` : `${rate}%`;
    const pitchStr = pitch >= 0 ? `+${pitch}Hz` : `${pitch}Hz`;

    await onGenerate(text, {
      rate: rateStr,
      pitch: pitchStr,
      voice: defaultVoice,
      model: defaultModel
    });
  };

  const getEngineBadge = () => {
    switch (engine) {
      case 'kokoro':
        return <span className="badge badge-emerald bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">Kokoro-82M (Qualidade Estúdio)</span>;
      case 'xtts':
        return <span className="badge badge-amber bg-amber-500/20 text-amber-300 border border-amber-500/30">XTTS-v2 (Clonagem)</span>;
      case 'piper':
        return <span className="badge badge-blue">Piper (Local)</span>;
      default:
        return <span className="badge badge-purple">Edge-TTS (Nuvem)</span>;
    }
  };

  return (
    <div className="glass-panel p-6 h-full flex flex-col justify-between">
      <form onSubmit={handleSubmit} className="flex-col flex justify-between h-full gap-4">
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-white/5">
          <h2 className="text-xl font-bold flex items-center gap-2 glow-text-secondary">
            <Sparkles className="text-cyan-400 w-5 h-5" />
            Sintetizador de Voz
          </h2>
          {getEngineBadge()}
        </div>

        {/* Text Area */}
        <div className="flex-1 flex flex-col gap-2 min-h-[180px]">
          <div className="flex items-center justify-between text-xs text-slate-400">
            <span className="flex items-center gap-1">
              <Type className="w-3.5 h-3.5 text-slate-500" />
              Texto para Fala
            </span>
            <span>{text.length} caracteres</span>
          </div>
          <textarea
            value={text}
            onChange={e => setText(e.target.value)}
            placeholder="Digite o texto que você deseja transformar em voz e baixar em MP3..."
            className="flex-1 w-full min-h-[140px] resize-none text-sm p-4 bg-black/40 border border-white/5 rounded-xl placeholder:text-slate-600 focus:border-cyan-500/50"
            maxLength={1000}
            disabled={loading}
          />
        </div>

        {/* Sliders (Only for edge-tts) */}
        {engine === 'edge-tts' && (
          <div className="p-4 bg-black/30 rounded-xl border border-white/5 space-y-4">
            <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
              <Sliders className="w-3.5 h-3.5 text-cyan-400" />
              Configurações Finas de Tom e Velocidade
            </h3>
            
            {/* Speed / Rate Slider */}
            <div className="space-y-1.5">
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Velocidade da Fala (Rate)</span>
                <span className="font-semibold text-cyan-400">{rate >= 0 ? `+${rate}%` : `${rate}%`}</span>
              </div>
              <input
                type="range"
                min="-50"
                max="50"
                value={rate}
                onChange={e => setRate(parseInt(e.target.value))}
                className="w-full"
                disabled={loading}
              />
              <div className="flex justify-between text-[9px] text-slate-600">
                <span>Lento (-50%)</span>
                <span>Normal</span>
                <span>Rápido (+50%)</span>
              </div>
            </div>

            {/* Pitch Slider */}
            <div className="space-y-1.5">
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Tom da Voz (Pitch)</span>
                <span className="font-semibold text-cyan-400">{pitch >= 0 ? `+${pitch}Hz` : `${pitch}Hz`}</span>
              </div>
              <input
                type="range"
                min="-20"
                max="20"
                value={pitch}
                onChange={e => setPitch(parseInt(e.target.value))}
                className="w-full"
                disabled={loading}
              />
              <div className="flex justify-between text-[9px] text-slate-600">
                <span>Grave (-20Hz)</span>
                <span>Normal</span>
                <span>Agudo (+20Hz)</span>
              </div>
            </div>
          </div>
        )}

        {engine === 'kokoro' && (
          <div className="p-4 bg-emerald-950/20 rounded-xl border border-emerald-500/20 text-xs text-emerald-300/90 flex items-center gap-3">
            <Volume2 className="w-6 h-6 text-emerald-400 shrink-0" />
            <div>
              <p className="font-semibold text-emerald-300">Motor Kokoro-82M v1.0 Ativo</p>
              <p className="text-[11px] text-emerald-400/70">
                Voz neural ativa: <b>{defaultVoice}</b>. Síntese local de altíssima fidelidade estilo ElevenLabs.
              </p>
            </div>
          </div>
        )}

        {engine === 'xtts' && (
          <div className="p-4 bg-amber-950/20 rounded-xl border border-amber-500/20 text-xs text-amber-300/90 flex items-center gap-3">
            <Volume2 className="w-6 h-6 text-amber-400 shrink-0" />
            <div>
              <p className="font-semibold text-amber-300">Clonagem de Voz XTTS-v2 Ativa</p>
              <p className="text-[11px] text-amber-400/70">
                O áudio será sintetizado em português clonando o timbre do áudio de referência configurado.
              </p>
            </div>
          </div>
        )}

        {engine === 'piper' && (
          <div className="p-4 bg-black/25 rounded-xl border border-white/5 text-center text-xs text-slate-400">
            <Volume2 className="w-5 h-5 mx-auto mb-2 text-cyan-500/50" />
            O motor <b>Piper local</b> gera áudio nativo ultraveloz em CPU.
          </div>
        )}

        {/* Generate Button */}
        <button
          type="submit"
          disabled={loading || !text.trim()}
          className="w-full btn-primary text-base py-3 glow-text-secondary"
        >
          {loading ? (
            <div className="flex items-center gap-2">
              <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              Sintetizando Voz...
            </div>
          ) : (
            <>
              <Play className="w-5 h-5 fill-white" />
              Sintetizar Áudio (Gerar MP3/WAV)
            </>
          )}
        </button>
      </form>
    </div>
  );
};
