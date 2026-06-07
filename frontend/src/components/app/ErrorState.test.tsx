import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import ErrorState from './ErrorState';

describe('ErrorState', () => {
  it('renders a retry action when provided', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();

    render(
      <ErrorState
        title="Unable to load Agents"
        message="Network unavailable"
        retryLabel="Try again"
        onRetry={onRetry}
      />
    );

    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent('Unable to load Agents');
    expect(alert).toHaveTextContent('Network unavailable');

    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('disables retry while retrying', () => {
    render(
      <ErrorState
        title="Unable to load Agents"
        message="Retrying"
        retryLabel="Try again"
        onRetry={() => undefined}
        isRetrying
      />
    );

    expect(screen.getByRole('button', { name: 'Retrying...' })).toBeDisabled();
  });
});
