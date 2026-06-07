import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import BuyCreditsPage from './BuyCreditsPage';

const apiMocks = vi.hoisted(() => ({
  listBundles: vi.fn(),
  listPlans: vi.fn(),
  checkout: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  bundlesApi: { list: apiMocks.listBundles },
  plansApi: { list: apiMocks.listPlans },
  billingApi: { checkout: apiMocks.checkout },
}));

vi.mock('sonner', () => ({ toast: { error: apiMocks.toastError } }));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderBuyCredits(initialEntry = '/buy-credits') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/buy-credits" element={<><BuyCreditsPage /><LocationDisplay /></>} />
        <Route path="/plan" element={<div>Plan page</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe('BuyCreditsPage', () => {
  let originalLocation: Location;

  beforeEach(() => {
    originalLocation = window.location;
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    apiMocks.listBundles.mockResolvedValue({
      bundles: [
        { id: 'bundle-1', name: 'Starter', credits: 100, priceCents: 900, isActive: true, sortOrder: 1, createdAt: '', updatedAt: '' },
        { id: 'bundle-2', name: 'Team', credits: 1000, priceCents: 4900, isActive: true, sortOrder: 2, createdAt: '', updatedAt: '' },
      ],
    });
    apiMocks.listPlans.mockResolvedValue({ tenantSubscriptionCredits: 24, tenantPurchasedCredits: 12, plans: [] });
  });

  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation });
    sessionStorage.clear();
  });

  it('renders current balance, credit education, and bundles with message estimates', async () => {
    renderBuyCredits();

    expect(await screen.findByText('Buy Credits')).toBeInTheDocument();
    expect(screen.getByText(/36/)).toBeInTheDocument();
    expect(screen.getByText('Credits are available immediately after checkout.')).toBeInTheDocument();
    expect(screen.getByText('Starter')).toBeInTheDocument();
    expect(screen.getByText('About 100 typical text messages')).toBeInTheDocument();
    expect(screen.getByText('$9.00')).toBeInTheDocument();
    expect(screen.getByText('Team')).toBeInTheDocument();
    expect(screen.getByText('Best for teams')).toBeInTheDocument();
  });

  it('shows an actionable empty state when bundles are unavailable', async () => {
    apiMocks.listBundles.mockResolvedValue({ bundles: [] });
    renderBuyCredits();

    expect(await screen.findByText('No credit bundles are available right now')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View plans' })).toHaveAttribute('href', '/plan');
  });

  it('stores return path and redirects to checkout URL after buying', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/chat/agent-1%3FconversationId%3Dconv-1');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(apiMocks.checkout).toHaveBeenCalledWith({ bundleId: 'bundle-1' });
    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBe('/chat/agent-1?conversationId=conv-1');
    expect(window.location.assign).toHaveBeenCalledWith('https://checkout.example/session');
  });

  it('disables buy buttons while checkout is loading', async () => {
    const user = userEvent.setup();
    let resolveCheckout: (value: { checkoutUrl: string }) => void = () => undefined;
    apiMocks.checkout.mockReturnValue(new Promise((resolve) => { resolveCheckout = resolve; }));

    renderBuyCredits();

    // Click the first buy button - it will change to loading state
    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));
    // Verify the button is disabled and shows loading state
    expect(screen.getByRole('button', { name: 'Starting checkout...' })).toBeDisabled();
    // Other buy buttons should also be disabled
    expect(screen.getByRole('button', { name: 'Buy Team' })).toBeDisabled();

    resolveCheckout({ checkoutUrl: 'https://checkout.example/session' });
  });

  it('rejects protocol-relative returnTo values like //evil.com', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=//evil.com');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
    expect(window.location.assign).toHaveBeenCalledWith('https://checkout.example/session');
  });

  it('rejects path-traversal returnTo like /../etc/passwd', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/../etc/passwd');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('rejects encoded path-traversal returnTo like /chat/../../../etc/passwd', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/chat/../../../etc/passwd');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('rejects double-encoded path-traversal returnTo like /..%2F..%2Fetc/passwd', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/..%2F..%2Fetc/passwd');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('rejects encoded path-traversal returnTo like /%2F..%2Fevil.com', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/%2F..%2Fevil.com');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('accepts /chat/agent-1?conversationId=conv-1 as returnTo', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({ checkoutUrl: 'https://checkout.example/session' });

    renderBuyCredits('/buy-credits?returnTo=/chat/agent-1%3FconversationId%3Dconv-1');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBe('/chat/agent-1?conversationId=conv-1');
  });

  it('shows error toast when checkout resolves without checkoutUrl', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockResolvedValue({});

    renderBuyCredits();

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    await waitFor(() => {
      expect(apiMocks.toastError).toHaveBeenCalledWith('Checkout failed. Please try again.');
    });
    expect(window.location.assign).not.toHaveBeenCalled();
  });

  it('does not store returnTo when checkout fails', async () => {
    const user = userEvent.setup();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, assign: vi.fn() },
    });
    apiMocks.checkout.mockRejectedValue(new Error('Network error'));

    renderBuyCredits('/buy-credits?returnTo=/chat/agent-1');

    await user.click(await screen.findByRole('button', { name: 'Buy Starter' }));

    await waitFor(() => {
      expect(apiMocks.toastError).toHaveBeenCalled();
    });
    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });

  it('guards price-per-100 calculation when credits is zero', async () => {
    apiMocks.listBundles.mockResolvedValue({
      bundles: [
        { id: 'bundle-free', name: 'Free', credits: 0, priceCents: 0, isActive: true, sortOrder: 1, createdAt: '', updatedAt: '' },
      ],
    });
    renderBuyCredits();

    expect(await screen.findByText('Free')).toBeInTheDocument();
    // Should not show NaN or Infinity; price-per-100 should be hidden or show "N/A"
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Infinity/)).not.toBeInTheDocument();
  });

  it('clears stale checkoutReturnTo from sessionStorage on mount', async () => {
    sessionStorage.setItem('agentstore.checkoutReturnTo', '/old-path');
    renderBuyCredits();

    await screen.findByText('Buy Credits');

    expect(sessionStorage.getItem('agentstore.checkoutReturnTo')).toBeNull();
  });
});
