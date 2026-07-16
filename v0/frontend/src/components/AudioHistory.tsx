import { useState, useRef, useEffect } from 'react';
import { Play, Pause, Download, History, Volume2, FileAudio, Clock, Search } from 'lucide-react';

export interface AudioFile {
  filename: string;
  url: string;
  text: string;
  engine: string;
  size: number;
  created_at: string;
}

interface AudioHistoryProps {
  files: AudioFile[];
  onPlay: (file: AudioFile) => void;
  selectedFile: AudioFile | null;
  loading: boolean;
}

export const AudioHistory: React.FC<AudioHistoryProps> = ({ files, onPlay, selectedFile, loading }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(0.8);
  
  const audioRef = useRef<HTMLAudioElement | null>(null);

  // Efeito ao trocar de arquivo selecionado
  useEffect(() => {
    if (selectedFile) {
      if (audioRef.current) {
        audioRef.current.src = selectedFile.url;
        audioRef.current.load();
        audioRef.current.play().then(() => {
          setPlaying(true);
        }).catch(() => {
          setPlaying(false);
        });
      }
    }
  }, [selectedFile]);

  const togglePlay = () => {
    if (!audioRef.current || !selectedFile) return;

    if (playing) {
      audioRef.current.pause();
      setPlaying(false);
    } else {
      audioRef.current.play().then(() => {
        setPlaying(true);
      }).catch(() => {
        setPlaying(false);
      });
    }
  };

  const handleTimeUpdate = () => {
    if (audioRef.current) {
      setCurrentTime(audioRef.current.currentTime);
    }
  };

  const handleLoadedMetadata = () => {
    if (audioRef.current) {
      setDuration(audioRef.current.duration);
    }
  };

  const handleAudioEnded = () => {
    setPlaying(false);
    setCurrentTime(0);
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseFloat(e.target.value);
    setCurrentTime(val);
    if (audioRef.current) {
      audioRef.current.currentTime = val;
    }
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseFloat(e.target.value);
    setVolume(val);
    if (audioRef.current) {
      audioRef.current.volume = val;
    }
  };

  const formatTime = (secs: number) => {
    if (isNaN(secs)) return '0:00';
    const m = Math.floor(secs / 60);
    const s = Math.floor(secs % 60);
    return `${m}:${s < 10 ? '0' : ''}${s}`;
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const formatDate = (isoStr: string) => {
    try {
      const date = new Date(isoStr);
      return date.toLocaleDateString('pt-BR', {
        day: '2-digit',
        month: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return isoStr;
    }
  };

  const filteredFiles = (files || []).filter(f =>
    f.text.toLowerCase().includes(searchTerm.toLowerCase()) ||
    f.filename.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="glass-panel p-6 h-full flex flex-col justify-between">
      {/* Elemento HTML5 oculto */}
      <audio
        ref={audioRef}
        onTimeUpdate={handleTimeUpdate}
        onLoadedMetadata={handleLoadedMetadata}
        onEnded={handleAudioEnded}
        className="hidden"
      />

      <div className="space-y-4 flex-1 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-white/5">
          <h2 className="text-xl font-bold flex items-center gap-2 glow-text-primary">
            <History className="text-purple-400 w-5 h-5" />
            Histórico & Downloads
          </h2>
          <span className="text-xs text-slate-400 font-medium">
            {(files || []).length} áudios sintetizados
          </span>
        </div>

        {/* Player Customizado */}
        <div className="p-4 bg-black/45 rounded-2xl border border-white/10 shadow-lg">
          {selectedFile ? (
            <div className="space-y-3">
              <div className="flex justify-between items-start gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    {selectedFile.engine === 'piper' ? (
                      <span className="badge badge-blue">Piper (Local)</span>
                    ) : (
                      <span className="badge badge-purple">Edge-TTS (Nuvem)</span>
                    )}
                    <span className="text-[10px] text-slate-500 font-mono truncate">{selectedFile.filename}</span>
                  </div>
                  <p className="text-xs text-slate-200 line-clamp-2 font-medium italic">
                    "{selectedFile.text}"
                  </p>
                </div>

                <a
                  href={selectedFile.url}
                  download={selectedFile.filename}
                  className="btn-secondary py-1.5 px-2.5 rounded-lg text-xs font-semibold flex items-center gap-1 text-cyan-400 border-cyan-500/20 bg-cyan-500/5"
                >
                  <Download className="w-3.5 h-3.5" />
                  Baixar {selectedFile.engine === 'piper' ? 'WAV' : 'MP3'}
                </a>
              </div>

              {/* Progress Slider */}
              <div className="space-y-1">
                <div className="flex items-center gap-3">
                  <span className="text-[10px] text-slate-400 font-mono w-8">{formatTime(currentTime)}</span>
                  <input
                    type="range"
                    min="0"
                    max={duration || 100}
                    value={currentTime}
                    onChange={handleSeek}
                    className="flex-1 h-1.5"
                  />
                  <span className="text-[10px] text-slate-400 font-mono w-8 text-right">{formatTime(duration)}</span>
                </div>
              </div>

              {/* Controls */}
              <div className="flex items-center justify-between pt-1 border-t border-white/5">
                <button
                  onClick={togglePlay}
                  className="w-10 h-10 rounded-full bg-purple-600 hover:bg-purple-500 flex items-center justify-center text-white transition focus:outline-none shadow-md shadow-purple-500/10"
                >
                  {playing ? <Pause className="w-5 h-5 fill-white" /> : <Play className="w-5 h-5 fill-white ml-0.5" />}
                </button>

                {/* Visualizer bars */}
                <div className="flex items-center gap-[2px] h-6 px-4">
                  {playing ? (
                    Array.from({ length: 12 }).map((_, i) => (
                      <div
                        key={i}
                        className="audio-visualizer-bar"
                        style={{ animationDelay: `${i * 0.1}s`, width: '2px' }}
                      />
                    ))
                  ) : (
                    <div className="text-[10px] text-slate-600 font-medium">Player em standby</div>
                  )}
                </div>

                {/* Volume */}
                <div className="flex items-center gap-2 w-28">
                  <Volume2 className="w-4 h-4 text-slate-500" />
                  <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.05"
                    value={volume}
                    onChange={handleVolumeChange}
                    className="w-full h-1"
                  />
                </div>
              </div>
            </div>
          ) : (
            <div className="text-center py-6 text-slate-500 text-xs">
              <FileAudio className="w-8 h-8 mx-auto mb-2 text-slate-600" />
              Nenhum áudio selecionado. Clique em algum item da lista abaixo para reproduzir e baixar.
            </div>
          )}
        </div>

        {/* Busca */}
        <div className="relative mt-2">
          <Search className="w-4 h-4 text-slate-500 absolute left-3 top-2.5" />
          <input
            type="text"
            placeholder="Pesquisar textos gerados..."
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
            className="w-full pl-9 pr-4 py-1.5 text-xs bg-black/30 border-white/5 placeholder:text-slate-600"
          />
        </div>

        {/* Lista de Histórico */}
        <div className="flex-1 overflow-y-auto max-h-[300px] mt-2 space-y-2 pr-1">
          {loading ? (
            <div className="text-center py-8 text-xs text-slate-400">Carregando lista...</div>
          ) : filteredFiles.length === 0 ? (
            <div className="text-center py-8 text-xs text-slate-600">Nenhum áudio no histórico.</div>
          ) : (
            filteredFiles.map(file => {
              const isSelected = selectedFile?.filename === file.filename;
              return (
                <div
                  key={file.filename}
                  onClick={() => onPlay(file)}
                  className={`p-3 rounded-xl border transition cursor-pointer flex justify-between items-center gap-3 ${
                    isSelected
                      ? 'bg-purple-600/10 border-purple-500/30'
                      : 'bg-black/20 hover:bg-black/45 border-white/5'
                  }`}
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-xs text-slate-200 line-clamp-1 font-medium">
                      "{file.text}"
                    </p>
                    <div className="flex items-center gap-3 text-[10px] text-slate-500 mt-1">
                      <span className="flex items-center gap-1 font-semibold text-slate-400">
                        {file.engine === 'piper' ? (
                          <span className="text-cyan-400">Piper</span>
                        ) : (
                          <span className="text-purple-400">Edge</span>
                        )}
                      </span>
                      <span className="flex items-center gap-1">
                        <Clock className="w-3 h-3 text-slate-600" />
                        {formatDate(file.created_at)}
                      </span>
                      <span>{formatSize(file.size)}</span>
                    </div>
                  </div>

                  <a
                    href={file.url}
                    download={file.filename}
                    onClick={e => e.stopPropagation()}
                    className="p-1.5 rounded-lg bg-black/40 border border-white/5 text-slate-400 hover:text-cyan-400 hover:border-cyan-500/20 transition"
                    title="Baixar arquivo de áudio"
                  >
                    <Download className="w-4 h-4" />
                  </a>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};
