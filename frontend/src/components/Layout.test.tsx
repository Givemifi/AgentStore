import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Layout from './Layout';
import type { NavItem } from '../types';

const apiMocks = vi.hoisted(() => ({
  unreadCount: vi.fn(),
  listPlans: vi.fn(),
  listBundles: vi.fn(),
  listAnnouncements: vi.fn(),
  logout: vi.fn(),
  setActiveTenant: vi.fn(),
}));

const authState = vi.hoisted(() => ({
  memberships: [
    { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false },
  ],
}));

const tenantState = vi.hoisted(() => ({
  activeTenant: { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false },
}));

const brandingState = vi.hoisted(() => ({
  navItems: [] as NavItem[],
}));

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { displayName: 'Ada Lovelace' },
    isAuthenticated: true,
    logout: apiMocks.logout,
    memberships: authState.memberships,
  }),
}));

vi.mock('../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: tenantState.activeTenant,
    setActiveTenant: apiMocks.setActiveTenant,
  }),
}));

vi.mock('../contexts/BrandingContext', () => ({
  useBranding: () => ({
    branding: {
      appName: 'AgentStore',
      logoMode: 'text',
      logoUrl: '',
      navItems: brandingState.navItems,
    },
  }),
}));

vi.mock('../contexts/ThemeContext', () => ({
  useTheme: () => ({
    resolvedTheme: 'dark',
    setTheme: vi.fn(),
  }),
}));

vi.mock('../api/client', () => ({
  messagesApi: { unreadCount: apiMocks.unreadCount },
  plansApi: { list: apiMocks.listPlans },
  bundlesApi: { list: apiMocks.listBundles },
  announcementsApi: { list: apiMocks.listAnnouncements },
}));

vi.mock('./ImpersonationBanner', () => ({ default: () => null }));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route path="dashboard" element={<><div>Dashboard content</div><LocationDisplay /></>} />
        <Route path="buy-credits" element={<><div>Buy credits content</div><LocationDisplay /></>} />
        <Route path="plan" element={<><div>Plan content</div><LocationDisplay /></>} />
        <Route path="team" element={<><div>Team content</div><LocationDisplay /></>} />
        <Route path="activity" element={<><div>Activity content</div><LocationDisplay /></>} />
        <Route path="settings" element={<><div>Settings content</div><LocationDisplay /></>} />
        <Route path="admin" element={<><div>Admin content</div><LocationDisplay /></>} />
      </Route>
    </Routes>
  );
}

