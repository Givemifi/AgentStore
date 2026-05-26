import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, X, Loader2, Plug, Check, AlertCircle } from 'lucide-react';
import { tenantModelsApi } from '../../../api/client';
import { useTenant } from '../../../contexts/TenantContext';
import type { ModelProvider, ModelConfig, ProviderType, ModelModality } from '../../../types';

const emptyProviderForm = {
  name: '',
  providerType: 'openai_compatible' as ProviderType,
  baseUrl: '',
  apiKey: '',
  enabled: true,
};

const emptyModelForm = {
  providerId: '',
  name: '',
  displayName: '',
  modality: 'text' as ModelModality,
  modelId: '',
  enabled: true,
};

export default function ModelSettingsTab() {
  const queryClient = useQueryClient();
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.tenantId ?? null;
  const [showProviderForm, setShowProviderForm] = useState(false);
  const [showModelForm, setShowModelForm] = useState(false);
  const [editingProviderId, setEditingProviderId] = useState<string | null>(null);
  const [editingModelId, setEditingModelId] = useState<string | null>(null);
  const [providerForm, setProviderForm] = useState(emptyProviderForm);
  const [modelForm, setModelForm] = useState(emptyModelForm);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [testingProviderId, setTestingProviderId] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  const { data: providers = [], isLoading: loadingProviders } = useQuery({
    queryKey: ['model-providers', tenantId],
    queryFn: tenantModelsApi.listProviders,
  });

  const { data: models = [] } = useQuery({
    queryKey: ['model-configs', tenantId],
    queryFn: tenantModelsApi.listModels,
  });

  const createProviderMutation = useMutation({
    mutationFn: tenantModelsApi.createProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-providers', tenantId] });
      resetProviderForm();
    },
  });

  const updateProviderMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ModelProvider> }) =>
      tenantModelsApi.updateProvider(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-providers', tenantId] });
      resetProviderForm();
    },
  });

  const deleteProviderMutation = useMutation({
    mutationFn: tenantModelsApi.deleteProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-providers', tenantId] });
      queryClient.invalidateQueries({ queryKey: ['model-configs', tenantId] });
    },
  });

  const createModelMutation = useMutation({
    mutationFn: tenantModelsApi.createModel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-configs', tenantId] });
      resetModelForm();
    },
  });

  const updateModelMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ModelConfig> }) =>
      tenantModelsApi.updateModel(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-configs', tenantId] });
      resetModelForm();
    },
  });

  const deleteModelMutation = useMutation({
    mutationFn: tenantModelsApi.deleteModel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-configs', tenantId] });
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: (providerId: string) => tenantModelsApi.testProvider(providerId),
    onSuccess: (result) => {
      setTestResult({ success: result.status === 'ok', message: result.status });
      setTimeout(() => setTestResult(null), 5000);
    },
    onError: (err: Error) => {
      setTestResult({ success: false, message: err.message });
      setTimeout(() => setTestResult(null), 5000);
    },
    onSettled: () => setTestingProviderId(null),
  });

  const resetProviderForm = () => {
    setShowProviderForm(false);
    setEditingProviderId(null);
    setProviderForm(emptyProviderForm);
    setErrors({});
  };

  const resetModelForm = () => {
    setShowModelForm(false);
    setEditingModelId(null);
    setModelForm(emptyModelForm);
    setErrors({});
  };

  const handleEditProvider = (provider: ModelProvider) => {
    setEditingProviderId(provider.id);
    setProviderForm({
      name: provider.name,
      providerType: provider.providerType,
      baseUrl: provider.baseUrl,
      apiKey: '',
      enabled: provider.enabled,
    });
    setShowProviderForm(true);
  };

  const handleEditModel = (model: ModelConfig) => {
    setEditingModelId(model.id);
    setModelForm({
      providerId: model.providerId,
      name: model.name,
      displayName: model.displayName,
      modality: model.modality,
      modelId: model.modelId,
      enabled: model.enabled,
    });
    setShowModelForm(true);
  };

  const validateProvider = () => {
    const errs: Record<string, string> = {};
    if (!providerForm.name.trim()) errs.name = 'Name is required';
    if (!providerForm.baseUrl.trim()) errs.baseUrl = 'Base URL is required';
    if (!providerForm.apiKey.trim() && !editingProviderId) errs.apiKey = 'API Key is required';
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const validateModel = () => {
    const errs: Record<string, string> = {};
    if (!modelForm.providerId) errs.providerId = 'Provider is required';
    if (!modelForm.name.trim()) errs.name = 'Name is required';
    if (!modelForm.modelId.trim()) errs.modelId = 'Model ID is required';
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSaveProvider = () => {
    if (!validateProvider()) return;

    const data = {
      name: providerForm.name,
      providerType: providerForm.providerType,
      baseUrl: providerForm.baseUrl,
      apiKey: providerForm.apiKey,
      enabled: providerForm.enabled,
    };

    if (editingProviderId) {
      updateProviderMutation.mutate({ id: editingProviderId, data });
    } else {
      createProviderMutation.mutate(data);
    }
  };

  const handleSaveModel = () => {
    if (!validateModel()) return;

    const data = {
      providerId: modelForm.providerId,
      name: modelForm.name,
      displayName: modelForm.displayName || modelForm.name,
      modality: modelForm.modality,
      modelId: modelForm.modelId,
      enabled: modelForm.enabled,
    };

    if (editingModelId) {
      updateModelMutation.mutate({ id: editingModelId, data });
    } else {
      createModelMutation.mutate(data);
    }
  };

  const handleDeleteProvider = (id: string) => {
    if (confirm('Are you sure you want to delete this provider? All associated models will also be deleted.')) {
      deleteProviderMutation.mutate(id);
    }
  };

  const handleDeleteModel = (id: string) => {
    if (confirm('Are you sure you want to delete this model?')) {
      deleteModelMutation.mutate(id);
    }
  };

  const handleTestConnection = (providerId: string) => {
    setTestingProviderId(providerId);
    setTestResult(null);
    testConnectionMutation.mutate(providerId);
  };

  const providerTypeOptions: { value: ProviderType; label: string }[] = [
    { value: 'openai_compatible', label: 'OpenAI Compatible' },
    { value: 'anthropic', label: 'Anthropic' },
    { value: 'gemini', label: 'Google Gemini' },
  ];

  const modalityOptions: { value: ModelModality; label: string }[] = [
    { value: 'text', label: 'Text' },
    { value: 'image', label: 'Image' },
    { value: 'video', label: 'Video' },
  ];

  const getModelsByProvider = (providerId: string) =>
    models.filter(m => m.providerId === providerId);

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-semibold">Model Settings</h2>
        <div className="flex gap-2">
          <button
            onClick={() => setShowModelForm(!showModelForm)}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
          >
            {showModelForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            {showModelForm ? 'Cancel' : 'New Model'}
          </button>
          <button
            onClick={() => setShowProviderForm(!showProviderForm)}
            className="flex items-center gap-2 px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600"
          >
            {showProviderForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            {showProviderForm ? 'Cancel' : 'New Provider'}
          </button>
        </div>
      </div>

      {/* Provider Form */}
      {showProviderForm && (
        <div className="border border-dark-700 rounded-lg p-4 mb-6 space-y-4">
          <h3 className="font-medium text-white">{editingProviderId ? 'Edit Provider' : 'New Provider'}</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input
                value={providerForm.name}
                onChange={e => setProviderForm({...providerForm, name: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
                placeholder="My Provider"
              />
              {errors.name && <p className="text-red-400 text-xs mt-1">{errors.name}</p>}
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Type</label>
              <select
                value={providerForm.providerType}
                onChange={e => setProviderForm({...providerForm, providerType: e.target.value as ProviderType})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              >
                {providerTypeOptions.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">Base URL *</label>
            <input
              value={providerForm.baseUrl}
              onChange={e => setProviderForm({...providerForm, baseUrl: e.target.value})}
              className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              placeholder="https://api.openai.com/v1"
            />
            {errors.baseUrl && <p className="text-red-400 text-xs mt-1">{errors.baseUrl}</p>}
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">API Key *</label>
            <input
              type="password"
              value={providerForm.apiKey}
              onChange={e => setProviderForm({...providerForm, apiKey: e.target.value})}
              className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              placeholder={editingProviderId ? '(unchanged)' : 'sk-...'}
            />
            {errors.apiKey && <p className="text-red-400 text-xs mt-1">{errors.apiKey}</p>}
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="providerEnabled"
              checked={providerForm.enabled}
              onChange={e => setProviderForm({...providerForm, enabled: e.target.checked})}
              className="rounded"
            />
            <label htmlFor="providerEnabled" className="text-sm text-dark-300">Enabled</label>
          </div>

          <button
            onClick={handleSaveProvider}
            disabled={createProviderMutation.isPending || updateProviderMutation.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
          >
            {(createProviderMutation.isPending || updateProviderMutation.isPending) &&
              <Loader2 className="w-4 h-4 animate-spin" />}
            {editingProviderId ? 'Update Provider' : 'Create Provider'}
          </button>
        </div>
      )}

      {/* Model Form */}
      {showModelForm && (
        <div className="border border-dark-700 rounded-lg p-4 mb-6 space-y-4">
          <h3 className="font-medium text-white">{editingModelId ? 'Edit Model' : 'New Model'}</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Provider *</label>
              <select
                value={modelForm.providerId}
                onChange={e => setModelForm({...modelForm, providerId: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              >
                <option value="">Select provider...</option>
                {providers.filter(p => p.enabled).map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
              {errors.providerId && <p className="text-red-400 text-xs mt-1">{errors.providerId}</p>}
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Modality</label>
              <select
                value={modelForm.modality}
                onChange={e => setModelForm({...modelForm, modality: e.target.value as ModelModality})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              >
                {modalityOptions.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input
                value={modelForm.name}
                onChange={e => setModelForm({...modelForm, name: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
                placeholder="gpt-4o"
              />
              {errors.name && <p className="text-red-400 text-xs mt-1">{errors.name}</p>}
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Display Name</label>
              <input
                value={modelForm.displayName}
                onChange={e => setModelForm({...modelForm, displayName: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
                placeholder="GPT-4o"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">Model ID *</label>
            <input
              value={modelForm.modelId}
              onChange={e => setModelForm({...modelForm, modelId: e.target.value})}
              className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              placeholder="gpt-4o"
            />
            {errors.modelId && <p className="text-red-400 text-xs mt-1">{errors.modelId}</p>}
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="modelEnabled"
              checked={modelForm.enabled}
              onChange={e => setModelForm({...modelForm, enabled: e.target.checked})}
              className="rounded"
            />
            <label htmlFor="modelEnabled" className="text-sm text-dark-300">Enabled</label>
          </div>

          <button
            onClick={handleSaveModel}
            disabled={createModelMutation.isPending || updateModelMutation.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
          >
            {(createModelMutation.isPending || updateModelMutation.isPending) &&
              <Loader2 className="w-4 h-4 animate-spin" />}
            {editingModelId ? 'Update Model' : 'Create Model'}
          </button>
        </div>
      )}

      {/* Test Result Toast */}
      {testResult && (
        <div className={`mb-4 p-3 rounded-lg flex items-center gap-2 ${
          testResult.success ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
        }`}>
          {testResult.success ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {testResult.message}
        </div>
      )}

      {/* Providers List */}
      {loadingProviders ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="w-6 h-6 animate-spin text-primary-500" />
        </div>
      ) : providers.length === 0 ? (
        <div className="text-center py-8 text-dark-400">
          No providers configured. Add a provider to get started.
        </div>
      ) : (
        <div className="space-y-6">
          {providers.map(provider => (
            <div key={provider.id} className="border border-dark-700 rounded-lg p-4">
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="font-semibold text-white flex items-center gap-2">
                    {provider.name}
                    <span className={`text-xs px-2 py-0.5 rounded ${
                      provider.enabled ? 'bg-green-500/20 text-green-400' : 'bg-dark-700 text-dark-400'
                    }`}>
                      {provider.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </h3>
                  <p className="text-sm text-dark-400">
                    {provider.providerType} • {provider.baseUrl}
                    {provider.apiKeyPreview && <> • {provider.apiKeyPreview}</>}
                  </p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleTestConnection(provider.id)}
                    disabled={testingProviderId === provider.id}
                    className="p-2 text-dark-400 hover:bg-dark-800 rounded disabled:opacity-50"
                    title="Test Connection"
                  >
                    {testingProviderId === provider.id ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Plug className="w-4 h-4" />
                    )}
                  </button>
                  <button
                    onClick={() => handleEditProvider(provider)}
                    className="p-2 text-dark-400 hover:bg-dark-800 rounded"
                    title="Edit"
                  >
                    <Pencil className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDeleteProvider(provider.id)}
                    className="p-2 text-red-400 hover:bg-dark-800 rounded"
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Models for this provider */}
              <div className="ml-4 border-l border-dark-700 pl-4">
                <h4 className="text-sm text-dark-400 mb-2">Models</h4>
                {getModelsByProvider(provider.id).length === 0 ? (
                  <p className="text-sm text-dark-500">No models configured</p>
                ) : (
                  <div className="grid gap-2">
                    {getModelsByProvider(provider.id).map(model => (
                      <div key={model.id} className="flex justify-between items-center bg-dark-800 rounded px-3 py-2">
                        <div>
                          <span className="text-sm text-white">{model.displayName || model.name}</span>
                          <span className="text-xs text-dark-400 ml-2">({model.modelId})</span>
                          <span className={`text-xs px-2 py-0.5 ml-2 rounded ${
                            model.enabled ? 'bg-green-500/20 text-green-400' : 'bg-dark-700 text-dark-400'
                          }`}>
                            {model.modality}
                          </span>
                        </div>
                        <div className="flex gap-1">
                          <button
                            onClick={() => handleEditModel(model)}
                            className="p-1 text-dark-400 hover:bg-dark-700 rounded"
                            title="Edit"
                          >
                            <Pencil className="w-3 h-3" />
                          </button>
                          <button
                            onClick={() => handleDeleteModel(model.id)}
                            className="p-1 text-red-400 hover:bg-dark-700 rounded"
                            title="Delete"
                          >
                            <Trash2 className="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}