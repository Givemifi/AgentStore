import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DashboardPage from './DashboardPage';

const mockAgents = [
  {
    id: 'agent-1',
    name: 'Growth Strategist',
    slug: 'growth-strategist',
    category: 'Marketing',
    description: 'Plans launch and growth work',
    avatar: '',
    icon: 'rocket',
    color: '#7C3AED',
    visibility: 'private' as const,
    welcomeMessage: 'Tell me about your launch.',
    suggestedPrompts: ['Draft a launch plan', 'Review my pricing page'],
    capabilities: ['text_chat'] as readonly string[],
    creditCost: { textMessageCredits: 3, imageGenerationCredits: 0, videoGenerationCredits: 0 },
    createdAt: '2026-05-21T12:00:00.000Z',
    updatedAt: '2026-05-21T12:00:00.000Z',
  },
];

const apiMocks = vi.hoisted(() => ({
  listAgents: vi.fn(),
  usageSummary: vi.fn(),
  getBranding: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  agentsApi: {
    list: apiMocks.listAgents,
  },
  usageApi: {
    summary: apiMocks.usageSummary,
  },
  brandingApi: {
    get: apiMocks.getBranding,
  },
}));

vi.mock('../../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: { tenantId: 'tenant-1', tenantName: 'Test Tenant', tenantSlug: 'test', role: 'owner', isRoot: false },
    setActiveTenant: vi.fn(),
    isRootTenant: false,
    role: 'owner',
  }),
}));

vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
  },
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderDashboardPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route
            path="/dashboard"
            element={
              <>
                <DashboardPage />
                <LocationDisplay />
              </>
            }
          />
          <Route path="/chat/:agentId" element={<LocationDisplay />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('DashboardPage', () => {
  beforeEach(() => {
    apiMocks.listAgents.mockReset();
    apiMocks.usageSummary.mockReset();
    apiMocks.getBranding.mockReset();

    apiMocks.listAgents.mockResolvedValue(mockAgents);
    apiMocks.usageSummary.mockResolvedValue({
      periodStart: '2026-05-01T00:00:00.000Z',
      usage: [],
      totalCreditsUsed: 0,
      subscriptionCredits: 24,
      purchasedCredits: 12,
    });
    apiMocks.getBranding.mockResolvedValue({ dashboardHtml: '' });
  });

  it('renders the agent marketplace hero, credit safety copy, search, categories, and agent cards', async () => {
    renderDashboardPage();

    expect(await screen.findByText('Agent Marketplace')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText(/36/)).toBeInTheDocument();
    });
    expect(screen.getByText('Credits are charged only after a successful response.')).toBeInTheDocument();
    expect(screen.getByRole('searchbox', { name: 'Search Agents' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Marketing' })).toBeInTheDocument();
    expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.getByText('3 credits/message')).toBeInTheDocument();
  });

  it('shows the Recommended category chip', async () => {
    renderDashboardPage();

    expect(await screen.findByRole('button', { name: 'Recommended' })).toBeInTheDocument();
  });

  it('shows "No Agents match your search" with Clear filters action when all agents are filtered out', async () => {
    const user = userEvent.setup();
    renderDashboardPage();

    expect(await screen.findByText('Growth Strategist')).toBeInTheDocument();

    await user.type(screen.getByRole('searchbox', { name: 'Search Agents' }), 'xyznonexistent');

    expect(await screen.findByText('No Agents match your search')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Clear filters' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    await waitFor(() => {
      expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    });
    expect(screen.queryByText('No Agents match your search')).not.toBeInTheDocument();
  });

  it('shows "No agents yet" empty state when there are zero agents', async () => {
    apiMocks.listAgents.mockResolvedValue([]);

    renderDashboardPage();

    expect(await screen.findByText('No agents yet')).toBeInTheDocument();
    expect(screen.getByText(/This workspace has not published any Agents yet/)).toBeInTheDocument();
  });

  it('marks selected category button with aria-pressed true and others with false', async () => {
    apiMocks.listAgents.mockResolvedValue([
      ...mockAgents,
      {
        ...mockAgents[0],
        id: 'agent-2',
        name: 'Legal Reviewer',
        slug: 'legal-reviewer',
        category: 'Legal',
        description: 'Reviews contracts',
        capabilities: ['text_chat'] as readonly string[],
        creditCost: { textMessageCredits: 2, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '2026-05-21T12:00:00.000Z',
        updatedAt: '2026-05-21T12:00:00.000Z',
      },
    ]);

    renderDashboardPage();

    const recommendedBtn = await screen.findByRole('button', { name: 'Recommended' });
    expect(recommendedBtn).toHaveAttribute('aria-pressed', 'true');

    const marketingBtn = screen.getByRole('button', { name: 'Marketing' });
    expect(marketingBtn).toHaveAttribute('aria-pressed', 'false');

    const user = userEvent.setup();
    await user.click(marketingBtn);
    expect(marketingBtn).toHaveAttribute('aria-pressed', 'true');
    expect(recommendedBtn).toHaveAttribute('aria-pressed', 'false');
  });

  it('renders custom dashboard HTML section with accessible role and label', async () => {
    apiMocks.getBranding.mockResolvedValue({
      dashboardHtml: '<h2>Custom Dashboard</h2>',
    });

    renderDashboardPage();

    const section = await screen.findByRole('region', { name: 'Custom dashboard content' });
    expect(section).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Custom Dashboard' })).toBeInTheDocument();
  });

  it('shows skeleton loading state before agents are loaded', async () => {
    let resolveAgents: (value: unknown) => void = () => {};
    apiMocks.listAgents.mockReturnValue(new Promise((resolve) => { resolveAgents = resolve; }));

    renderDashboardPage();

    // Skeleton blocks use animate-pulse; verify they exist while loading
    const skeletonBlocks = document.querySelectorAll('.animate-pulse');
    expect(skeletonBlocks.length).toBeGreaterThan(0);

    // Resolve to let cleanup proceed
    resolveAgents(mockAgents);
    await waitFor(() => {
      expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    });
  });

  it('filters agents by search text and category chip', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([
      ...mockAgents,
      {
        ...mockAgents[0],
        id: 'agent-2',
        name: 'Legal Reviewer',
        slug: 'legal-reviewer',
        category: 'Legal',
        description: 'Reviews contracts and clauses',
        suggestedPrompts: ['Review this contract'],
        capabilities: ['text_chat'] as readonly string[],
        creditCost: { textMessageCredits: 2, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '2026-05-21T12:00:00.000Z',
        updatedAt: '2026-05-21T12:00:00.000Z',
      },
    ]);

    renderDashboardPage();

    expect(await screen.findByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.getByText('Legal Reviewer')).toBeInTheDocument();

    await user.type(screen.getByRole('searchbox', { name: 'Search Agents' }), 'legal');
    expect(screen.queryByText('Growth Strategist')).not.toBeInTheDocument();
    expect(screen.getByText('Legal Reviewer')).toBeInTheDocument();

    await user.clear(screen.getByRole('searchbox', { name: 'Search Agents' }));
    await user.click(screen.getByRole('button', { name: 'Marketing' }));
    expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.queryByText('Legal Reviewer')).not.toBeInTheDocument();
  });

  it('starts chat using the agent slug', async () => {
    const user = userEvent.setup();
    renderDashboardPage();

    await user.click(await screen.findByRole('button', { name: 'Start with Growth Strategist' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/growth-strategist');
  });

  it('starts chat with a selected suggested prompt in the query string', async () => {
    const user = userEvent.setup();
    renderDashboardPage();

    await user.click(await screen.findByRole('button', { name: 'Try prompt: Draft a launch plan' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/growth-strategist?prompt=Draft+a+launch+plan');
  });

  it('hides ordinary-user image and video labels even when backend capabilities include them', async () => {
    apiMocks.listAgents.mockResolvedValue([
      {
        ...mockAgents[0],
        capabilities: ['text_chat', 'image_generation', 'video_generation'] as readonly string[],
      },
    ]);

    renderDashboardPage();

    expect(await screen.findByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.queryByText('Image')).not.toBeInTheDocument();
    expect(screen.queryByText('Video')).not.toBeInTheDocument();
  });

  it('shows a persistent inline error with retry when experts fail to load', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents
      .mockRejectedValueOnce(new Error('Service unavailable'))
      .mockResolvedValueOnce(mockAgents);

    renderDashboardPage();

    expect(await screen.findByText('Unable to load agents right now.')).toBeInTheDocument();
    expect(screen.getByText('Service unavailable')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => {
      expect(apiMocks.listAgents).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.queryByText('Unable to load agents right now.')).not.toBeInTheDocument();
  });

  it('renders sanitized custom dashboard HTML from branding', async () => {
    apiMocks.getBranding.mockResolvedValue({
      dashboardHtml: '<section><h2>Custom Dashboard</h2><img src="x" onerror="window.__dashboardXss = true"><script>window.__dashboardScript = true</script></section>',
    });

    renderDashboardPage();

    expect(await screen.findByRole('heading', { name: 'Custom Dashboard' })).toBeInTheDocument();
    expect(document.querySelector('script')).not.toBeInTheDocument();
    expect(document.querySelector('[onerror]')).not.toBeInTheDocument();
    expect((window as typeof window & { __dashboardXss?: boolean; __dashboardScript?: boolean }).__dashboardXss).toBeUndefined();
    expect((window as typeof window & { __dashboardXss?: boolean; __dashboardScript?: boolean }).__dashboardScript).toBeUndefined();
  });
});