function renderLayout(initialEntry = '/dashboard') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <AppRoutes />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('Layout', () => {
  beforeEach(() => {
    localStorage.clear();
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    authState.memberships = [{ tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false }];
    tenantState.activeTenant = authState.memberships[0];
    brandingState.navItems = [];
    apiMocks.unreadCount.mockResolvedValue({ count: 0 });
    apiMocks.listPlans.mockResolvedValue({
      plans: [{ usageCreditsPerMonth: 10, bonusCredits: 0 }],
      tenantSubscriptionCredits: 10,
      tenantPurchasedCredits: 5,
      maxPlanUserLimit: 5,
    });
    apiMocks.listBundles.mockResolvedValue({ bundles: [{ id: 'bundle-1' }] });
    apiMocks.listAnnouncements.mockResolvedValue({ announcements: [] });
  });

  it('renders desktop and mobile ordinary-user navigation without covering content', async () => {
    renderLayout('/dashboard');

    expect(await screen.findByText('Dashboard content')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: 'Agents' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: 'Credits' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: 'History' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('link', { name: 'Settings' }).length).toBeGreaterThan(0);
    const mobileNav = screen.getByRole('navigation', { name: 'Primary mobile navigation' });
    expect(within(mobileNav).getByRole('link', { name: 'Agents' })).toHaveAttribute('href', '/dashboard');
    expect(within(mobileNav).getByRole('link', { name: 'Credits' })).toHaveAttribute('href', '/buy-credits');
    expect(within(mobileNav).getByRole('link', { name: 'History' })).toHaveAttribute('href', '/activity');
    expect(within(mobileNav).getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings');
    expect(screen.getByRole('main')).toHaveClass('pb-28', 'md:pb-8');
  });

  it('normalizes legacy backend default branding navigation to P0 app navigation for non-root users', async () => {
    authState.memberships = [
      { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'user', isRoot: false },
    ];
    tenantState.activeTenant = authState.memberships[0];
    brandingState.navItems = [
      {
        id: 'dashboard',
        label: 'Dashboard',
        icon: 'LayoutDashboard',
        target: '/dashboard',
        isBuiltIn: true,
        visible: true,
        sortOrder: 0,
      },
      {
        id: 'team',
        label: 'Team',
        icon: 'Users',
        target: '/team',
        isBuiltIn: true,
        visible: true,
        sortOrder: 1,
      },
      {
        id: 'plan',
        label: 'Plan',
        icon: 'CreditCard',
        target: '/plan',
        isBuiltIn: true,
        visible: true,
        sortOrder: 2,
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: 'Settings',
        target: '/settings',
        isBuiltIn: true,
        visible: true,
        sortOrder: 3,
      },
    ];

    renderLayout('/dashboard');

    await screen.findByText('Dashboard content');
    expect(screen.getAllByRole('link', { name: 'Agents' })).toHaveLength(2);
    expect(screen.getAllByRole('link', { name: 'Credits' })).toHaveLength(2);
    expect(screen.getAllByRole('link', { name: 'History' })).toHaveLength(2);
    expect(screen.getAllByRole('link', { name: 'Settings' })).toHaveLength(2);
    expect(screen.queryByRole('link', { name: 'Dashboard' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Plan' })).not.toBeInTheDocument();

    const mobileNav = screen.getByRole('navigation', { name: 'Primary mobile navigation' });
    expect(within(mobileNav).getByRole('link', { name: 'Agents' })).toHaveAttribute('href', '/dashboard');
    expect(within(mobileNav).getByRole('link', { name: 'Credits' })).toHaveAttribute('href', '/buy-credits');
    expect(within(mobileNav).getByRole('link', { name: 'History' })).toHaveAttribute('href', '/activity');
    expect(within(mobileNav).getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings');
  });

  it('reflects branded custom navigation in the mobile navigation landmark', async () => {
    brandingState.navItems = [
      {
        id: 'settings',
        label: 'Workspace Settings',
        icon: 'Settings',
        target: '/settings',
        isBuiltIn: true,
        visible: true,
        sortOrder: 30,
      },
      {
        id: 'hidden-history',
        label: 'Hidden History',
        icon: 'FileText',
        target: '/activity',
        isBuiltIn: false,
        visible: false,
        sortOrder: 10,
      },
      {
        id: 'marketplace',
        label: 'Marketplace',
        icon: 'Star',
        target: '/marketplace?featured=true',
        isBuiltIn: false,
        visible: true,
        sortOrder: 20,
      },
      {
        id: 'plan',
        label: 'Plan',
        icon: 'CreditCard',
        target: '/plan',
        isBuiltIn: true,
        visible: true,
        sortOrder: 35,
      },
      {
        id: 'team',
        label: 'Team',
        icon: 'Users',
        target: '/team',
        isBuiltIn: true,
        visible: true,
        sortOrder: 40,
      },
    ];

    renderLayout('/dashboard');

    await screen.findByText('Dashboard content');
    const mobileNav = screen.getByRole('navigation', { name: 'Primary mobile navigation' });
    const mobileLinks = within(mobileNav).getAllByRole('link');

    expect(mobileLinks.map((link) => link.textContent)).toEqual(['Marketplace', 'Workspace Settings', 'Plan', 'Team']);
    expect(within(mobileNav).getByRole('link', { name: 'Marketplace' })).toHaveAttribute('href', '/marketplace?featured=true');
    expect(within(mobileNav).getByRole('link', { name: 'Workspace Settings' })).toHaveAttribute('href', '/settings');
    expect(within(mobileNav).getByRole('link', { name: 'Plan' })).toHaveAttribute('href', '/plan');
    expect(within(mobileNav).getByRole('link', { name: 'Team' })).toHaveAttribute('href', '/team');
    expect(within(mobileNav).queryByRole('link', { name: 'Agents' })).not.toBeInTheDocument();
    expect(within(mobileNav).queryByRole('link', { name: 'Hidden History' })).not.toBeInTheDocument();
  });

  it('keeps branded Team and Plan discoverable for non-root tenant owners on multi-user plans', async () => {
    authState.memberships = [
      { tenantId: 'tenant-1', tenantName: 'Tenant One', tenantSlug: 'one', role: 'owner', isRoot: false },
    ];
    tenantState.activeTenant = authState.memberships[0];
    brandingState.navItems = [
      {
        id: 'team',
        label: 'Team',
        icon: 'Users',
        target: '/team',
        isBuiltIn: true,
        visible: true,
        sortOrder: 10,
      },
      {
        id: 'plan',
        label: 'Plan',
        icon: 'CreditCard',
        target: '/plan',
        isBuiltIn: true,
        visible: true,
        sortOrder: 20,
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: 'Settings',
        target: '/settings',
        isBuiltIn: true,
        visible: true,
        sortOrder: 30,
      },
    ];
    apiMocks.listPlans.mockResolvedValue({
      plans: [{ usageCreditsPerMonth: 0, bonusCredits: 0 }],
      tenantSubscriptionCredits: 0,
      tenantPurchasedCredits: 0,
      maxPlanUserLimit: 5,
    });

    renderLayout('/dashboard');

    await screen.findByText('Dashboard content');
    await waitFor(() => {
      expect(screen.getAllByRole('link', { name: 'Team' })).toHaveLength(2);
      expect(screen.getAllByRole('link', { name: 'Plan' })).toHaveLength(2);
    });
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();

    const mobileNav = screen.getByRole('navigation', { name: 'Primary mobile navigation' });
    expect(within(mobileNav).getByRole('link', { name: 'Team' })).toHaveAttribute('href', '/team');
    expect(within(mobileNav).getByRole('link', { name: 'Plan' })).toHaveAttribute('href', '/plan');
  });

  it('shows admin navigation only for root memberships', async () => {
    const { unmount } = renderLayout('/dashboard');
    await screen.findByText('Dashboard content');
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();
    unmount();

    authState.memberships = [{ tenantId: 'root-tenant', tenantName: 'Root', tenantSlug: 'root', role: 'owner', isRoot: true }];
    tenantState.activeTenant = authState.memberships[0];

    renderLayout('/dashboard');

    const adminLinks = await screen.findAllByRole('link', { name: 'Admin' });
    expect(adminLinks.length).toBeGreaterThan(0);
    expect(adminLinks.every((link) => link.getAttribute('href') === '/admin')).toBe(true);
  });

  it('routes the credits indicator to buy credits when bundles exist', async () => {
    const user = userEvent.setup();
    renderLayout('/dashboard');

    const creditButton = await screen.findByRole('button', { name: /15/i });
    await user.click(creditButton);

    expect(screen.getByTestId('location-display')).toHaveTextContent('/buy-credits');
  });

  it('routes the credits indicator to plan when bundles are unavailable', async () => {
    const user = userEvent.setup();
    apiMocks.listBundles.mockResolvedValue({ bundles: [] });
    renderLayout('/dashboard');

    const creditButton = await screen.findByRole('button', { name: /15/i });
    await user.click(creditButton);

    expect(screen.getByTestId('location-display')).toHaveTextContent('/plan');
  });

  it('dismisses latest announcement and stores dismissal', async () => {
    const user = userEvent.setup();
    apiMocks.listAnnouncements.mockResolvedValue({ announcements: [{ id: 'ann-1', title: 'New Agents available' }] });

    renderLayout('/dashboard');

    expect(await screen.findByText('New Agents available')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Dismiss' }));

    await waitFor(() => {
      expect(screen.queryByText('New Agents available')).not.toBeInTheDocument();
    });
    expect(localStorage.getItem('dismissed_announcement')).toBe('ann-1');
  });
});
