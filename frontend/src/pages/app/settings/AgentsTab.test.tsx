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

  it('new agent form shows only text chat capability with coming soon message for image/video', async () => {
    const user = userEvent.setup();
    renderAgentsTab();
    await user.click(screen.getByRole('button', { name: /new agent/i }));

    // Should show the coming soon message
    expect(screen.getByText(/Image and video generation are coming soon/)).toBeInTheDocument();
    expect(screen.getByText(/P0 Agents use text chat only/)).toBeInTheDocument();

    // Image and video credit inputs should be hidden or disabled
    const imageCreditsInput = screen.queryByLabelText(/Image Credits/i);
    const videoCreditsInput = screen.queryByLabelText(/Video Credits/i);
    // These inputs should not exist or should be hidden for P0
    expect(imageCreditsInput).not.toBeInTheDocument();
    expect(videoCreditsInput).not.toBeInTheDocument();
  });

  it('saves new agent with only text chat capability and zero image/video credit costs', async () => {
    const user = userEvent.setup();
    renderAgentsTab();
    await user.click(screen.getByRole('button', { name: /new agent/i }));

    // Fill in required fields using placeholder text
    await user.type(screen.getByPlaceholderText('My Agent'), 'Test Agent');
    await user.type(screen.getByPlaceholderText('Marketing'), 'Marketing');
    await user.type(screen.getByPlaceholderText('What does this agent do?'), 'A test agent');
    await user.type(screen.getByPlaceholderText('You are a helpful AI assistant...'), 'You are a helpful assistant');

    // Submit the form
    await user.click(screen.getByRole('button', { name: /create agent/i }));

    // Verify the API was called with text_chat only and zero for image/video
    await waitFor(() => {
      const call = apiMocks.create.mock.calls[0];
      expect(call[0]).toEqual(
        expect.objectContaining({
          capabilities: ['text_chat'],
          creditCost: {
            textMessageCredits: 1,
            imageGenerationCredits: 0,
            videoGenerationCredits: 0,
          },
        })
      );
    });
  });

  it('displays (coming soon) label for existing agents with image generation capability', async () => {
    apiMocks.list.mockResolvedValue([
      { id: 'agent-1', name: 'Image Agent', slug: 'image-agent', category: 'Marketing', description: 'Generates images', systemPrompt: 'Prompt', visibility: 'private', capabilities: ['text_chat', 'image_generation'], creditCost: { textMessageCredits: 1, imageGenerationCredits: 5, videoGenerationCredits: 0 }, createdAt: '', updatedAt: '' },
    ]);
    renderAgentsTab();

    expect(await screen.findByText('image generation (coming soon)')).toBeInTheDocument();
    expect(screen.queryByText('image_generation')).not.toBeInTheDocument();
  });

  it('displays (coming soon) label for existing agents with video generation capability', async () => {
    apiMocks.list.mockResolvedValue([
      { id: 'agent-1', name: 'Video Agent', slug: 'video-agent', category: 'Marketing', description: 'Generates videos', systemPrompt: 'Prompt', visibility: 'private', capabilities: ['text_chat', 'video_generation'], creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 10 }, createdAt: '', updatedAt: '' },
    ]);
    renderAgentsTab();

    expect(await screen.findByText('video generation (coming soon)')).toBeInTheDocument();
    expect(screen.queryByText('video_generation')).not.toBeInTheDocument();
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
