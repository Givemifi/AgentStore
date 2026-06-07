import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AdminLayout from './AdminLayout';

const tenantMock = vi.hoisted(() => ({
  activeTenantReady: true,
  isRootTenant: true,
  role: 'owner' as 'owner' | 'admin' | 'user' | null,
}));

const apiMocks = vi.hoisted(() => ({
  unreadCount: vi.fn(),
}));

vi.mock('../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: tenantMock.activeTenantReady ? {
      tenantId: 'tenant-1',
      tenantName: 'Root Tenant',
      tenantSlug: 'root',
      role: tenantMock.role,
      isRoot: tenantMock.isRootTenant,
    } : null,
    setActiveTenant: vi.fn(),
    isRootTenant: tenantMock.activeTenantReady ? tenantMock.isRootTenant : false,
    role: tenantMock.activeTenantReady ? tenantMock.role : null,
    isTenantReady: tenantMock.activeTenantReady,
  }),
}));

vi.mock('../api/client', () => ({
  messagesApi: {
    unreadCount: apiMocks.unreadCount,
  },
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}</div>;
}

function renderAdminLayout(initialEntry = '/admin/llm-config') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="/admin"
          element={<AdminLayout />}
        >
          <Route path="llm-config" element={<div>LLM Config Outlet</div>} />
          <Route index element={<div>Admin Outlet</div>} />
        </Route>
        <Route path="/dashboard" element={<LocationDisplay />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('AdminLayout', () => {
  beforeEach(() => {
    tenantMock.activeTenantReady = true;
    tenantMock.isRootTenant = true;
    tenantMock.role = 'owner';
    apiMocks.unreadCount.mockReset();
    apiMocks.unreadCount.mockResolvedValue({ count: 0 });
  });

  it('redirects non-root users away before rendering admin navigation or content', async () => {
    tenantMock.isRootTenant = false;
    tenantMock.role = 'user';

    renderAdminLayout('/admin/llm-config');

    expect(await screen.findByTestId('location-display')).toHaveTextContent('/dashboard');
    expect(screen.queryByText('LLM Config')).not.toBeInTheDocument();
    expect(screen.queryByText('LLM Config Outlet')).not.toBeInTheDocument();
    expect(apiMocks.unreadCount).not.toHaveBeenCalled();
  });

  it('hides the LLM Config navigation item from non-owner root members', async () => {
    tenantMock.isRootTenant = true;
    tenantMock.role = 'admin';

    renderAdminLayout('/admin');

    expect(await screen.findByText('Admin Outlet')).toBeInTheDocument();
    expect(screen.queryByText('LLM Config')).not.toBeInTheDocument();
  });

  it('waits for tenant restoration before redirecting admin deep links', async () => {
    tenantMock.activeTenantReady = false;
    tenantMock.isRootTenant = true;
    tenantMock.role = null;

    renderAdminLayout('/admin/llm-config');

    expect(screen.queryByTestId('location-display')).not.toBeInTheDocument();
    expect(screen.queryByText('LLM Config Outlet')).not.toBeInTheDocument();
    expect(apiMocks.unreadCount).not.toHaveBeenCalled();
  });

  it('shows the LLM Config navigation item and outlet for owner root members', async () => {
    renderAdminLayout('/admin/llm-config');

    expect(await screen.findByText('LLM Config Outlet')).toBeInTheDocument();
    expect(screen.getByText('LLM Config')).toBeInTheDocument();
    await waitFor(() => {
      expect(apiMocks.unreadCount).toHaveBeenCalledTimes(1);
    });
  });
});
