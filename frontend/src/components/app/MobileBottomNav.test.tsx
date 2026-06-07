import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { Activity, Bot, CreditCard, Settings, Star } from 'lucide-react';
import { describe, expect, it } from 'vitest';
import MobileBottomNav from './MobileBottomNav';
import type { LucideIcon } from 'lucide-react';

const defaultItems: Array<{ path: string; label: string; icon: LucideIcon }> = [
  { path: '/dashboard', label: 'Agents', icon: Bot },
  { path: '/buy-credits', label: 'Credits', icon: CreditCard },
  { path: '/activity', label: 'History', icon: Activity },
  { path: '/settings', label: 'Settings', icon: Settings },
];

function renderNav(path = '/dashboard', hasAdminAccess = false, items = defaultItems) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <MobileBottomNav items={items} hasAdminAccess={hasAdminAccess} />
    </MemoryRouter>
  );
}

describe('MobileBottomNav', () => {
  it('renders ordinary user mobile nav links', () => {
    renderNav('/dashboard');

    expect(screen.getByRole('link', { name: 'Agents' })).toHaveAttribute('href', '/dashboard');
    expect(screen.getByRole('link', { name: 'Credits' })).toHaveAttribute('href', '/buy-credits');
    expect(screen.getByRole('link', { name: 'History' })).toHaveAttribute('href', '/activity');
    expect(screen.getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings');
  });

  it('renders the provided mobile nav links', () => {
    renderNav('/marketplace', false, [
      { path: '/marketplace', label: 'Marketplace', icon: Star },
      { path: '/settings', label: 'Workspace Settings', icon: Settings },
    ]);

    expect(screen.getByRole('link', { name: 'Marketplace' })).toHaveAttribute('href', '/marketplace');
    expect(screen.getByRole('link', { name: 'Workspace Settings' })).toHaveAttribute('href', '/settings');
    expect(screen.queryByRole('link', { name: 'Agents' })).not.toBeInTheDocument();
  });

  it('marks the matching link as the current page', () => {
    renderNav('/dashboard');

    expect(screen.getByRole('link', { name: 'Agents' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Credits' })).not.toHaveAttribute('aria-current');
  });

  it('shows admin link for platform admins', () => {
    renderNav('/dashboard', true);

    expect(screen.getByRole('link', { name: 'Admin' })).toHaveAttribute('href', '/admin');
  });

  it('marks Admin as the current page on admin routes', () => {
    renderNav('/admin/users', true);

    expect(screen.getByRole('link', { name: 'Admin' })).toHaveAttribute('aria-current', 'page');
  });

  it('hides admin link for users without admin access', () => {
    renderNav('/dashboard', false);

    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();
  });
});
