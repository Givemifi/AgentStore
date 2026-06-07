import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { MessageCircle } from 'lucide-react';
import { describe, expect, it, vi } from 'vitest';
import EmptyState from './EmptyState';

vi.mock('react-router-dom', () => ({
  Link: ({ to, children, className }: { to: string; children: ReactNode; className?: string }) => (
    <a href={to} className={className}>{children}</a>
  ),
}));

describe('EmptyState', () => {
  it('renders title, description, icon, and link action', () => {
    render(
      <EmptyState
        icon={MessageCircle}
        title="No agents yet"
        description="Ask an administrator to publish your first Agent."
        action={{ label: 'Browse plans', to: '/plan' }}
      />
    );

    expect(screen.getByText('No agents yet')).toBeInTheDocument();
    expect(screen.getByText('Ask an administrator to publish your first Agent.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Browse plans' })).toHaveAttribute('href', '/plan');
  });

  it('renders button action when onClick is provided', () => {
    const onClick = vi.fn();
    render(
      <EmptyState
        icon={MessageCircle}
        title="No conversations"
        description="Start a new chat to create your first thread."
        action={{ label: 'Start chat', onClick }}
      />
    );

    screen.getByRole('button', { name: 'Start chat' }).click();
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
