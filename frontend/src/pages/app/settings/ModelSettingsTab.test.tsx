import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ModelSettingsTab from './ModelSettingsTab';

const apiMocks = vi.hoisted(() => ({
  listProviders: vi.fn(), createProvider: vi.fn(), updateProvider: vi.fn(), testProvider: vi.fn(),
  listModels: vi.fn(), createModel: vi.fn(), updateModel: vi.fn(), updateDefaults: vi.fn(),
}));
const tenantState = vi.hoisted(() => ({ activeTenant: { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'tenant-one', role: 'owner', isRoot: false } }));
vi.mock('../../../api/client', () => ({ tenantModelsApi: apiMocks }));
vi.mock('../../../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: tenantState.activeTenant,
    setActiveTenant: vi.fn(),
    isRootTenant: false,
    role: tenantState.activeTenant?.role ?? null,
  }),
}));

function renderModelSettingsTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}><ModelSettingsTab /></QueryClientProvider>);
}

describe('ModelSettingsTab', () => {
  beforeEach(() => {
    tenantState.activeTenant = { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'tenant-one', role: 'owner', isRoot: false };
    apiMocks.listProviders.mockResolvedValue([{ id: 'provider-1', tenantId: 'tenant-1', name: 'OpenAI', providerType: 'openai_compatible', baseUrl: 'https://api.example.com/v1', apiKeyPreview: 'sk-***1234', enabled: true, createdAt: '', updatedAt: '' }]);
    apiMocks.listModels.mockResolvedValue([]);
    apiMocks.testProvider.mockReset();
  });

  it('shows masked provider keys without raw secrets', async () => {
    renderModelSettingsTab();
    expect(await screen.findByText(/sk-\*\*\*1234/)).toBeInTheDocument();
    expect(screen.queryByText('sk-secret-1234')).not.toBeInTheDocument();
  });

  it('displays (coming soon) label for existing Anthropic provider', async () => {
    apiMocks.listProviders.mockResolvedValue([
      { id: 'provider-1', tenantId: 'tenant-1', name: 'Anthropic', providerType: 'anthropic', baseUrl: 'https://api.anthropic.com', apiKeyPreview: 'sk-ant-***', enabled: true, createdAt: '', updatedAt: '' },
    ]);
    renderModelSettingsTab();
    expect(await screen.findByText('Anthropic (coming soon)')).toBeInTheDocument();
  });

  it('displays (coming soon) label for existing Google Gemini provider', async () => {
    apiMocks.listProviders.mockResolvedValue([
      { id: 'provider-1', tenantId: 'tenant-1', name: 'Google Gemini', providerType: 'gemini', baseUrl: 'https://generativelanguage.googleapis.com', apiKeyPreview: 'AIza***', enabled: true, createdAt: '', updatedAt: '' },
    ]);
    renderModelSettingsTab();
    expect(await screen.findByText('Google Gemini (coming soon)')).toBeInTheDocument();
  });

  it('new provider form shows only OpenAI-compatible option', async () => {
    const user = userEvent.setup();
    renderModelSettingsTab();
    await user.click(screen.getByRole('button', { name: /new provider/i }));

    // Find the select element by its label text
    const typeSelect = document.querySelector('select');
    expect(typeSelect).toBeInTheDocument();

    // Should only have OpenAI Compatible option for P0
    const options = screen.getAllByRole('option');
    expect(options).toHaveLength(1);
    expect(options[0]).toHaveTextContent('OpenAI Compatible');
  });

  it('new model form shows text-only modality with coming soon message for image/video', async () => {
    const user = userEvent.setup();
    renderModelSettingsTab();
    await user.click(screen.getByRole('button', { name: /new model/i }));

    // Should show the coming soon message
    expect(screen.getByText(/Image and video models are coming soon/)).toBeInTheDocument();

    // Find the select element in the form
    const selects = document.querySelectorAll('select');
    const modalitySelect = selects[1]; // Second select is for modality
    expect(modalitySelect).toBeInTheDocument();
    expect(modalitySelect).toHaveValue('text');
  });

  it('displays provider test result message when provided by backend', async () => {
    const user = userEvent.setup();
    apiMocks.testProvider.mockResolvedValue({
      status: 'unsupported',
      message: 'Provider type not supported yet. OpenAI-compatible providers are supported. More provider types coming soon...'
    });
    renderModelSettingsTab();

    // Click test button on the provider
    const testButtons = await screen.findAllByTitle('Test Connection');
    await user.click(testButtons[0]);

    // Should show the message from backend
    expect(await screen.findByText(/Provider type not supported yet/)).toBeInTheDocument();
  });

  it('refetches provider and model settings after switching tenants instead of reusing previous tenant cache', async () => {
    apiMocks.listProviders
      .mockResolvedValueOnce([{ id: 'provider-1', tenantId: 'tenant-1', name: 'Tenant One Provider', providerType: 'openai_compatible', baseUrl: 'https://one.example.com/v1', apiKeyPreview: 'sk-***1111', enabled: true, createdAt: '', updatedAt: '' }])
      .mockResolvedValueOnce([{ id: 'provider-2', tenantId: 'tenant-2', name: 'Tenant Two Provider', providerType: 'anthropic', baseUrl: 'https://two.example.com/v1', apiKeyPreview: 'sk-***2222', enabled: true, createdAt: '', updatedAt: '' }]);
    apiMocks.listModels
      .mockResolvedValueOnce([{ id: 'model-1', tenantId: 'tenant-1', providerId: 'provider-1', name: 'tenant-one-model', displayName: 'Tenant One Model', modality: 'text', modelId: 'model-one', enabled: true, createdAt: '', updatedAt: '' }])
      .mockResolvedValueOnce([{ id: 'model-2', tenantId: 'tenant-2', providerId: 'provider-2', name: 'tenant-two-model', displayName: 'Tenant Two Model', modality: 'text', modelId: 'model-two', enabled: true, createdAt: '', updatedAt: '' }]);

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <ModelSettingsTab />
      </QueryClientProvider>
    );

    expect(await screen.findByText('Tenant One Provider')).toBeInTheDocument();
    expect(await screen.findByText('Tenant One Model')).toBeInTheDocument();
    const providerCallsBeforeSwitch = apiMocks.listProviders.mock.calls.length;
    const modelCallsBeforeSwitch = apiMocks.listModels.mock.calls.length;

    tenantState.activeTenant = { tenantId: 'tenant-2', tenantName: 'Tenant Two', tenantSlug: 'tenant-two', role: 'owner', isRoot: false };

    rerender(
      <QueryClientProvider client={queryClient}>
        <ModelSettingsTab />
      </QueryClientProvider>
    );

    expect(await screen.findByText('Tenant Two Provider (coming soon)')).toBeInTheDocument();
    expect(await screen.findByText('Tenant Two Model')).toBeInTheDocument();
    expect(screen.queryByText('Tenant One Provider')).not.toBeInTheDocument();
    expect(screen.queryByText('Tenant One Model')).not.toBeInTheDocument();
    await waitFor(() => expect(apiMocks.listProviders.mock.calls.length).toBeGreaterThan(providerCallsBeforeSwitch));
    await waitFor(() => expect(apiMocks.listModels.mock.calls.length).toBeGreaterThan(modelCallsBeforeSwitch));
  });
});
