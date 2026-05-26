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
    capabilities: ['text_chat'] as const,
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

  it('renders the expert marketplace hero, credit balance, and agent cards', async () => {
    renderDashboardPage();

    expect(await screen.findByText('Launch your AgentStore')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText((_, element) => element?.textContent === '36 credits available')).toBeInTheDocument();
    });
    expect(screen.getByText('Growth Strategist')).toBeInTheDocument();
    expect(screen.getByText('3 credits/message')).toBeInTheDocument();
    expect(screen.getByText('Text chat')).toBeInTheDocument();
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
    expect(screen.queryByText('Unable to load experts right now.')).not.toBeInTheDocument();
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

  it('shows an empty state when no experts are returned', async () => {
    apiMocks.listAgents.mockResolvedValue([]);

    renderDashboardPage();

    expect(await screen.findByText('No agents yet')).toBeInTheDocument();
    expect(screen.getByText('Check back soon or contact your administrator to publish agents.')).toBeInTheDocument();
  });
});
