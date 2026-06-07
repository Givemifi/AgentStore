import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SettingsPage from './SettingsPage';

const tenantState = vi.hoisted(() => ({
  value: { isRootTenant: false, role: null as 'owner' | 'admin' | 'user' | null },
}));

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: {
      id: 'user-1',
      email: 'admin@example.com',
      displayName: 'Admin User',
      authMethods: ['password'],
      emailVerified: true,
      isActive: true,
      totpEnabled: false,
    },
  }),
}));

vi.mock('../../contexts/TenantContext', () => ({
  useTenant: () => tenantState.value,
}));

vi.mock('../../contexts/BrandingContext', () => ({
  useBranding: () => ({
    branding: { authProviders: { passkeys: false, mfa: false } },
  }),
}));

vi.mock('./settings/ProfileTab', () => ({ default: () => <div>Profile panel</div> }));
vi.mock('./settings/SecurityTab', () => ({ default: () => <div>Security panel</div> }));
vi.mock('./settings/SessionsTab', () => ({ default: () => <div>Sessions panel</div> }));
vi.mock('./settings/BillingTab', () => ({ default: () => <div>Billing panel</div> }));
vi.mock('./settings/AgentsTab', () => ({ default: () => <div>Agents panel</div> }));
vi.mock('./settings/ModelSettingsTab', () => ({ default: () => <div>Model Settings panel</div> }));

function renderSettings(path = '/settings/models') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SettingsPage />
    </MemoryRouter>
  );
}

describe('SettingsPage', () => {
  beforeEach(() => {
    tenantState.value = { isRootTenant: false, role: null };
  });

  it('shows the Models tab content for /settings/models after root admin tenant context loads', async () => {
    const view = renderSettings('/settings/models');

    expect(screen.getByText('Profile panel')).toBeInTheDocument();
    expect(screen.queryByText('Model Settings panel')).not.toBeInTheDocument();

    tenantState.value = { isRootTenant: true, role: 'owner' };
    view.rerender(
      <MemoryRouter initialEntries={['/settings/models']}>
        <SettingsPage />
      </MemoryRouter>
    );

    await waitFor(() => expect(screen.getByText('Model Settings panel')).toBeInTheDocument());
    expect(screen.queryByText('Profile panel')).not.toBeInTheDocument();
  });
});
