import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, Send, X, Loader2 } from 'lucide-react';
import { tenantAgentsApi, tenantModelsApi } from '../../../api/client';
import { useTenant } from '../../../contexts/TenantContext';
import type { Agent, AgentCapability, AgentVisibility, AgentCreditCost } from '../../../types';

const emptyForm = {
  name: '',
  slug: '',
  category: '',
  description: '',
  systemPrompt: '',
  welcomeMessage: '',
  visibility: 'private' as AgentVisibility,
  capabilities: ['text_chat'] as AgentCapability[],
  creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 } as AgentCreditCost,
};

export default function AgentsTab() {
  const queryClient = useQueryClient();
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.tenantId ?? null;
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const { data: agents = [], isLoading } = useQuery({
    queryKey: ['tenant-agents', tenantId],
    queryFn: tenantAgentsApi.list,
  });

  useQuery({
    queryKey: ['model-configs', tenantId],
    queryFn: tenantModelsApi.listModels,
  });

  const createMutation = useMutation({
    mutationFn: tenantAgentsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-agents', tenantId] });
      resetForm();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Agent> }) => tenantAgentsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-agents', tenantId] });
      resetForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: tenantAgentsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-agents', tenantId] });
    },
  });

  const publishMutation = useMutation({
    mutationFn: tenantAgentsApi.publish,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-agents', tenantId] });
    },
  });

  const archiveMutation = useMutation({
    mutationFn: tenantAgentsApi.archive,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-agents', tenantId] });
    },
  });

  const resetForm = () => {
    setShowForm(false);
    setEditingId(null);
    setForm(emptyForm);
    setErrors({});
  };

  const handleEdit = (agent: Agent) => {
    setEditingId(agent.id);
    setForm({
      name: agent.name,
      slug: agent.slug,
      category: agent.category,
      description: agent.description,
      systemPrompt: agent.systemPrompt || '',
      welcomeMessage: agent.welcomeMessage || '',
      visibility: agent.visibility,
      capabilities: agent.capabilities || ['text_chat'],
      creditCost: agent.creditCost || { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
    });
    setShowForm(true);
  };

  const validate = () => {
    const errs: Record<string, string> = {};
    if (!form.name.trim()) errs.name = 'Name is required';
    if (!form.category.trim()) errs.category = 'Category is required';
    if (!form.description.trim()) errs.description = 'Description is required';
    if (!form.systemPrompt.trim()) errs.systemPrompt = 'System prompt is required';
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSave = () => {
    if (!validate()) return;

    const slug = form.slug || form.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');

    // For P0: only text_chat capability, image/video credits = 0
    const data = {
      ...form,
      slug,
      suggestedPrompts: form.welcomeMessage ? [form.welcomeMessage] : [],
      capabilities: ['text_chat'], // P0: only text chat
      creditCost: {
        ...form.creditCost,
        imageGenerationCredits: 0,
        videoGenerationCredits: 0,
      },
    };

    if (editingId) {
      updateMutation.mutate({ id: editingId, data });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this agent?')) {
      deleteMutation.mutate(id);
    }
  };

  const capabilityOptions: AgentCapability[] = ['text_chat'];
  const visibilityOptions: { value: AgentVisibility; label: string }[] = [
    { value: 'private', label: 'Private' },
    { value: 'public', label: 'Public' },
  ];

  // Get display label for capability - show "coming soon" for image/video
  const getCapabilityLabel = (cap: AgentCapability) => {
    if (cap === 'image_generation') return 'image generation (coming soon)';
    if (cap === 'video_generation') return 'video generation (coming soon)';
    return cap.replace('_', ' ');
  };

  // Check if capability is "coming soon"
  const isComingSoon = (cap: AgentCapability) => {
    return cap === 'image_generation' || cap === 'video_generation';
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-semibold">Agents</h2>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          {showForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
          {showForm ? 'Cancel' : 'New Agent'}
        </button>
      </div>

      {showForm && (
        <div className="border border-dark-700 rounded-lg p-4 mb-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input
                value={form.name}
                onChange={e => setForm({...form, name: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
                placeholder="My Agent"
              />
              {errors.name && <p className="text-red-400 text-xs mt-1">{errors.name}</p>}
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Slug</label>
              <input
                value={form.slug}
                onChange={e => setForm({...form, slug: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
                placeholder="my-agent (auto-generated if empty)"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Category *</label>
              <input
                value={form.category}
                onChange={e => setForm({...form, category: e.target.value})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
                placeholder="Marketing"
              />
              {errors.category && <p className="text-red-400 text-xs mt-1">{errors.category}</p>}
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Visibility</label>
              <select
                value={form.visibility}
                onChange={e => setForm({...form, visibility: e.target.value as AgentVisibility})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              >
                {visibilityOptions.map(opt => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">Description *</label>
            <textarea
              value={form.description}
              onChange={e => setForm({...form, description: e.target.value})}
              className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              rows={2}
              placeholder="What does this agent do?"
            />
            {errors.description && <p className="text-red-400 text-xs mt-1">{errors.description}</p>}
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">System Prompt *</label>
            <textarea
              value={form.systemPrompt}
              onChange={e => setForm({...form, systemPrompt: e.target.value})}
              className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              rows={4}
              placeholder="You are a helpful AI assistant..."
            />
            {errors.systemPrompt && <p className="text-red-400 text-xs mt-1">{errors.systemPrompt}</p>}
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">Welcome Message</label>
            <input
              value={form.welcomeMessage}
              onChange={e => setForm({...form, welcomeMessage: e.target.value})}
              className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              placeholder="How can I help you today?"
            />
          </div>

          <div>
            <label className="block text-sm text-dark-400 mb-1">Capabilities</label>
            <div className="flex gap-4">
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.capabilities.includes('text_chat')}
                  onChange={e => {
                    if (e.target.checked) {
                      setForm({...form, capabilities: [...form.capabilities, 'text_chat']});
                    } else {
                      setForm({...form, capabilities: form.capabilities.filter(c => c !== 'text_chat')});
                    }
                  }}
                  className="rounded"
                />
                Text chat
              </label>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Text Credits</label>
              <input
                type="number"
                min="0"
                value={form.creditCost.textMessageCredits}
                onChange={e => setForm({...form, creditCost: {...form.creditCost, textMessageCredits: parseInt(e.target.value) || 0}})}
                className="w-full bg-dark-800 border border-dark-700 rounded px-3 py-2"
              />
            </div>
            <p className="text-xs text-dark-500">Image and video generation are coming soon. P0 Agents use text chat only.</p>
          </div>

          <button
            onClick={handleSave}
            disabled={createMutation.isPending || updateMutation.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
          >
            {(createMutation.isPending || updateMutation.isPending) && <Loader2 className="w-4 h-4 animate-spin" />}
            {editingId ? 'Update Agent' : 'Create Agent'}
          </button>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="w-6 h-6 animate-spin text-primary-500" />
        </div>
      ) : agents.length === 0 ? (
        <div className="text-center py-8 text-dark-400">
          No agents yet. Create your first agent!
        </div>
      ) : (
        <div className="grid gap-4">
          {agents.map(agent => (
            <div key={agent.id} className="border border-dark-700 rounded-lg p-4">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="font-semibold text-white">{agent.name}</h3>
                  <p className="text-sm text-dark-400">{agent.category} • {agent.visibility}</p>
                  <p className="text-sm text-dark-300 mt-2">{agent.description}</p>
                  <div className="flex gap-2 mt-2">
                    {agent.capabilities?.map(cap => (
                      <span key={cap} className={`text-xs px-2 py-0.5 rounded ${isComingSoon(cap) ? 'bg-yellow-500/20 text-yellow-400' : 'bg-dark-800 rounded'}`}>
                        {getCapabilityLabel(cap)}
                      </span>
                    ))}
                    <span className="text-xs px-2 py-0.5 bg-primary-500/20 text-primary-400 rounded">
                      {agent.creditCost?.textMessageCredits || 1} credits
                    </span>
                  </div>
                </div>
                <div className="flex gap-2">
                  {agent.status === 'draft' && (
                    <button
                      onClick={() => publishMutation.mutate(agent.id)}
                      className="p-2 text-green-400 hover:bg-dark-800 rounded"
                      title="Publish"
                    >
                      <Send className="w-4 h-4" />
                    </button>
                  )}
                  {agent.status === 'published' && (
                    <button
                      onClick={() => archiveMutation.mutate(agent.id)}
                      className="p-2 text-yellow-400 hover:bg-dark-800 rounded"
                      title="Archive"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                  <button
                    onClick={() => handleEdit(agent)}
                    className="p-2 text-dark-400 hover:bg-dark-800 rounded"
                    title="Edit"
                  >
                    <Pencil className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(agent.id)}
                    className="p-2 text-red-400 hover:bg-dark-800 rounded"
                    title="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
              {agent.status && (
                <span className={`inline-block text-xs px-2 py-0.5 rounded mt-2 ${
                  agent.status === 'published' ? 'bg-green-500/20 text-green-400' :
                  agent.status === 'archived' ? 'bg-yellow-500/20 text-yellow-400' :
                  'bg-dark-700 text-dark-400'
                }`}>
                  {agent.status}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}