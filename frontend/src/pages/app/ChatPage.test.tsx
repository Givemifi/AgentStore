import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import axios from 'axios';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ChatPage from './ChatPage';

const originalDomException = globalThis.DOMException;

const mockAgent = {
  id: 'agent-1',
  name: 'Growth Expert',
  slug: 'growth-expert',
  category: 'Marketing',
  description: 'Helps with growth plans',
  systemPrompt: 'Be helpful',
  creditCost: { textMessageCredits: 7, imageGenerationCredits: 0, videoGenerationCredits: 0 },
  suggestedPrompts: ['Draft a launch plan', 'Review my pricing page', 'Suggest onboarding improvements'],
  icon: 'rocket_launch',
  color: '#7c3aed',
  visibility: 'private',
  capabilities: ['text_chat'],
  createdAt: '2026-05-20T10:00:00.000Z',
  updatedAt: '2026-05-20T10:00:00.000Z',
};

const mockConversation = {
  id: 'conv-1',
  tenantId: 'tenant-1',
  userId: 'user-1',
  agentId: 'agent-1',
  title: 'Launch strategy',
  createdAt: '2026-05-20T10:00:00.000Z',
  updatedAt: '2026-05-21T12:00:00.000Z',
};

const mockMessages = [
  {
    id: 'msg-1',
    tenantId: 'tenant-1',
    userId: 'user-1',
    conversationId: 'conv-1',
    agentId: 'agent-1',
    role: 'user' as const,
    content: 'How should we launch?',
    creditsCharged: 0,
    model: 'model',
    createdAt: '2026-05-21T12:00:00.000Z',
  },
  {
    id: 'msg-2',
    tenantId: 'tenant-1',
    userId: 'user-1',
    conversationId: 'conv-1',
    agentId: 'agent-1',
    role: 'assistant' as const,
    content: 'Start with your best-fit audience.',
    creditsCharged: 7,
    model: 'model',
    createdAt: '2026-05-21T12:01:00.000Z',
  },
];

const apiMocks = vi.hoisted(() => ({
  agentsGet: vi.fn(),
  conversations: vi.fn(),
  messages: vi.fn(),
  stream: vi.fn(),
  usageSummary: vi.fn(),
}));

vi.mock('../../api/client', () => ({
  agentsApi: {
    get: apiMocks.agentsGet,
  },
  chatApi: {
    conversations: apiMocks.conversations,
    messages: apiMocks.messages,
    stream: apiMocks.stream,
  },
  usageApi: {
    summary: apiMocks.usageSummary,
  },
}));

vi.mock('../../contexts/TenantContext', () => ({
  useTenant: () => ({
    activeTenant: { tenantId: 'tenant-1', tenantName: 'Test Tenant', tenantSlug: 'test', role: 'owner', isRoot: false },
    setActiveTenant: vi.fn(),
    isRootTenant: false,
    role: 'owner',
  }),
}));

vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
  },
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}{location.search}</div>;
}

