import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Save, Loader2, Key, Globe, Cpu, ShieldAlert } from 'lucide-react';
import axios from 'axios';
import { adminApi } from '../../api/client';
import LoadingSpinner from '../../components/LoadingSpinner';
import { toast } from 'sonner';

interface LLMConfig {
  id?: string;
  apiKey: string;
  baseURL: string;
  model: string;
  isActive: boolean;
}

export default function LLMConfigPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<LLMConfig>({
    apiKey: '',
    baseURL: '',
    model: '',
    isActive: true,
  });
  const [validationErrors, setValidationErrors] = useState<Partial<Record<keyof Pick<LLMConfig, 'apiKey' | 'baseURL' | 'model'>, string>>>({});

  const { data: config, isLoading, error } = useQuery({
    queryKey: ['llm-config'],
    queryFn: () => adminApi.getLLMConfig(),
  });

  useEffect(() => {
    if (config) {
      setForm({
        apiKey: config.apiKey || '********',
        baseURL: config.baseURL || '',
        model: config.model || '',
        isActive: config.isActive ?? true,
      });
    }
  }, [config]);

  const updateMutation = useMutation({
    mutationFn: (data: LLMConfig) => adminApi.updateLLMConfig(data),
    onSuccess: () => {
      toast.success('Configuration saved');
      queryClient.invalidateQueries({ queryKey: ['llm-config'] });
    },
    onError: () => {
      toast.error('Failed to save configuration');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const nextErrors: Partial<Record<keyof Pick<LLMConfig, 'apiKey' | 'baseURL' | 'model'>, string>> = {};
    if (!form.apiKey.trim()) nextErrors.apiKey = 'API Key is required.';
    if (!form.baseURL.trim()) nextErrors.baseURL = 'Base URL is required.';
    if (!form.model.trim()) nextErrors.model = 'Model is required.';

    setValidationErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      return;
    }

    updateMutation.mutate({
      ...form,
      apiKey: form.apiKey.trim(),
      baseURL: form.baseURL.trim(),
      model: form.model.trim(),
    });
  };

  if (isLoading) {
    return <LoadingSpinner size="lg" className="py-20" />;
  }

  if (axios.isAxiosError(error) && error.response?.status === 403) {
    return (
      <div className="max-w-2xl mx-auto rounded-2xl border border-amber-500/25 bg-amber-500/10 p-6 text-amber-100">
        <div className="flex items-start gap-3">
          <ShieldAlert className="mt-0.5 h-6 w-6 flex-shrink-0" />
          <div>
            <h1 className="text-xl font-semibold text-white">Access denied</h1>
            <p className="mt-2 text-sm text-amber-100/90">
              You do not have permission to manage LLM configuration.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
          <Cpu className="w-7 h-7 text-primary-400" />
          LLM Configuration
        </h1>
        <p className="text-dark-400 mt-1">
          Configure the AI model provider for Expert Chat
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="bg-dark-900/50 border border-dark-800 rounded-2xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Key className="w-5 h-5" />
            API Configuration
          </h2>

          <div className="space-y-4">
            <div>
              <label htmlFor="apiKey" className="block text-sm font-medium text-dark-300 mb-2">
                API Key
              </label>
              <input
                id="apiKey"
                type="password"
                value={form.apiKey}
                onChange={(e) => {
                  setForm({ ...form, apiKey: e.target.value });
                  setValidationErrors((current) => ({ ...current, apiKey: undefined }));
                }}
                placeholder={config?.apiKey ? '******** (saved)' : 'Enter API key'}
                className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500"
              />
              {validationErrors.apiKey && <p className="mt-2 text-sm text-red-300">{validationErrors.apiKey}</p>}
            </div>

            <div>
              <label htmlFor="baseURL" className="block text-sm font-medium text-dark-300 mb-2 flex items-center gap-2">
                <Globe className="w-4 h-4" />
                Base URL
              </label>
              <input
                id="baseURL"
                type="text"
                value={form.baseURL}
                onChange={(e) => {
                  setForm({ ...form, baseURL: e.target.value });
                  setValidationErrors((current) => ({ ...current, baseURL: undefined }));
                }}
                placeholder="https://api.openai.com/v1"
                className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500"
              />
              {validationErrors.baseURL && <p className="mt-2 text-sm text-red-300">{validationErrors.baseURL}</p>}
              <p className="text-xs text-dark-500 mt-1">
                The base URL for the OpenAI-compatible API
              </p>
            </div>

            <div>
              <label htmlFor="model" className="block text-sm font-medium text-dark-300 mb-2">
                Model
              </label>
              <input
                id="model"
                type="text"
                value={form.model}
                onChange={(e) => {
                  setForm({ ...form, model: e.target.value });
                  setValidationErrors((current) => ({ ...current, model: undefined }));
                }}
                placeholder="gpt-4o-mini"
                className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500"
              />
              {validationErrors.model && <p className="mt-2 text-sm text-red-300">{validationErrors.model}</p>}
            </div>

            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="isActive"
                checked={form.isActive}
                onChange={(e) => setForm({ ...form, isActive: e.target.checked })}
                className="w-5 h-5 rounded border-dark-700 bg-dark-800 text-primary-500 focus:ring-primary-500"
              />
              <label htmlFor="isActive" className="text-dark-300">
                Enable AI Chat Feature
              </label>
            </div>
          </div>
        </div>

        <button
          type="submit"
          disabled={updateMutation.isPending}
          className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {updateMutation.isPending ? (
            <Loader2 className="w-5 h-5 animate-spin" />
          ) : (
            <Save className="w-5 h-5" />
          )}
          Save Configuration
        </button>
      </form>
    </div>
  );
}