import { useState, useEffect } from 'react';
import { Download, Check, AlertCircle, HardDrive, Globe, Info } from 'lucide-react';

interface AvailableModel {
  name: string;
  quality: string;
  description: string;
  size: string;
  filename: string;
}

interface ModelsPanelProps {
  currentModelPath: string;
  onSelectModel: (modelPath: string) => Promise<void>;
  apiBase: string;
}

export const ModelsPanel: React.FC<ModelsPanelProps> = ({ currentModelPath, onSelectModel, apiBase }) => {
  const [installedModels, setInstalledModels] = useState<string[]>([]);
  const [availableModels, setAvailableModels] = useState<AvailableModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [downloadingModel, setDownloadingModel] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchModels();
  }, []);

  const fetchModels = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${apiBase}/api/models`);
      if (res.ok) {
        const data = await res.json();
        setInstalledModels(data.installed || []);
        setAvailableModels(data.available || []);
      } else {
        setError('Falha ao obter lista de modelos do servidor.');
      }
    } catch (err) {
      console.error(err);
      setError('Erro de conexão ao buscar modelos.');
    } finally {
      setLoading(false);
    }
  };

  const handleDownload = async (model: AvailableModel) => {
    const modelId = `${model.name}-${model.quality}`;
    setDownloadingModel(modelId);
    setError(null);
    try {
      const res = await fetch(`${apiBase}/api/models/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: model.name,
          quality: model.quality,
        }),
      });

      if (res.ok) {
        alert(`Modelo ${model.filename} baixado e instalado com sucesso!`);
        await fetchModels(); // Atualiza lista
      } else {
        const txt = await res.text();
        setError(`Erro no download: ${txt}`);
      }
    } catch (err) {
      console.error(err);
      setError('Erro de conexão durante o download do modelo.');
    } finally {
      setDownloadingModel(null);
    }
  };

  const handleActivate = async (modelFilename: string) => {
    // Resolver caminho absoluto no mesmo diretório base
    // O backend salvará no diretório padrão ~/.local/share/myvoice/
    // Então passamos o caminho relativo ao appDir: filepath.Join(appDir, modelFilename)
    // O backend já cuida disso se passarmos apenas o filename ou o path completo.
    // Vamos passar o caminho resolvido na pasta ~/.local/share/myvoice/
    const baseSharePath = currentModelPath.substring(0, currentModelPath.lastIndexOf('/') + 1);
    const fullModelPath = `${baseSharePath}${modelFilename}`;
    
    try {
      await onSelectModel(fullModelPath);
      alert(`Modelo "${modelFilename}" ativado com sucesso!`);
    } catch (err) {
      alert('Erro ao ativar o modelo.');
    }
  };

  const isModelActive = (modelFilename: string) => {
    return currentModelPath.endsWith(modelFilename);
  };

  const isModelInstalled = (modelFilename: string) => {
    return installedModels.includes(modelFilename);
  };

  return (
    <div className="glass-panel p-6 h-full flex flex-col justify-between">
      <div className="space-y-6 flex-1 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between pb-3 border-b border-b-white/5">
          <h2 className="text-xl font-bold flex items-center gap-2 glow-text-primary">
            <Globe className="text-purple-400 w-5 h-5" />
            Biblioteca de Vozes ONNX (Piper Local)
          </h2>
          <button 
            onClick={fetchModels}
            className="text-xs text-cyan-400 hover:text-cyan-300 font-semibold"
            disabled={loading}
          >
            Atualizar
          </button>
        </div>

        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-xl text-xs text-red-400 flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 flex-1 items-start">
          {/* Coluna 1: Modelos Instalados */}
          <div className="space-y-4">
            <h3 className="text-sm font-bold text-slate-300 flex items-center gap-1.5 uppercase tracking-wider">
              <HardDrive className="w-4 h-4 text-purple-400" />
              Modelos Instalados Localmente
            </h3>
            
            {loading ? (
              <div className="text-center py-8 text-xs text-slate-500">Buscando modelos...</div>
            ) : installedModels.length === 0 ? (
              <div className="p-4 bg-black/20 border border-white/5 rounded-xl text-center text-xs text-slate-500">
                Nenhum modelo ONNX local encontrado.
              </div>
            ) : (
              <div className="space-y-2 overflow-y-auto max-h-[360px] pr-1">
                {installedModels.map(filename => {
                  const active = isModelActive(filename);
                  return (
                    <div 
                      key={filename}
                      className={`p-3 rounded-xl border flex justify-between items-center gap-2 ${
                        active 
                          ? 'bg-purple-600/15 border-purple-500/30' 
                          : 'bg-black/20 border-white/5 hover:bg-black/35'
                      }`}
                    >
                      <div className="min-w-0">
                        <div className="text-xs font-semibold text-slate-200 truncate">{filename}</div>
                        <div className="text-[10px] text-slate-500 font-mono mt-0.5 truncate">
                          ~/.local/share/myvoice/{filename}
                        </div>
                      </div>
                      
                      {active ? (
                        <span className="badge badge-green flex items-center gap-1">
                          <Check className="w-3 h-3" /> Ativo
                        </span>
                      ) : (
                        <button
                          onClick={() => handleActivate(filename)}
                          className="btn-secondary py-1 px-2.5 rounded-lg text-[10px] font-semibold"
                        >
                          Ativar
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
            
            {/* Info Box */}
            <div className="p-3 bg-black/30 border border-white/5 rounded-xl text-[11px] text-slate-400 flex gap-2">
              <Info className="w-4 h-4 text-cyan-400 flex-shrink-0 mt-0.5" />
              <div>
                <span className="font-semibold text-slate-300">Caminho dos Modelos:</span> O Piper busca os arquivos <code className="font-mono bg-black/50 px-1 rounded text-purple-300">.onnx</code> e <code className="font-mono bg-black/50 px-1 rounded text-purple-300">.json</code> na pasta local do sistema. Você pode arrastar novos modelos diretamente para o seu diretório share se preferir.
              </div>
            </div>
          </div>

          {/* Coluna 2: Biblioteca HF para baixar */}
          <div className="space-y-4">
            <h3 className="text-sm font-bold text-slate-300 flex items-center gap-1.5 uppercase tracking-wider">
              <Download className="w-4 h-4 text-cyan-400" />
              Baixar Novas Vozes pt-BR (Hugging Face)
            </h3>

            {loading ? (
              <div className="text-center py-8 text-xs text-slate-500">Buscando biblioteca...</div>
            ) : (
              <div className="space-y-2 overflow-y-auto max-h-[360px] pr-1">
                {availableModels.map(model => {
                  const installed = isModelInstalled(model.filename);
                  const active = isModelActive(model.filename);
                  const modelId = `${model.name}-${model.quality}`;
                  const downloading = downloadingModel === modelId;

                  return (
                    <div 
                      key={modelId}
                      className="p-3 bg-black/20 border border-white/5 rounded-xl hover:bg-black/35 flex justify-between items-center gap-3"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="text-xs font-semibold text-slate-200">{model.description}</span>
                          <span className="text-[10px] text-slate-500 font-mono">({model.size})</span>
                        </div>
                        <div className="text-[10px] text-slate-400 mt-1 truncate">
                          Arquivo: <code className="font-mono text-cyan-400/80">{model.filename}</code>
                        </div>
                      </div>

                      <div className="flex-shrink-0">
                        {downloading ? (
                          <div className="flex items-center gap-1.5 text-[10px] text-cyan-400 font-semibold bg-cyan-500/10 border border-cyan-500/20 px-2.5 py-1 rounded-lg">
                            <div className="w-2.5 h-2.5 border-2 border-cyan-400/30 border-t-cyan-400 rounded-full animate-spin" />
                            Baixando...
                          </div>
                        ) : active ? (
                          <span className="badge badge-green flex items-center gap-1">
                            <Check className="w-3 h-3" /> Ativo
                          </span>
                        ) : installed ? (
                          <button
                            onClick={() => handleActivate(model.filename)}
                            className="btn-secondary py-1 px-2.5 rounded-lg text-[10px] font-semibold text-purple-400 border-purple-500/20"
                          >
                            Ativar
                          </button>
                        ) : (
                          <button
                            onClick={() => handleDownload(model)}
                            className="btn-primary py-1 px-2.5 rounded-lg text-[10px] font-semibold flex items-center gap-1"
                          >
                            <Download className="w-3 h-3" /> Baixar
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