function renderChatPage(initialEntry = '/chat/agent-1') {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route
            path="/chat/:agentId"
            element={
              <>
                <ChatPage />
                <LocationDisplay />
              </>
            }
          />
          <Route path="/dashboard" element={<div>Experts page</div>} />
          <Route path="/buy-credits" element={<div>Buy credits page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('ChatPage', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());

    apiMocks.agentsGet.mockResolvedValue(mockAgent);
    apiMocks.conversations.mockResolvedValue([mockConversation]);
    apiMocks.messages.mockResolvedValue(mockMessages);
    apiMocks.stream.mockImplementation(async (_request, onEvent) => {
      onEvent({ event: 'message_start', data: { conversationId: 'conv-new', messageId: 'msg-new-assistant' } });
      onEvent({ event: 'delta', data: { text: 'Here is a plan.' } });
      onEvent({
        event: 'message_done',
        data: {
          conversationId: 'conv-new',
          messageId: 'msg-new-assistant',
          creditsCharged: 7,
          remainingCredits: 18,
          model: 'model',
        },
      });
    });
    apiMocks.usageSummary.mockResolvedValue({
      periodStart: '2026-05-01T00:00:00.000Z',
      usage: [],
      totalCreditsUsed: 0,
      subscriptionCredits: 20,
      purchasedCredits: 5,
    });
  });

  afterEach(() => {
    if (originalDomException) {
      globalThis.DOMException = originalDomException;
    }
  });

  it('shows a recoverable load state when the expert request fails without a 404', async () => {
    apiMocks.agentsGet.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 500,
        statusText: 'Server Error',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Backend unavailable' },
      } as never)
    );

    renderChatPage('/chat/agent-1');

    expect(await screen.findByText('Unable to load this expert')).toBeInTheDocument();
    expect(screen.queryByText('Expert not found')).not.toBeInTheDocument();
  });

  it('shows a welcome workspace when there is no conversationId and suggested prompts fill the composer', async () => {
    const user = userEvent.setup();
    renderChatPage('/chat/agent-1');

    await screen.findByText('Start chatting with Growth Expert');
    expect(screen.getByText('New chat')).toBeInTheDocument();
    expect(screen.getAllByText('Growth Expert').length).toBeGreaterThan(0);
    expect(screen.getByText('Draft a launch plan')).toBeInTheDocument();
    expect(screen.getByText('Launch strategy')).toBeInTheDocument();
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1');

    await user.click(screen.getByRole('button', { name: 'Draft a launch plan' }));

    expect(screen.getByRole('textbox')).toHaveValue('Draft a launch plan');
    expect(apiMocks.messages).not.toHaveBeenCalled();
  });

  it('loads an existing conversation from the query string', async () => {
    renderChatPage('/chat/agent-1?conversationId=conv-1');

    await waitFor(() => {
      expect(apiMocks.messages).toHaveBeenCalledWith('conv-1');
    });

    expect(await screen.findByText('How should we launch?')).toBeInTheDocument();
    expect(screen.getByText('Start with your best-fit audience.')).toBeInTheDocument();
  });

  it('sets the conversationId in the URL after the first successful send', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.messages.mockResolvedValue([]);

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');
    await user.keyboard('{Enter}');

    await waitFor(() => {
      expect(apiMocks.stream).toHaveBeenCalledWith(
        {
          agentId: 'agent-1',
          conversationId: undefined,
          message: 'Help me plan a launch',
        },
        expect.any(Function),
        expect.any(AbortSignal)
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1?conversationId=conv-new');
    });

    expect(await screen.findByText('Here is a plan.')).toBeInTheDocument();
  });

  it('keeps the first response visible after creating a conversation even if the conversation list stays stale', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.messages.mockImplementation(async (conversationId: string) => {
      if (conversationId === 'conv-new') {
        return [];
      }

      return mockMessages;
    });

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');
    await user.keyboard('{Enter}');

    await waitFor(() => {
      expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1?conversationId=conv-new');
    });

    expect(await screen.findByText('Help me plan a launch')).toBeInTheDocument();
    expect(screen.getByText('Here is a plan.')).toBeInTheDocument();
    expect(screen.queryByText('Conversation not found')).not.toBeInTheDocument();
  });

  it('clears the conversationId query param and returns to the welcome state when starting a new chat', async () => {
    const user = userEvent.setup();
    renderChatPage('/chat/agent-1?conversationId=conv-1');

    expect(await screen.findByText('How should we launch?')).toBeInTheDocument();

    await user.click(screen.getAllByRole('button', { name: 'New chat' })[0]);

    await waitFor(() => {
      expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1');
    });

    expect(screen.getByText('Start chatting with Growth Expert')).toBeInTheDocument();
    expect(screen.queryByText('How should we launch?')).not.toBeInTheDocument();
  });

  it('shows conversation not found for a deep-linked conversation outside the current agent list and does not fetch messages for it', async () => {
    apiMocks.conversations.mockResolvedValue([mockConversation]);

    renderChatPage('/chat/agent-1?conversationId=conv-other');

    expect(await screen.findByText('Conversation not found')).toBeInTheDocument();
    expect(
      screen.getByText('This conversation is no longer available. Start a new chat to continue with Growth Expert.')
    ).toBeInTheDocument();
    expect(apiMocks.messages).not.toHaveBeenCalled();
    expect(screen.queryByText('How should we launch?')).not.toBeInTheDocument();
  });

  it('does not fetch a deep-linked conversation when current-agent conversation validation fails', async () => {
    apiMocks.conversations.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 500,
        statusText: 'Server Error',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Unable to load conversations' },
      } as never)
    );

    renderChatPage('/chat/agent-1?conversationId=conv-other');

    expect(await screen.findByText('Conversation not found')).toBeInTheDocument();
    expect(apiMocks.messages).not.toHaveBeenCalled();
    expect(screen.queryByText('How should we launch?')).not.toBeInTheDocument();
  });

  it('shows a proactive insufficient credits warning with buy credits link and prevents sending', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.usageSummary.mockResolvedValue({
      periodStart: '2026-05-01T00:00:00.000Z',
      usage: [],
      totalCreditsUsed: 0,
      subscriptionCredits: 3,
      purchasedCredits: 1,
    });

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');

    expect(await screen.findByText('You do not have enough credits for this message. Buy more credits to continue.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Buy Credits' })).toHaveAttribute('href', '/buy-credits');
    expect(screen.getByRole('button', { name: 'Send message' })).toBeDisabled();

    await user.keyboard('{Enter}');

    expect(apiMocks.stream).not.toHaveBeenCalled();
    expect(textbox).toHaveValue('Help me plan a launch');
  });

  it('distinguishes a billing inactive 402 from insufficient credits and preserves the draft', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.stream.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 402,
        statusText: 'Payment Required',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Billing is not active for this tenant' },
      } as never)
    );

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');
    await user.keyboard('{Enter}');

    expect(await screen.findByText('Billing is not active for this workspace. Please contact an owner to activate billing.')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Buy Credits' })).not.toBeInTheDocument();
    expect(textbox).toHaveValue('Help me plan a launch');
    expect(screen.queryByText('Here is a plan.')).not.toBeInTheDocument();
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1');
  });

  it('keeps insufficient credits 402 actionable with a buy credits link and preserves the draft', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.stream.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 402,
        statusText: 'Payment Required',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Insufficient credits' },
      } as never)
    );

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');
    await user.keyboard('{Enter}');

    expect(await screen.findByText('You do not have enough credits for this message. Buy more credits to continue.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Buy Credits' })).toHaveAttribute('href', '/buy-credits');
    expect(textbox).toHaveValue('Help me plan a launch');
    expect(screen.queryByText('Here is a plan.')).not.toBeInTheDocument();
    expect(screen.getByTestId('location-display')).toHaveTextContent('/chat/agent-1');
  });

  it('maps non-axios stream errors with response status and body into the billing inactive notice', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.stream.mockRejectedValue({
      message: 'Unable to start chat stream',
      response: {
        status: 402,
        data: { error: 'Billing is not active for this tenant' },
      },
    });

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Help me plan a launch');
    await user.keyboard('{Enter}');

    expect(await screen.findByText('Billing is not active for this workspace. Please contact an owner to activate billing.')).toBeInTheDocument();
    expect(textbox).toHaveValue('Help me plan a launch');
    expect(screen.queryByText('Here is a plan.')).not.toBeInTheDocument();
  });

  it('treats AbortError from stopping a stream as an intentional cancellation without showing an error', async () => {
    const user = userEvent.setup();
    globalThis.DOMException = class DOMException extends Error {
      name = 'AbortError';
      constructor(message?: string, name = 'AbortError') {
        super(message);
        this.name = name;
      }
    } as typeof DOMException;
    apiMocks.stream.mockImplementation(async (_request, onEvent, signal) => {
      onEvent({ event: 'message_start', data: { conversationId: 'conv-1', messageId: 'msg-new-assistant' } });
      onEvent({ event: 'delta', data: { text: 'Partial answer' } });
      await new Promise((_resolve, reject) => {
        signal?.addEventListener('abort', () => reject(new globalThis.DOMException('The user aborted a request.', 'AbortError')));
      });
    });

    renderChatPage('/chat/agent-1?conversationId=conv-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Stop this');
    await user.keyboard('{Enter}');

    expect(await screen.findByText('Generating...')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Stop generating' }));

    expect(await screen.findByText('Partial answer [interrupted]')).toBeInTheDocument();
    expect(screen.queryByText('Unable to start chat stream')).not.toBeInTheDocument();
    expect(screen.queryByText('An unexpected error occurred')).not.toBeInTheDocument();
  });

  it('preserves the draft when sending fails', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);
    apiMocks.stream.mockRejectedValue(
      new axios.AxiosError('Request failed', undefined, undefined, undefined, {
        status: 500,
        statusText: 'Server Error',
        headers: {},
        config: { headers: {} as never },
        data: { error: 'Backend unavailable' },
      } as never)
    );

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'Keep this draft');
    await user.keyboard('{Enter}');

    await waitFor(() => {
      expect(apiMocks.stream).toHaveBeenCalledWith(
        {
          agentId: 'agent-1',
          conversationId: undefined,
          message: 'Keep this draft',
        },
        expect.any(Function),
        expect.any(AbortSignal)
      );
    });

    await waitFor(() => {
      expect(textbox).toHaveValue('Keep this draft');
    });
  });

  it('inserts a newline with Shift+Enter instead of sending', async () => {
    const user = userEvent.setup();
    apiMocks.conversations.mockResolvedValue([]);

    renderChatPage('/chat/agent-1');

    const textbox = await screen.findByRole('textbox');
    await user.type(textbox, 'First line');
    await user.keyboard('{Shift>}{Enter}{/Shift}');
    await user.type(textbox, 'Second line');

    expect(textbox).toHaveValue('First line\nSecond line');
    expect(apiMocks.stream).not.toHaveBeenCalled();
  });
});
