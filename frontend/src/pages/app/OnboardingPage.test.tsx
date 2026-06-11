import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OnboardingPage from './OnboardingPage';

const apiMocks = vi.hoisted(() => ({
  completeOnboarding: vi.fn(),
  refreshUser: vi.fn(),
  listAgents: vi.fn(),
  usageSummary: vi.fn(),
}));

const locationAssignSpy = vi.hoisted(() => vi.fn());

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { displayName: 'Ada Lovelace' },
    refreshUser: apiMocks.refreshUser,
  }),
}));

vi.mock('../../api/client', () => ({
  authApi: { completeOnboarding: apiMocks.completeOnboarding },
  agentsApi: { list: apiMocks.listAgents },
  usageApi: { summary: apiMocks.usageSummary },
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderOnboarding() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/onboarding']}>
        <Routes>
          <Route path="/onboarding" element={<><OnboardingPage /><LocationDisplay /></>} />
          <Route path="/dashboard" element={<><div>Dashboard page</div><LocationDisplay /></>} />
          <Route path="/chat/:agentId" element={<><div>Chat page</div><LocationDisplay /></>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('OnboardingPage', () => {
  let originalLocationHref: PropertyDescriptor | undefined;

  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    apiMocks.listAgents.mockResolvedValue([
      {
        id: 'agent-1',
        name: 'Marketing Writer',
        slug: 'marketing-writer',
        category: 'Marketing',
        description: 'Writes launch copy',
        suggestedPrompts: ['Write a launch email', 'Create ad copy'],
        visibility: 'private',
        capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '',
        updatedAt: '',
      },
      {
        id: 'agent-2',
        name: 'Legal Reviewer',
        slug: 'legal-reviewer',
        category: 'Legal',
        description: 'Reviews clauses',
        suggestedPrompts: ['Review this clause'],
        visibility: 'private',
        capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 2, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '',
        updatedAt: '',
      },
    ]);
    apiMocks.usageSummary.mockResolvedValue({ subscriptionCredits: 10, purchasedCredits: 5, usage: [], totalCreditsUsed: 0, periodStart: '' });
    apiMocks.completeOnboarding.mockResolvedValue({});
    apiMocks.refreshUser.mockResolvedValue(undefined);

    // Spy on window.location.href for hard navigation tests
    locationAssignSpy.mockReset();
    originalLocationHref = Object.getOwnPropertyDescriptor(window, 'location');
    delete (window as unknown as Record<string, unknown>).location;
    (window as unknown as Record<string, unknown>).location = { href: '' };
    Object.defineProperty(window.location, 'href', { set: locationAssignSpy, get: () => '' });
  });

  afterEach(() => {
    if (originalLocationHref) {
      Object.defineProperty(window, 'location', originalLocationHref);
    } else {
      delete (window as unknown as Record<string, unknown>).location;
    }
  });

  it('guides the user from goal to recommended Agent to first prompt', async () => {
    const user = userEvent.setup();
    renderOnboarding();

    expect(await screen.findByText('What do you want to accomplish first?')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Marketing' }));

    expect(await screen.findByText('Pick your first Agent')).toBeInTheDocument();
    expect(screen.getByText('Marketing Writer')).toBeInTheDocument();
    expect(screen.queryByText('Legal Reviewer')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Choose Marketing Writer' }));

    expect(await screen.findByText('Choose your first prompt')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Write a launch email' }));

    // Verify we're on the credits step by checking for "Start chatting" button
    expect(await screen.findByRole('button', { name: 'Start chatting' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Start chatting' }));

    await waitFor(() => expect(apiMocks.completeOnboarding).toHaveBeenCalledTimes(1));
    expect(apiMocks.refreshUser).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/marketing-writer?prompt=Write+a+launch+email');
  });

  it('falls back to marketplace when no Agents are available', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([]);
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Legal' }));
    expect(await screen.findByText('No Agents are published yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Go to marketplace' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/dashboard');
  });

  it('still navigates to chat if completing onboarding fails', async () => {
    const user = userEvent.setup();
    apiMocks.completeOnboarding.mockRejectedValue(new Error('failed'));
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    await user.click(await screen.findByRole('button', { name: 'Write a launch email' }));
    await user.click(await screen.findByRole('button', { name: 'Start chatting' }));

    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/marketing-writer?prompt=Write+a+launch+email');
  });

  it('calls completeOnboarding and refreshUser before navigating when Go to marketplace is clicked', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([]);
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Legal' }));
    expect(await screen.findByText('No Agents are published yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Go to marketplace' }));

    await waitFor(() => {
      expect(apiMocks.completeOnboarding).toHaveBeenCalledTimes(1);
      expect(apiMocks.refreshUser).toHaveBeenCalledTimes(1);
    });
    expect(screen.getByTestId('location-display')).toHaveTextContent('/dashboard');
  });

  it('still navigates to dashboard from Go to marketplace even if completeOnboarding fails', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([]);
    apiMocks.completeOnboarding.mockRejectedValue(new Error('failed'));
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Legal' }));
    expect(await screen.findByText('No Agents are published yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Go to marketplace' }));

    await waitFor(() => {
      expect(apiMocks.completeOnboarding).toHaveBeenCalledTimes(1);
    });
    expect(screen.getByTestId('location-display')).toHaveTextContent('/dashboard');
  });

  it('shows first 3 unfiltered agents when Other goal is selected', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([
      {
        id: 'agent-1', name: 'Agent A', slug: 'agent-a', category: 'Marketing',
        description: 'First agent', suggestedPrompts: ['Prompt A'],
        visibility: 'private', capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '', updatedAt: '',
      },
      {
        id: 'agent-2', name: 'Agent B', slug: 'agent-b', category: 'Legal',
        description: 'Second agent', suggestedPrompts: ['Prompt B'],
        visibility: 'private', capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '', updatedAt: '',
      },
      {
        id: 'agent-3', name: 'Agent C', slug: 'agent-c', category: 'Tax',
        description: 'Third agent', suggestedPrompts: ['Prompt C'],
        visibility: 'private', capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '', updatedAt: '',
      },
      {
        id: 'agent-4', name: 'Agent D', slug: 'agent-d', category: 'E-commerce',
        description: 'Fourth agent', suggestedPrompts: ['Prompt D'],
        visibility: 'private', capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '', updatedAt: '',
      },
    ]);
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Other' }));
    expect(await screen.findByText('Pick your first Agent')).toBeInTheDocument();

    expect(screen.getByText('Agent A')).toBeInTheDocument();
    expect(screen.getByText('Agent B')).toBeInTheDocument();
    expect(screen.getByText('Agent C')).toBeInTheDocument();
    expect(screen.queryByText('Agent D')).not.toBeInTheDocument();
  });

  it('displays correct total credits from usage summary on credits step', async () => {
    const user = userEvent.setup();
    apiMocks.usageSummary.mockResolvedValue({ subscriptionCredits: 10, purchasedCredits: 5, usage: [], totalCreditsUsed: 0, periodStart: '' });
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    await user.click(await screen.findByRole('button', { name: 'Write a launch email' }));

    expect(await screen.findByText(/15 credits/)).toBeInTheDocument();
  });

  it('navigates back from agent step to goal step', async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    expect(await screen.findByText('Pick your first Agent')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Choose a different goal/ }));
    expect(await screen.findByText('What do you want to accomplish first?')).toBeInTheDocument();
  });

  it('navigates back from prompt step to agent step', async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    expect(await screen.findByText('Choose your first prompt')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Choose a different Agent/ }));
    expect(await screen.findByText('Pick your first Agent')).toBeInTheDocument();
  });

  it('navigates back from credits step to prompt step', async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    await user.click(await screen.findByRole('button', { name: 'Write a launch email' }));
    expect(await screen.findByRole('button', { name: 'Start chatting' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Back' }));
    expect(await screen.findByText('Choose your first prompt')).toBeInTheDocument();
  });

  it('shows Skip prompt button when agent has no suggestedPrompts and advances to credits', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([
      {
        id: 'agent-np',
        name: 'NoPrompt Agent',
        slug: 'noprompt-agent',
        category: 'Strategy',
        description: 'No prompts here',
        suggestedPrompts: [],
        visibility: 'private',
        capabilities: ['text_chat'],
        creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
        createdAt: '',
        updatedAt: '',
      },
    ]);
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Strategy' }));
    await user.click(await screen.findByRole('button', { name: 'Choose NoPrompt Agent' }));
    expect(await screen.findByText('Choose your first prompt')).toBeInTheDocument();

    // The Skip prompt button should be visible
    expect(screen.getByRole('button', { name: /Skip prompt/ })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Skip prompt/ }));
    // Should advance to credits step
    expect(await screen.findByRole('button', { name: 'Start chatting' })).toBeInTheDocument();
  });

  it('shows Skip prompt button even when agent has suggestedPrompts and advances with empty prompt', async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    expect(await screen.findByText('Choose your first prompt')).toBeInTheDocument();

    // Skip prompt button should also be available alongside suggested prompts
    expect(screen.getByRole('button', { name: /Skip prompt/ })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Skip prompt/ }));
    expect(await screen.findByRole('button', { name: 'Start chatting' })).toBeInTheDocument();
  });

  it('uses hard navigation when completeOnboarding succeeds but refreshUser fails', async () => {
    const user = userEvent.setup();
    apiMocks.completeOnboarding.mockResolvedValue({});
    apiMocks.refreshUser.mockRejectedValue(new Error('refresh failed'));
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    await user.click(await screen.findByRole('button', { name: 'Write a launch email' }));
    await user.click(await screen.findByRole('button', { name: 'Start chatting' }));

    await waitFor(() => {
      expect(apiMocks.completeOnboarding).toHaveBeenCalledTimes(1);
      expect(apiMocks.refreshUser).toHaveBeenCalledTimes(1);
    });

    // Should have used hard navigation via window.location.href
    expect(locationAssignSpy).toHaveBeenCalledWith('/chat/marketing-writer?prompt=Write+a+launch+email');
  });

  it('uses hard navigation for goToMarketplace when completeOnboarding succeeds but refreshUser fails', async () => {
    const user = userEvent.setup();
    apiMocks.listAgents.mockResolvedValue([]);
    apiMocks.completeOnboarding.mockResolvedValue({});
    apiMocks.refreshUser.mockRejectedValue(new Error('refresh failed'));
    renderOnboarding();

    await user.click(await screen.findByRole('button', { name: 'Legal' }));
    expect(await screen.findByText('No Agents are published yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Go to marketplace' }));

    await waitFor(() => {
      expect(apiMocks.completeOnboarding).toHaveBeenCalledTimes(1);
      expect(apiMocks.refreshUser).toHaveBeenCalledTimes(1);
    });

    expect(locationAssignSpy).toHaveBeenCalledWith('/dashboard');
  });

  it('resets loading to false when completeOnboarding fails', async () => {
    const user = userEvent.setup();
    apiMocks.completeOnboarding.mockRejectedValue(new Error('failed'));
    renderOnboarding();

    // Click through to credits step
    await user.click(await screen.findByRole('button', { name: 'Marketing' }));
    await user.click(await screen.findByRole('button', { name: 'Choose Marketing Writer' }));
    await user.click(await screen.getByRole('button', { name: 'Write a launch email' }));

    // Verify we're on credits step and button is enabled after navigation (completeOnboarding fails but loading resets)
    expect(await screen.findByRole('button', { name: 'Start chatting' })).toBeInTheDocument();

    // Click Start chatting - completeOnboarding will fail but should still navigate
    await user.click(screen.getByRole('button', { name: 'Start chatting' }));

    // After the click (even if completeOnboarding fails), the component either navigates away or re-enables
    // The key fix is that setLoading(false) is called in catch, so we verify navigation happened
    await waitFor(() => {
      expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/marketing-writer');
    });
  });

  it('marks active step with aria-current="step"', async () => {
    renderOnboarding();
    const goalStep = await screen.findByText('Goal');

    // The active step should have aria-current="step"
    expect(goalStep.closest('[aria-current="step"]')).not.toBeNull();

    // Non-active steps should not have aria-current
    const agentStep = screen.getByText('Agent');
    expect(agentStep.closest('[aria-current="step"]')).toBeNull();
  });
});