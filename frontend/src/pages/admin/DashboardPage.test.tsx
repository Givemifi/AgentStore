import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AdminDashboardPage from './DashboardPage';

const apiMocks = vi.hoisted(() => ({
  getDashboard: vi.fn(),
  getHealthIntegrations: vi.fn(),
  getFinancialMetrics: vi.fn(),
  getLaunchReadiness: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  adminApi: {
    getDashboard: apiMocks.getDashboard,
    getHealthIntegrations: apiMocks.getHealthIntegrations,
    getFinancialMetrics: apiMocks.getFinancialMetrics,
    getLaunchReadiness: apiMocks.getLaunchReadiness,
  },
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Area: () => <div />,
  XAxis: () => <div />,
  YAxis: () => <div />,
  Tooltip: () => <div />,
}));

function renderDashboard() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AdminDashboardPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('AdminDashboardPage', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    localStorage.clear();
    apiMocks.getDashboard.mockResolvedValue({ users: 42, tenants: 7, health: { healthy: true, issues: [] } });
    apiMocks.getHealthIntegrations.mockResolvedValue({ integrations: [
      { name: 'stripe', status: 'not_configured', message: 'Missing key', lastCheck: '', responseMs: 0, calls24h: 0 },
      { name: 'resend', status: 'not_configured', message: 'Missing key', lastCheck: '', responseMs: 0, calls24h: 0 },
    ]});
    apiMocks.getFinancialMetrics.mockResolvedValue({ data: [{ date: '2026-06-06', value: 12345 }] });
    apiMocks.getLaunchReadiness.mockResolvedValue({
      items: [
        { id: 'brand', label: 'Brand configured', status: 'pending', description: 'Review app name and logo.', actionPath: '/admin/branding' },
        { id: 'model', label: 'Model provider connected', status: 'warning', description: 'Connect a provider.', actionPath: '/settings/models' },
        { id: 'agent', label: 'First Agent published', status: 'pending', description: 'Create and publish at least one Agent.', actionPath: '/settings/agents' },
        { id: 'credits', label: 'Credit bundle or plan active', status: 'warning', description: 'Enable a plan or credit bundle.', actionPath: '/admin/plans' },
        { id: 'stripe', label: 'Stripe webhook healthy', status: 'warning', description: 'Configure Stripe webhook handling.', actionPath: '/admin/health#integrations' },
        { id: 'resend', label: 'Email provider ready', status: 'warning', description: 'Configure Resend.', actionPath: '/admin/health#integrations' },
        { id: 'test-chat', label: 'Test chat passed', status: 'pending', description: 'Run a real smoke chat.', actionPath: '/dashboard' },
      ],
      summary: { complete: 0, warning: 4, pending: 3, total: 7 },
    });
  });

  it('renders summary metrics and launch checklist', async () => {
    renderDashboard();

    expect(await screen.findByText('Admin Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.getByText('Total Users')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('Tenants')).toBeInTheDocument();
    expect(screen.getByText('7')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Stripe webhook healthy/i })).toHaveAttribute('href', '/admin/health#integrations');
  });

  it('renders all 7 checklist items from API', async () => {
    renderDashboard();

    expect(await screen.findByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.getByText('Brand configured')).toBeInTheDocument();
    expect(screen.getByText('Model provider connected')).toBeInTheDocument();
    expect(screen.getByText('First Agent published')).toBeInTheDocument();
    expect(screen.getByText('Credit bundle or plan active')).toBeInTheDocument();
    expect(screen.getByText('Stripe webhook healthy')).toBeInTheDocument();
    expect(screen.getByText('Email provider ready')).toBeInTheDocument();
    expect(screen.getByText('Test chat passed')).toBeInTheDocument();
  });

  it('shows integration warning banner when integrations are not configured', async () => {
    renderDashboard();

    expect(await screen.findByText(/integrations? not configured/i)).toBeInTheDocument();
    expect(screen.getByText(/Stripe, Resend/)).toBeInTheDocument();
  });

  it('does not show integration warning banner when all integrations are configured', async () => {
    apiMocks.getHealthIntegrations.mockResolvedValue({ integrations: [
      { name: 'stripe', status: 'healthy', message: 'OK', lastCheck: '', responseMs: 50, calls24h: 5 },
      { name: 'resend', status: 'healthy', message: 'OK', lastCheck: '', responseMs: 30, calls24h: 2 },
    ]});
    renderDashboard();

    expect(await screen.findByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.queryByText(/integrations? not configured/i)).not.toBeInTheDocument();
  });

  it('renders Launch Checklist above integration warning banner in DOM', async () => {
    renderDashboard();

    expect(await screen.findByText('Launch Checklist')).toBeInTheDocument();

    const checklistSection = screen.getByText('Launch Checklist').closest('section');
    const warningBanner = screen.getByText(/integrations? not configured/i).closest('a');

    // Checklist should appear before warning in the DOM
    expect(checklistSection).not.toBeNull();
    expect(warningBanner).not.toBeNull();
    if (checklistSection && warningBanner) {
      const commonParent = checklistSection.parentElement;
      if (commonParent) {
        const allChildren = Array.from(commonParent.children);
        const checklistIdx = allChildren.indexOf(checklistSection);
        const warningIdx = allChildren.indexOf(warningBanner);
        expect(checklistIdx).toBeLessThan(warningIdx);
      }
    }
  });

  it('renders Launch Checklist above metrics cards in DOM', async () => {
    renderDashboard();

    expect(await screen.findByText('Launch Checklist')).toBeInTheDocument();

    const checklistSection = screen.getByText('Launch Checklist').closest('section');
    // Find the first metrics card - Total Users card is inside a Link
    const metricsCard = screen.getByText('Total Users').closest('a');

    expect(checklistSection).not.toBeNull();
    expect(metricsCard).not.toBeNull();
    if (checklistSection && metricsCard) {
      const commonParent = checklistSection.parentElement;
      if (commonParent) {
        const allChildren = Array.from(commonParent.children);
        const checklistIdx = allChildren.indexOf(checklistSection);
        // Find the parent card that contains the Total Users link
        const metricsCardParent = metricsCard.closest('[class*="grid"]') || metricsCard.parentElement;
        if (metricsCardParent) {
          const metricsIdx = allChildren.indexOf(metricsCardParent as Element);
          if (metricsIdx >= 0) {
            expect(checklistIdx).toBeLessThan(metricsIdx);
            return;
          }
        }
        // Fallback: check that at least the checklist comes before Total Users text
        const checklistPos = checklistSection.compareDocumentPosition(metricsCard);
        expect(checklistPos & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
      }
    }
  });

  it('renders link href for First Agent published checklist item', async () => {
    renderDashboard();

    expect(await screen.findByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /First Agent published/i })).toHaveAttribute('href', '/settings/agents');
  });

  it('changes chart range and refetches financial metrics', async () => {
    const user = userEvent.setup();
    renderDashboard();

    await screen.findByText('Business Metrics');
    await user.click(screen.getByRole('button', { name: '7d' }));

    expect(apiMocks.getFinancialMetrics).toHaveBeenCalledWith({ range: '7d', metric: 'revenue' });
    expect(apiMocks.getFinancialMetrics).toHaveBeenCalledWith({ range: '7d', metric: 'arr' });
    expect(apiMocks.getFinancialMetrics).toHaveBeenCalledWith({ range: '7d', metric: 'dau' });
  });

  it('shows singular "integration not configured" when only one is unconfigured', async () => {
    apiMocks.getHealthIntegrations.mockResolvedValue({ integrations: [
      { name: 'resend', status: 'not_configured', message: 'Missing key', lastCheck: '', responseMs: 0, calls24h: 0 },
    ]});
    renderDashboard();

    expect(await screen.findByText('1 integration not configured')).toBeInTheDocument();
  });

  it('shows plural "integrations not configured" when multiple are unconfigured', async () => {
    renderDashboard();

    expect(await screen.findByText('2 integrations not configured')).toBeInTheDocument();
  });

  it('renders launch checklist from readiness API instead of local storage', async () => {
    localStorage.setItem('launch-checklist-brand', 'complete');
    apiMocks.getLaunchReadiness.mockResolvedValue({
      items: [
        { id: 'brand', label: 'Brand configured', status: 'pending', description: 'Backend says brand is still pending.', actionPath: '/admin/branding' },
      ],
      summary: { complete: 0, warning: 0, pending: 1, total: 1 },
    });

    renderDashboard();

    expect(await screen.findByText('Backend says brand is still pending.')).toBeInTheDocument();
    const brandLink = screen.getByRole('link', { name: /Brand configured/i });
    expect(within(brandLink).getByText('Review')).toBeInTheDocument();
  });

  it('shows a visible warning when launch readiness cannot be loaded', async () => {
    apiMocks.getLaunchReadiness.mockRejectedValue(new Error('readiness unavailable'));

    renderDashboard();

    expect(await screen.findByText('Launch readiness unavailable')).toBeInTheDocument();
    expect(screen.getByText('Refresh the page or check admin API health before using the checklist for launch decisions.')).toBeInTheDocument();
  });

  it('shows loading state while fetching launch readiness', async () => {
    // Make the readiness query hang by not resolving
    let resolveReadiness: (value: unknown) => void;
    apiMocks.getLaunchReadiness.mockImplementation(() => new Promise((resolve) => {
      resolveReadiness = resolve;
    }));

    renderDashboard();

    // Dashboard loads but readiness is still loading
    expect(await screen.findByText('Admin Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Loading launch readiness...')).toBeInTheDocument();
  });
});