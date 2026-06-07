import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import LaunchChecklist, { type LaunchReadinessItem, INTEGRATION_DISPLAY_NAMES } from './LaunchChecklist';

describe('LaunchChecklist', () => {
  const allSevenItems: LaunchReadinessItem[] = [
    { id: 'brand', label: 'Brand configured', status: 'complete', description: 'Review app name and logo.', actionPath: '/admin/branding' },
    { id: 'model', label: 'Model provider connected', status: 'warning', description: 'Connect a provider.', actionPath: '/settings/models' },
    { id: 'agent', label: 'First Agent published', status: 'pending', description: 'Create and publish at least one Agent.', actionPath: '/settings/agents' },
    { id: 'credits', label: 'Credit bundle or plan active', status: 'complete', description: 'Review plans and credit bundles.', actionPath: '/admin/plans' },
    { id: 'stripe', label: 'Stripe webhook healthy', status: 'complete', description: 'Stripe integration is configured.', actionPath: '/admin/health#integrations' },
    { id: 'email', label: 'Email provider ready', status: 'warning', description: 'Configure Resend before relying on emails.', actionPath: '/admin/health#integrations' },
    { id: 'test-chat', label: 'Test chat passed', status: 'pending', description: 'Run a real smoke chat.', actionPath: '/dashboard' },
  ];

  it('renders all 7 checklist items', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    expect(screen.getByText('Launch Checklist')).toBeInTheDocument();
    expect(screen.getByText('Brand configured')).toBeInTheDocument();
    expect(screen.getByText('Model provider connected')).toBeInTheDocument();
    expect(screen.getByText('First Agent published')).toBeInTheDocument();
    expect(screen.getByText('Credit bundle or plan active')).toBeInTheDocument();
    expect(screen.getByText('Stripe webhook healthy')).toBeInTheDocument();
    expect(screen.getByText('Email provider ready')).toBeInTheDocument();
    expect(screen.getByText('Test chat passed')).toBeInTheDocument();
  });

  it('shows correct completed count in counter', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    // 3 complete out of 7
    expect(screen.getByText('3/7 ready')).toBeInTheDocument();
  });

  it('shows status labels for each status type', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    // Should have all three status types present
    expect(screen.getAllByText('Complete').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Needs attention').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Review').length).toBeGreaterThanOrEqual(1);
  });

  it('renders link href for First Agent published', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    expect(screen.getByRole('link', { name: /First Agent published/i })).toHaveAttribute('href', '/settings/agents');
  });

  it('renders link href for all items', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    expect(screen.getByRole('link', { name: /Brand configured/i })).toHaveAttribute('href', '/admin/branding');
    expect(screen.getByRole('link', { name: /Model provider connected/i })).toHaveAttribute('href', '/settings/models');
    expect(screen.getByRole('link', { name: /First Agent published/i })).toHaveAttribute('href', '/settings/agents');
    expect(screen.getByRole('link', { name: /Credit bundle or plan active/i })).toHaveAttribute('href', '/admin/plans');
    expect(screen.getByRole('link', { name: /Stripe webhook healthy/i })).toHaveAttribute('href', '/admin/health#integrations');
    expect(screen.getByRole('link', { name: /Email provider ready/i })).toHaveAttribute('href', '/admin/health#integrations');
    expect(screen.getByRole('link', { name: /Test chat passed/i })).toHaveAttribute('href', '/dashboard');
  });

  it('adds aria-hidden to decorative icons', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    // Every item has a status icon that should be aria-hidden
    const ariaHiddenIcons = document.querySelectorAll('svg[aria-hidden="true"]');
    expect(ariaHiddenIcons.length).toBeGreaterThanOrEqual(allSevenItems.length);
  });

  it('renders item descriptions', () => {
    render(
      <MemoryRouter>
        <LaunchChecklist items={allSevenItems} />
      </MemoryRouter>
    );

    expect(screen.getByText('Review app name and logo.')).toBeInTheDocument();
    expect(screen.getByText('Connect a provider.')).toBeInTheDocument();
    expect(screen.getByText('Create and publish at least one Agent.')).toBeInTheDocument();
  });
});

describe('INTEGRATION_DISPLAY_NAMES', () => {
  it('maps known integration keys to display names', () => {
    expect(INTEGRATION_DISPLAY_NAMES.stripe).toBe('Stripe');
    expect(INTEGRATION_DISPLAY_NAMES.resend).toBe('Resend');
    expect(INTEGRATION_DISPLAY_NAMES.mongodb).toBe('MongoDB');
    expect(INTEGRATION_DISPLAY_NAMES.google_oauth).toBe('Google Login');
    expect(INTEGRATION_DISPLAY_NAMES.github_oauth).toBe('GitHub Login');
  });
});
