import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import BillingSuccessPage from './BillingSuccessPage';

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: vi.fn(),
  };
});

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderSuccess() {
  return render(
    <MemoryRouter initialEntries={['/billing/success?session_id=test']}>
      <Routes>
        <Route path="/billing/success" element={<><BillingSuccessPage /><LocationDisplay /></>} />
        <Route path="/dashboard" element={<><div>Dashboard page</div><LocationDisplay /></>} />
        <Route path="/plan" element={<><div>Plan page</div><LocationDisplay /></>} />
        <Route path="/chat/:agentId" element={<><div>Chat page</div><LocationDisplay /></>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('BillingSuccessPage', () => {
  let mockedNavigate: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockedNavigate = vi.fn();
    vi.mocked(useNavigate).mockReturnValue(mockedNavigate);
  });

  afterEach(() => {
    vi.useRealTimers();
    sessionStorage.clear();
    vi.clearAllMocks();
  });

  it('shows success actions and continues previous conversation when return path exists', async () => {
    const user = userEvent.setup();
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/agent-1?conversationId=conv-1');

    renderSuccess();

    expect(screen.getByText('Credits added successfully.')).toBeInTheDocument();
    await user.click(screen.getByRole('link', { name: 'Continue conversation' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1?conversationId=conv-1');
    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('falls back to plan after delay when no return path exists', async () => {
    vi.useFakeTimers();

    renderSuccess();

    expect(screen.getByRole('link', { name: 'Browse Agents' })).toHaveAttribute('href', '/dashboard');

    // Advance timers and wait for the navigate to be called
    act(() => {
      vi.advanceTimersByTime(5000);
    });

    // Verify navigate was called with '/plan'
    expect(mockedNavigate).toHaveBeenCalledWith('/plan');
  });

  it('rejects protocol-relative returnTo like //evil.com', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '//evil.com');

    renderSuccess();

    // Should NOT show "Continue conversation" link for unsafe paths
    expect(screen.queryByRole('link', { name: 'Continue conversation' })).not.toBeInTheDocument();
    // Should fall back to auto-redirect behavior (show redirect message)
    expect(screen.getByText(/Redirecting to billing/)).toBeInTheDocument();
  });

  it('rejects path-traversal returnTo like /../etc/passwd', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/../etc/passwd');

    renderSuccess();

    expect(screen.queryByRole('link', { name: 'Continue conversation' })).not.toBeInTheDocument();
    expect(screen.getByText(/Redirecting to billing/)).toBeInTheDocument();
  });

  it('rejects path-traversal returnTo like /chat/../../../etc/passwd', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/../../../etc/passwd');

    renderSuccess();

    expect(screen.queryByRole('link', { name: 'Continue conversation' })).not.toBeInTheDocument();
    expect(screen.getByText(/Redirecting to billing/)).toBeInTheDocument();
  });

  it('rejects encoded path-traversal returnTo like /..%2F..%2Fetc/passwd', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/..%2F..%2Fetc/passwd');

    renderSuccess();

    expect(screen.queryByRole('link', { name: 'Continue conversation' })).not.toBeInTheDocument();
    expect(screen.getByText(/Redirecting to billing/)).toBeInTheDocument();
  });

  it('rejects encoded path-traversal returnTo like /%2F..%2Fevil.com', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/%2F..%2Fevil.com');

    renderSuccess();

    expect(screen.queryByRole('link', { name: 'Continue conversation' })).not.toBeInTheDocument();
    expect(screen.getByText(/Redirecting to billing/)).toBeInTheDocument();
  });

  it('accepts /chat/agent-1?conversationId=conv-1 as returnTo', async () => {
    const user = userEvent.setup();
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/agent-1?conversationId=conv-1');

    renderSuccess();

    const continueLink = screen.getByRole('link', { name: 'Continue conversation' });
    expect(continueLink).toBeInTheDocument();
    await user.click(continueLink);
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1?conversationId=conv-1');
  });

  it('accepts /dashboard as returnTo', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/dashboard');

    renderSuccess();

    const continueLink = screen.getByRole('link', { name: 'Continue conversation' });
    expect(continueLink).toBeInTheDocument();
  });

  it('clears checkoutReturnTo from sessionStorage on mount after reading', () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/agent-1?conversationId=conv-1');

    renderSuccess();

    // After mount, the key should be removed from sessionStorage to prevent stale paths
    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('clears returnTo when clicking Browse Agents link', async () => {
    const user = userEvent.setup();
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/agent-1');

    renderSuccess();

    await user.click(screen.getByRole('link', { name: 'Browse Agents' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('clears returnTo when clicking View billing link', async () => {
    const user = userEvent.setup();
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/chat/agent-1');

    renderSuccess();

    await user.click(screen.getByRole('link', { name: 'View billing' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('has aria-live region for success/redirect announcement', () => {
    renderSuccess();

    // There should be an aria-live region announcing the status
    const liveRegion = screen.getByRole('status');
    expect(liveRegion).toBeInTheDocument();
  });
});
