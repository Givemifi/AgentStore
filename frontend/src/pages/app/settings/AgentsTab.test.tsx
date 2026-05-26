import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AgentsTab from './AgentsTab';

const apiMocks = vi.hoisted(() => ({ list: vi.fn(), create: vi.fn(), update: vi.fn(), publish: vi.fn(), archive: vi.fn() }));
const tenantState = vi.hoisted(() => ({ activeTenant: { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'tenant-one', role: 'owner', isRoot: false } }));

vi.mock('../../../api/client', () => ({ tenantAgentsApi: apiMocks, tenantModelsApi: { listModels: vi.fn().mockResolvedValue([]) } }));
vi.mock('../../../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: tenantState.activeTenant,
    setActiveTenant: vi.fn(),
    isRootTenant: false,
    role: tenantState.activeTenant?.role ?? null,
  }),
}));

function renderAgentsTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}><AgentsTab /></QueryClientProvider>);
}

describe('AgentsTab', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    tenantState.activeTenant = { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'tenant-one', role: 'owner', isRoot: false };
    apiMocks.list.mockResolvedValue([]);
    apiMocks.create.mockResolvedValue({});
  });

  it('does not save without required fields', async () => {
    const user = userEvent.setup();
    renderAgentsTab();
    await screen.findByText(/No agents yet/i);
    await user.click(screen.getByRole('button', { name: /new agent/i }));
    await user.click(screen.getByRole('button', { name: /create agent/i }));
    expect(await screen.findByText(/Name is required/i)).toBeInTheDocument();
    expect(screen.getByText(/System prompt is required/i)).toBeInTheDocument();
    expect(apiMocks.create).not.toHaveBeenCalled();
  });

  it('refetches agents after switching tenants instead of reusing the previous tenant cache', async () => {
    apiMocks.list
      .mockResolvedValueOnce([{ id: 'agent-1', name: 'Tenant One Agent', slug: 'tenant-one-agent', category: 'Ops', description: 'First tenant agent', systemPrompt: 'Prompt', visibility: 'private', capabilities: ['text_chat'], creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 }, createdAt: '', updatedAt: '' }])
      .mockResolvedValueOnce([{ id: 'agent-2', name: 'Tenant Two Agent', slug: 'tenant-two-agent', category: 'Ops', description: 'Second tenant agent', systemPrompt: 'Prompt', visibility: 'private', capabilities: ['text_chat'], creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 }, createdAt: '', updatedAt: '' }]);

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <AgentsTab />
      </QueryClientProvider>
    );

    expect(await screen.findByText('Tenant One Agent')).toBeInTheDocument();
    expect(apiMocks.list).toHaveBeenCalledTimes(1);

    tenantState.activeTenant = { tenantId: 'tenant-2', tenantName: 'Tenant Two', tenantSlug: 'tenant-two', role: 'owner', isRoot: false };

    rerender(
      <QueryClientProvider client={queryClient}>
        <AgentsTab />
      </QueryClientProvider>
    );

    expect(await screen.findByText('Tenant Two Agent')).toBeInTheDocument();
    expect(screen.queryByText('Tenant One Agent')).not.toBeInTheDocument();
    await waitFor(() => expect(apiMocks.list).toHaveBeenCalledTimes(2));
  });
});
