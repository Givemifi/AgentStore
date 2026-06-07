import { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useNavigate, useParams, useSearchParams, Link, useLocation } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, ArrowLeft, Globe2, Headphones, Landmark, Loader2, Menu, MessageSquare, PenLine, Plus, Scale, Send, Square, TowerControl, X, Zap } from 'lucide-react';
import axios from 'axios';
import { agentsApi, chatApi, usageApi, type ChatStreamEvent } from '../../api/client';
import { useTenant } from '../../contexts/TenantContext';
import type { ChatMessage, Conversation } from '../../types';
import { ErrorState, MarkdownMessage } from '../../components/app';
import LoadingSpinner from '../../components/LoadingSpinner';
import { getErrorMessage } from '../../utils/errors';

type ComposerNotice = {
  tone: 'error' | 'warning';
  message: string;
};

type ApiLikeError = {
  response?: {
    status?: number;
    data?: unknown;
  };
  name?: string;
};

const expertIconMap = {
  'legal-expert': Scale,
  'tax-advisor': Landmark,
  'marketing-copywriter': PenLine,
  'customer-support': Headphones,
  'telecom-business': TowerControl,
  'cross-border-ecommerce': Globe2,
};

function getExpertIcon(agentId: string) {
  return expertIconMap[agentId as keyof typeof expertIconMap] ?? MessageSquare;
}

function buildTemporaryMessage(params: {
  id: string;
  agentId: string;
  conversationId: string;
  role: ChatMessage['role'];
  content: string;
  creditsCharged: number;
  model?: string;
}): ChatMessage {
  return {
    id: params.id,
    tenantId: '',
    userId: '',
    conversationId: params.conversationId,
    agentId: params.agentId,
    role: params.role,
    content: params.content,
    creditsCharged: params.creditsCharged,
    model: params.model ?? '',
    createdAt: new Date().toISOString(),
  };
}

function formatConversationTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return 'Recently';
  }

  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(date);
}

function getApiErrorText(error: unknown) {
  if (axios.isAxiosError(error) && error.response?.data && typeof error.response.data === 'object' && 'error' in error.response.data) {
    return String(error.response.data.error);
  }

  const responseData = (error as ApiLikeError | null)?.response?.data;
  if (responseData && typeof responseData === 'object' && 'error' in responseData) {
    return String(responseData.error);
  }

  return '';
}

function getSendNotice(error: unknown): ComposerNotice {
  const status = axios.isAxiosError(error)
    ? error.response?.status
    : (error as ApiLikeError | null)?.response?.status;
  const apiError = getApiErrorText(error).toLowerCase();

  if (status === 402) {
    if (apiError.includes('billing') || apiError.includes('subscription')) {
      return {
        tone: 'error',
        message: 'Billing is not active for this workspace. Please contact an owner to activate billing.',
      };
    }

    return {
      tone: 'warning',
      message: 'You do not have enough credits for this message. Buy more credits to continue.',
    };
  }

  if (apiError.includes('not configured')) {
    return {
      tone: 'error',
      message: 'AI model is not configured right now. Please contact an administrator.',
    };
  }

  if (apiError.includes('conversation not found')) {
    return {
      tone: 'error',
      message: 'This conversation is no longer available. Start a new chat to continue.',
    };
  }

  return {
    tone: 'error',
    message: getErrorMessage(error),
  };
}

function isExpectedStreamAbort(error: unknown) {
  return (axios.isAxiosError(error) && error.code === 'Canceled') || (error as ApiLikeError | null)?.name === 'AbortError';
}

function ConversationListSkeleton() {
  return (
    <div className="space-y-3">
      {[0, 1, 2].map((item) => (
        <div key={item} className="rounded-2xl border border-dark-800 bg-dark-900/60 p-4">
          <div className="h-4 w-3/4 animate-pulse rounded bg-dark-800" />
          <div className="mt-3 h-3 w-1/2 animate-pulse rounded bg-dark-800" />
        </div>
      ))}
    </div>
  );
}

function MessageSkeleton() {
  return (
    <div className="space-y-4 py-2">
      <div className="max-w-[80%] rounded-2xl rounded-bl-md bg-dark-800 px-4 py-4">
        <div className="h-4 w-full animate-pulse rounded bg-dark-700" />
        <div className="mt-2 h-4 w-4/5 animate-pulse rounded bg-dark-700" />
      </div>
      <div className="ml-auto max-w-[70%] rounded-2xl rounded-br-md bg-dark-800 px-4 py-4">
        <div className="h-4 w-full animate-pulse rounded bg-dark-700" />
        <div className="mt-2 h-4 w-2/3 animate-pulse rounded bg-dark-700" />
      </div>
    </div>
  );
}

function ConversationItem({
  conversation,
  isActive,
  onClick,
}: {
  conversation: Conversation;
  isActive: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full rounded-2xl border px-4 py-3 text-left transition-colors ${
        isActive
          ? 'border-primary-500/60 bg-primary-500/10'
          : 'border-dark-800 bg-dark-900/60 hover:border-dark-700 hover:bg-dark-900'
      }`}
    >
      <p className="truncate text-sm font-medium text-white">{conversation.title}</p>
      <p className="mt-1 text-xs text-dark-400">Updated {formatConversationTime(conversation.updatedAt)}</p>
    </button>
  );
}

function getAssistantStatus(message: ChatMessage): { label: string; className: string } | null {
  if (message.role !== 'assistant') return null;
  if (message.status === 'interrupted') {
    return {
      label: 'Interrupted · no charge',
      className: 'border-amber-500/30 bg-amber-500/10 text-amber-200',
    };
  }
  if (message.status === 'error') {
    return {
      label: 'Failed · no charge',
      className: 'border-red-500/25 bg-red-500/10 text-red-200',
    };
  }
  return null;
}

function findPreviousUserMessage(messages: ChatMessage[], index: number): ChatMessage | null {
  for (let i = index - 1; i >= 0; i -= 1) {
    if (messages[i].role === 'user') {
      return messages[i];
    }
  }
  return null;
}

function ConversationSidebarContent({
  agent,
  conversations,
  conversationsLoading,
  conversationsError,
  conversationId,
  startNewChat,
  openConversation,
}: {
  agent: { name: string; category: string };
  conversations: Conversation[] | undefined;
  conversationsLoading: boolean;
  conversationsError: unknown;
  conversationId: string | null;
  startNewChat: () => void;
  openConversation: (nextConversationId: string) => void;
}) {
  return (
    <>
      <div className="border-b border-dark-800 p-4">
        <button
          onClick={startNewChat}
          className="flex w-full items-center justify-center gap-2 rounded-2xl bg-primary-500 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-primary-600"
        >
          <Plus className="h-4 w-4" />
          New chat
        </button>
        <div className="mt-4 rounded-2xl border border-dark-800 bg-dark-950/60 p-4">
          <p className="text-xs font-medium uppercase tracking-[0.16em] text-dark-500">Current Agent</p>
          <p className="mt-2 text-sm font-semibold text-white">{agent.name}</p>
          <p className="mt-1 text-sm text-dark-400">{agent.category}</p>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-white">Conversations</h2>
          {conversations && conversations.length > 0 && (
            <span className="text-xs text-dark-500">{conversations.length}</span>
          )}
        </div>

        {conversationsLoading ? (
          <ConversationListSkeleton />
        ) : conversationsError ? (
          <div className="rounded-2xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-200">
            {getErrorMessage(conversationsError)}
          </div>
        ) : conversations && conversations.length > 0 ? (
          <div className="space-y-3">
            {conversations.map((conversation) => (
              <ConversationItem
                key={conversation.id}
                conversation={conversation}
                isActive={conversation.id === conversationId}
                onClick={() => openConversation(conversation.id)}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-2xl border border-dashed border-dark-800 bg-dark-950/40 p-5 text-sm text-dark-400">
            No conversations yet. Start a new chat to create your first thread.
          </div>
        )}
      </div>
    </>
  );
}

export default function ChatPage() {
  const { agentId } = useParams<{ agentId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { activeTenant } = useTenant();
  const tenantReady = !!activeTenant;
  const [searchParams, setSearchParams] = useSearchParams();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const [draft, setDraft] = useState('');
  const [pendingUserMessage, setPendingUserMessage] = useState<ChatMessage | null>(null);
  const [composerNotice, setComposerNotice] = useState<ComposerNotice | null>(null);
  const [creditBalanceOverride, setCreditBalanceOverride] = useState<number | null>(null);
  const [createdConversationIds, setCreatedConversationIds] = useState<string[]>([]);
  const [isConversationDrawerOpen, setIsConversationDrawerOpen] = useState(false);
  const previouslyOpenRef = useRef(false);
  const openDrawerButtonRef = useRef<HTMLButtonElement>(null);
  const closeDrawerButtonRef = useRef<HTMLButtonElement>(null);
  const drawerRef = useRef<HTMLElement>(null);

  // Streaming state
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const streamingContentRef = useRef('');
  const abortControllerRef = useRef<AbortController | null>(null);

  const conversationId = searchParams.get('conversationId')?.trim() || null;
  const promptParam = searchParams.get('prompt')?.trim() || '';

  const {
    data: agent,
    isLoading: agentLoading,
    error: agentError,
  } = useQuery({
    queryKey: ['agent', agentId, activeTenant?.tenantId],
    queryFn: () => agentsApi.get(agentId!),
    enabled: !!agentId && tenantReady,
    retry: false,
  });

  const { data: usageData } = useQuery({
    queryKey: ['usage-summary', activeTenant?.tenantId],
    queryFn: () => usageApi.summary(),
    enabled: tenantReady,
  });

  const {
    data: conversations,
    isLoading: conversationsLoading,
    error: conversationsError,
  } = useQuery({
    queryKey: ['conversations', agentId, activeTenant?.tenantId],
    queryFn: () => chatApi.conversations(agentId),
    enabled: !!agentId && tenantReady,
  });

  const isConversationCreatedInSession = conversationId ? createdConversationIds.includes(conversationId) : false;
  // Safely resolve agentId - prefer agent.id from query, fallback to URL param
  const resolvedAgentId = agent?.id ?? agentId ?? '';
  const isConversationInCurrentAgent = conversationId
    ? (conversations?.some((conversation) => conversation.id === conversationId) ?? false) || isConversationCreatedInSession
    : false;
  const canValidateConversationSelection = Boolean(conversationId) && !conversationsLoading;
  const isConversationMissingFromAgent = canValidateConversationSelection && !isConversationInCurrentAgent;
  const canDisplayMessages = Boolean(conversationId) && isConversationInCurrentAgent && tenantReady;
  const shouldFetchMessages = canDisplayMessages && !isConversationCreatedInSession;

  const {
    data: messages,
    isLoading: messagesLoading,
    error: messagesError,
    refetch: refetchMessages,
    isFetching: messagesFetching,
  } = useQuery({
    queryKey: ['messages', conversationId],
    queryFn: () => chatApi.messages(conversationId!),
    enabled: shouldFetchMessages,
    retry: false,
  });

  const remainingCredits = creditBalanceOverride ?? ((usageData?.subscriptionCredits ?? 0) + (usageData?.purchasedCredits ?? 0));
  const isCreditBalanceKnown = creditBalanceOverride !== null || usageData !== undefined;
  const requiredCredits = agent?.creditCost?.textMessageCredits ?? 1;
  const hasInsufficientCredits = isCreditBalanceKnown && remainingCredits < requiredCredits;
  const isValidatingConversationSelection = Boolean(conversationId) && conversationsLoading && !isConversationCreatedInSession;
  const messageCount = messages?.length ?? 0;

  const proactiveCreditNotice = hasInsufficientCredits
    ? {
        tone: 'warning' as const,
        message: 'You do not have enough credits for this message. Buy more credits to continue.',
      }
    : null;
  const activeComposerNotice = composerNotice ?? proactiveCreditNotice;

  const displayedMessages = useMemo(() => {
    const baseMessages = canDisplayMessages ? (messages ?? []) : [];
    return pendingUserMessage ? [...baseMessages, pendingUserMessage] : baseMessages;
  }, [messages, pendingUserMessage, canDisplayMessages]);

  const hasActiveThread = Boolean(conversationId || pendingUserMessage || displayedMessages.length > 0);
  const isConversationMissing = isConversationMissingFromAgent || (axios.isAxiosError(messagesError) && messagesError.response?.status === 404);
  const agentMissing = axios.isAxiosError(agentError) && agentError.response?.status === 404;
  const ExpertIcon = agent ? getExpertIcon(agent.id) : MessageSquare;

  // Build returnTo for Buy Credits link - preserve current path and query params
  const currentPath = location.pathname;
  const currentSearch = searchParams.toString();
  const returnTo = currentSearch ? `${currentPath}?${currentSearch}` : currentPath;

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [displayedMessages.length, conversationId, messagesLoading]);

  useEffect(() => {
    setCreatedConversationIds([]);
  }, [agentId]);

  useEffect(() => {
    setComposerNotice(null);
    setPendingUserMessage(null);
  }, [conversationId]);

  // Focus management for drawer
  // Use autoFocus on close button when drawer opens
  // For focus return on close, we use useEffect with previouslyOpenRef
  useEffect(() => {
    if (!isConversationDrawerOpen && previouslyOpenRef.current) {
      // Drawer just closed after being open - return focus to open button
      // Use setImmediate/setTimeout to run after React has updated the DOM
      const handle = setTimeout(() => {
        openDrawerButtonRef.current?.focus();
      }, 10);
      return () => clearTimeout(handle);
    }
    // Update ref for next render
    previouslyOpenRef.current = isConversationDrawerOpen;
  }, [isConversationDrawerOpen]);

  // Handle Escape key to close drawer
  const handleDrawerKeyDown = useCallback((event: React.KeyboardEvent<HTMLElement>) => {
    if (event.key === 'Escape') {
      setIsConversationDrawerOpen(false);
    }

    // Focus trap: cycle through focusable elements inside drawer
    if ((event.key === 'Tab' || event.key === 'Shift+Tab') && drawerRef.current) {
      const focusableElements = drawerRef.current.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];

      if (event.shiftKey && document.activeElement === firstElement) {
        event.preventDefault();
        lastElement?.focus();
      } else if (!event.shiftKey && document.activeElement === lastElement) {
        event.preventDefault();
        firstElement?.focus();
      }
    }
  }, []);

  const focusComposer = useCallback(() => {
    requestAnimationFrame(() => {
      inputRef.current?.focus();
    });
  }, []);

  useEffect(() => {
    if (!promptParam) {
      return;
    }
    setDraft(promptParam);
    const nextParams = new URLSearchParams(searchParams);
    nextParams.delete('prompt');
    setSearchParams(nextParams, { replace: true });
    focusComposer();
  }, [promptParam, searchParams, setSearchParams, focusComposer]);

  const sendMessageMutation = useMutation({
    mutationFn: ({ message, currentConversationId }: { message: string; currentConversationId: string | null }) =>
      chatApi.send({
        agentId: resolvedAgentId,
        conversationId: currentConversationId || undefined,
        message,
      }),
    onSuccess: (response, variables) => {
      const resolvedConversationId = response.conversationId;
      setCreatedConversationIds((currentIds) => (
        currentIds.includes(resolvedConversationId) ? currentIds : [...currentIds, resolvedConversationId]
      ));
      const nextMessages = [
        buildTemporaryMessage({
          id: `user-${Date.now()}`,
          agentId: resolvedAgentId,
          conversationId: resolvedConversationId,
          role: 'user',
          content: variables.message,
          creditsCharged: 0,
        }),
        buildTemporaryMessage({
          id: `assistant-${Date.now()}`,
          agentId: resolvedAgentId,
          conversationId: resolvedConversationId,
          role: 'assistant',
          content: response.answer,
          creditsCharged: response.creditsCharged,
        }),
      ];

      queryClient.setQueryData<ChatMessage[]>(['messages', resolvedConversationId], (currentMessages) => [
        ...(currentMessages ?? []),
        ...nextMessages,
      ]);

      if (resolvedConversationId !== conversationId) {
        setSearchParams({ conversationId: resolvedConversationId });
      }

      setPendingUserMessage(null);
      setComposerNotice(null);
      setCreditBalanceOverride(response.remainingCredits);
      setDraft('');

      queryClient.invalidateQueries({ queryKey: ['conversations', agentId, activeTenant?.tenantId] });
      queryClient.invalidateQueries({ queryKey: ['usage-summary', activeTenant?.tenantId] });
    },
    onError: (error, variables) => {
      setPendingUserMessage(null);
      setComposerNotice(getSendNotice(error));
      setDraft(variables.message);
    },
  });

  const startNewChat = () => {
    setIsConversationDrawerOpen(false);
    setSearchParams({});
    setDraft('');
    setPendingUserMessage(null);
    setComposerNotice(null);
    // Clear streaming state
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsStreaming(false);
    setStreamingContent('');
    streamingContentRef.current = '';
    focusComposer();
  };

  const openConversation = (nextConversationId: string) => {
    setIsConversationDrawerOpen(false);
    setSearchParams({ conversationId: nextConversationId });
    setPendingUserMessage(null);
    setComposerNotice(null);
    // Clear streaming state
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsStreaming(false);
    setStreamingContent('');
    streamingContentRef.current = '';
  };

  const handleStopStreaming = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsStreaming(false);
  };

  const sendStreamMessage = useCallback(async (message: string, currentConversationId: string | null, agentIdToUse: string) => {
    // Guard against missing agentId
    if (!agentIdToUse) {
      return;
    }

    const conversationId = currentConversationId || `new-${Date.now()}`;
    const messageId = `assistant-${Date.now()}`;

    // Set streaming state
    setIsStreaming(true);
    setStreamingContent('');
    streamingContentRef.current = '';

    // Create abort controller
    abortControllerRef.current = new AbortController();

    // Add user message immediately
    const userMsg: ChatMessage = {
      id: `user-${Date.now()}`,
      tenantId: '',
      userId: '',
      conversationId,
      agentId: agentIdToUse,
      role: 'user',
      content: message,
      creditsCharged: 0,
      createdAt: new Date().toISOString(),
    };
    setPendingUserMessage(userMsg);
    setDraft('');

    try {
      await chatApi.stream(
        {
          agentId: agentIdToUse,
          conversationId: currentConversationId || undefined,
          message,
        },
        (event: ChatStreamEvent) => {
          if (event.event === 'delta') {
            streamingContentRef.current += event.data.text;
            setStreamingContent(streamingContentRef.current);
          } else if (event.event === 'message_done') {
            // Stream complete
            setIsStreaming(false);
            setCreditBalanceOverride(event.data.remainingCredits);

            // Add the completed messages to the cache
            const assistantMsg: ChatMessage = {
              id: event.data.messageId,
              tenantId: '',
              userId: '',
              conversationId: event.data.conversationId,
              agentId: agentIdToUse,
              role: 'assistant',
              content: streamingContentRef.current,
              creditsCharged: event.data.creditsCharged,
              model: event.data.model,
              createdAt: new Date().toISOString(),
            };

            queryClient.setQueryData<ChatMessage[]>(['messages', event.data.conversationId], (current) => [
              ...(current ?? []),
              userMsg,
              assistantMsg,
            ]);

            // Update URL if new conversation
            if (!currentConversationId) {
              setSearchParams({ conversationId: event.data.conversationId });
              setCreatedConversationIds(prev => [...prev, event.data.conversationId]);
            }

            // Invalidate queries
            queryClient.invalidateQueries({ queryKey: ['conversations', agentId, activeTenant?.tenantId] });
            queryClient.invalidateQueries({ queryKey: ['usage-summary', activeTenant?.tenantId] });

            // Clear streaming state
            setStreamingContent('');
            streamingContentRef.current = '';
            setPendingUserMessage(null);
          } else if (event.event === 'error') {
            setIsStreaming(false);
            setComposerNotice({ tone: 'error', message: event.data.message });
            setDraft(message);
            setPendingUserMessage(null);
            setStreamingContent('');
            streamingContentRef.current = '';
          }
        },
        abortControllerRef.current.signal
      );
    } catch (error) {
      if (isExpectedStreamAbort(error)) {
        // User cancelled - this is expected when stopping
        setIsStreaming(false);
        // Keep partial content if any
        if (streamingContentRef.current) {
          const partialMsg: ChatMessage = {
            id: messageId,
            tenantId: '',
            userId: '',
            conversationId,
            agentId: agentIdToUse,
            role: 'assistant',
            content: streamingContentRef.current,
            status: 'interrupted',
            creditsCharged: 0,
            createdAt: new Date().toISOString(),
          };
          queryClient.setQueryData<ChatMessage[]>(['messages', conversationId], (current) => [
            ...(current ?? []),
            userMsg,
            partialMsg,
          ]);
        }
      } else {
        setIsStreaming(false);
        setComposerNotice(getSendNotice(error));
        setDraft(message);
      }
      setPendingUserMessage(null);
      setStreamingContent('');
      streamingContentRef.current = '';
    }
  }, [agentId, activeTenant?.tenantId, queryClient, setSearchParams]);

  const handleExampleClick = (example: string) => {
    setDraft(example);
    setComposerNotice(null);
    focusComposer();
  };

  const handleRetryAssistantMessage = (messageIndex: number) => {
    const previousUserMessage = findPreviousUserMessage(displayedMessages, messageIndex);
    if (!previousUserMessage || isStreaming || sendMessageMutation.isPending || hasInsufficientCredits) {
      if (previousUserMessage) {
        setDraft(previousUserMessage.content);
        focusComposer();
      }
      return;
    }

    setComposerNotice(null);
    void sendStreamMessage(previousUserMessage.content, conversationId, resolvedAgentId);
  };

  const handleSendMessage = async () => {
    const message = draft.trim();
    // Guard: require a valid agentId
    if (!message || sendMessageMutation.isPending || isStreaming || !resolvedAgentId || hasInsufficientCredits) {
      return;
    }

    setComposerNotice(null);

    // Use streaming API with explicitly resolved agentId
    await sendStreamMessage(message, conversationId, resolvedAgentId);
  };

  const handleKeyDown = async (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      await handleSendMessage();
    }
  };

  if (agentLoading) {
    return <LoadingSpinner size="lg" className="py-20" />;
  }

  if (agentMissing) {
    return (
      <div className="mx-auto flex max-w-2xl flex-col items-center justify-center rounded-3xl border border-dark-800 bg-dark-900/60 px-6 py-16 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-dark-800 text-dark-300">
          <AlertCircle className="h-7 w-7" />
        </div>
        <h1 className="mt-5 text-2xl font-semibold text-white">Agent not found</h1>
        <p className="mt-2 max-w-lg text-sm text-dark-400">
          The Agent you tried to open is unavailable or may have been removed.
        </p>
        <button
          onClick={() => navigate('/dashboard')}
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-600"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Agents
        </button>
      </div>
    );
  }

  if (agentError || !agent) {
    return (
      <div className="mx-auto flex max-w-2xl flex-col items-center justify-center rounded-3xl border border-dark-800 bg-dark-900/60 px-6 py-16 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-dark-800 text-dark-300">
          <AlertCircle className="h-7 w-7" />
        </div>
        <h1 className="mt-5 text-2xl font-semibold text-white">Unable to load this Agent</h1>
        <p className="mt-2 max-w-lg text-sm text-dark-400">{getErrorMessage(agentError)}</p>
        <button
          onClick={() => navigate('/dashboard')}
          className="mt-6 inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-600"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Agents
        </button>
      </div>
    );
  }

  return (
    <div className="flex h-[calc(100vh-10rem)] min-h-[34rem] flex-col gap-4 lg:h-[calc(100vh-8rem)] lg:min-h-[40rem] lg:flex-row">
      <aside className="hidden w-full flex-col rounded-3xl border border-dark-800 bg-dark-900/60 lg:flex lg:w-80 lg:min-w-80">
        <ConversationSidebarContent
          agent={agent}
          conversations={conversations}
          conversationsLoading={conversationsLoading}
          conversationsError={conversationsError}
          conversationId={conversationId}
          startNewChat={startNewChat}
          openConversation={openConversation}
        />
      </aside>

      {isConversationDrawerOpen && (
        <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true" aria-label="Conversations">
          <button
            type="button"
            className="absolute inset-0 cursor-pointer bg-black/60"
            aria-label="Close conversations overlay"
            onClick={() => setIsConversationDrawerOpen(false)}
          />
          <aside
            ref={drawerRef}
            onKeyDown={handleDrawerKeyDown}
            className="relative z-10 flex h-full w-[min(22rem,86vw)] flex-col border-r border-dark-800 bg-dark-950 shadow-2xl"
          >
            <div className="flex items-center justify-between border-b border-dark-800 px-4 py-3">
              <h2 className="text-sm font-semibold text-white">Conversations</h2>
              <button
                type="button"
                ref={closeDrawerButtonRef}
                autoFocus
                onClick={() => setIsConversationDrawerOpen(false)}
                className="rounded-xl bg-dark-800 p-2 text-dark-300 transition-colors hover:bg-dark-700 hover:text-white"
                aria-label="Close conversations"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <ConversationSidebarContent
              agent={agent}
              conversations={conversations}
              conversationsLoading={conversationsLoading}
              conversationsError={conversationsError}
              conversationId={conversationId}
              startNewChat={startNewChat}
              openConversation={openConversation}
            />
          </aside>
        </div>
      )}

      <section className="flex min-h-0 min-w-0 flex-1 flex-col rounded-3xl border border-dark-800 bg-dark-900/60">
        <div className="border-b border-dark-800 p-4 sm:p-5">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex min-w-0 gap-3">
              <button
                ref={openDrawerButtonRef}
                onClick={() => setIsConversationDrawerOpen(true)}
                className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-dark-800 text-dark-300 transition-colors hover:bg-dark-700 hover:text-white lg:hidden"
                aria-label="Open conversations"
              >
                <Menu className="h-5 w-5" />
              </button>
              <button
                onClick={() => navigate('/dashboard')}
                className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-dark-800 text-dark-300 transition-colors hover:bg-dark-700 hover:text-white"
                aria-label="Back to Agents"
              >
                <ArrowLeft className="h-5 w-5" />
              </button>
              <div
                className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl"
                style={{ backgroundColor: `${agent.color}20` }}
              >
                <ExpertIcon className="h-6 w-6" style={{ color: agent.color }} />
              </div>
              <div className="min-w-0">
                <h1 className="truncate text-xl font-semibold text-white">{agent.name}</h1>
                <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-dark-300">
                  <span className="rounded-full bg-dark-800 px-2.5 py-1">{agent.category}</span>
                  <span className="inline-flex items-center gap-1 rounded-full bg-dark-800 px-2.5 py-1">
                    <Zap className="h-3.5 w-3.5 text-primary-400" />
                    {requiredCredits} credits / message
                  </span>
                </div>
                <p className="mt-3 max-w-3xl text-sm leading-6 text-dark-400">{agent.description}</p>
              </div>
            </div>

            <div className="inline-flex items-center gap-1.5 self-start rounded-xl border border-dark-700 bg-dark-950/60 px-3 py-2 text-sm text-dark-200">
              <Zap className="h-4 w-4 text-primary-400" />
              <span className="font-medium">{remainingCredits.toLocaleString()}</span>
              <span className="text-dark-500">credits left</span>
            </div>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-5">
          {isValidatingConversationSelection ? (
            <MessageSkeleton />
          ) : !hasActiveThread ? (
            <div className="flex h-full flex-col items-center justify-center text-center">
              <div
                className="flex h-18 w-18 items-center justify-center rounded-3xl"
                style={{ backgroundColor: `${agent.color}20` }}
              >
                <ExpertIcon className="h-10 w-10" style={{ color: agent.color }} />
              </div>
              <h2 className="mt-6 text-2xl font-semibold text-white">Start chatting with {agent.name}</h2>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-dark-400">{agent.description}</p>

              {agent.suggestedPrompts && agent.suggestedPrompts.length > 0 && (
                <div className="mt-8 w-full max-w-3xl">
                  <p className="mb-3 text-xs font-medium uppercase tracking-[0.16em] text-dark-500">Suggested prompts</p>
                  <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    {agent.suggestedPrompts.slice(0, 5).map((example) => (
                      <button
                        key={example}
                        onClick={() => handleExampleClick(example)}
                        aria-label={`Use suggested prompt: ${example}`}
                        className="rounded-2xl border border-dark-800 bg-dark-950/60 px-4 py-4 text-left text-sm text-dark-200 transition-colors hover:border-dark-700 hover:bg-dark-900 hover:text-white"
                      >
                        {example}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : messagesLoading && messageCount === 0 && !pendingUserMessage ? (
            <MessageSkeleton />
          ) : isConversationMissing ? (
            <div className="flex h-full flex-col items-center justify-center text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-dark-800 text-dark-300">
                <AlertCircle className="h-7 w-7" />
              </div>
              <h2 className="mt-5 text-xl font-semibold text-white">Conversation not found</h2>
              <p className="mt-2 max-w-lg text-sm text-dark-400">
                This conversation is no longer available. Start a new chat to continue with {agent.name}.
              </p>
              <button
                onClick={startNewChat}
                className="mt-6 inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-600"
              >
                <Plus className="h-4 w-4" />
                New chat
              </button>
            </div>
          ) : messagesError ? (
            <ErrorState
              title="Unable to load messages."
              message={getErrorMessage(messagesError)}
              retryLabel="Retry"
              onRetry={() => void refetchMessages()}
              isRetrying={messagesFetching}
            />
          ) : (
            <div className="space-y-4">
              {displayedMessages.map((message, index) => {
                const assistantStatus = getAssistantStatus(message);
                return (
                <div
                  key={message.id}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`max-w-[85%] rounded-2xl px-4 py-3 sm:max-w-[75%] ${
                      message.role === 'user'
                        ? 'rounded-br-md bg-primary-500 text-white'
                        : 'rounded-bl-md bg-dark-800 text-dark-100'
                    }`}
                  >
                    {message.role === 'assistant' ? (
                      <MarkdownMessage content={message.content} />
                    ) : (
                      <p className="whitespace-pre-wrap break-words text-sm leading-6">{message.content}</p>
                    )}
                    {((message.creditsCharged ?? 0) > 0) && (
                      <p className={`mt-2 text-xs ${message.role === 'user' ? 'text-primary-100' : 'text-dark-500'}`}>
                        {message.creditsCharged ?? 0} credits
                      </p>
                    )}
                    {assistantStatus && (
                      <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-white/6 pt-3">
                        <span className={`rounded-full border px-2.5 py-1 text-xs font-semibold ${assistantStatus.className}`}>
                          {assistantStatus.label}
                        </span>
                        <button
                          type="button"
                          onClick={() => handleRetryAssistantMessage(index)}
                          className="rounded-lg border border-dark-700 px-2.5 py-1 text-xs font-medium text-dark-200 transition-colors hover:border-primary-400/40 hover:text-white"
                          aria-label="Retry message"
                        >
                          Retry
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              );})}

              {/* Loading/streaming indicator */}
              {(sendMessageMutation.isPending || isStreaming) && (
                <div className="flex justify-start">
                  <div className="max-w-[85%] rounded-2xl rounded-bl-md bg-dark-800 px-4 py-3 text-dark-100 sm:max-w-[75%]">
                    {isStreaming ? (
                      <div className="space-y-2">
                        <div className="flex items-center gap-2 text-sm text-dark-300">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          <span>Generating... <button onClick={handleStopStreaming} className="text-primary-400 hover:underline">Stop</button></span>
                        </div>
                        {streamingContent && (
                          <div className="text-sm leading-6">
                            <MarkdownMessage content={streamingContent} />
                            <span className="animate-pulse">|</span>
                          </div>
                        )}
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 text-sm text-dark-300">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span>{agent.name} is thinking…</span>
                      </div>
                    )}
                  </div>
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        <div className="border-t border-dark-800 p-4 sm:p-5">
          {activeComposerNotice && (
            <div
              className={`mb-3 rounded-2xl border px-4 py-3 text-sm ${
                activeComposerNotice.tone === 'warning'
                  ? 'border-amber-500/30 bg-amber-500/10 text-amber-100'
                  : 'border-red-500/20 bg-red-500/10 text-red-200'
              }`}
            >
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <p>{activeComposerNotice.message}</p>
                {activeComposerNotice.tone === 'warning' && (
                  <Link
                    to={`/buy-credits?returnTo=${encodeURIComponent(returnTo)}`}
                    className="inline-flex items-center gap-2 self-start rounded-xl bg-amber-400/15 px-3 py-1.5 text-xs font-medium text-amber-100 transition-colors hover:bg-amber-400/25"
                  >
                    Buy Credits
                  </Link>
                )}
              </div>
            </div>
          )}

          <p className="mb-3 text-xs text-dark-400">
            This message costs {requiredCredits} credits. Failed responses are not charged.
          </p>

          <div className="flex items-end gap-3">
            {isStreaming ? (
              <button
                onClick={handleStopStreaming}
                className="flex h-12 w-12 items-center justify-center rounded-2xl bg-red-500 text-white transition-colors hover:bg-red-600"
                aria-label="Stop generating"
              >
                <Square className="h-5 w-5" />
              </button>
            ) : (
              <div className="relative flex-1">
                <textarea
                  ref={inputRef}
                  value={draft}
                  onChange={(event) => setDraft(event.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder={`Message ${agent.name}...`}
                  rows={1}
                  disabled={sendMessageMutation.isPending || isStreaming}
                  aria-invalid={hasInsufficientCredits}
                  aria-label={`Message ${agent.name}`}
                  className="min-h-[52px] w-full resize-none rounded-2xl border border-dark-700 bg-dark-800 px-4 py-3 pr-12 text-sm text-white placeholder-dark-500 focus:border-primary-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-70"
                  style={{ maxHeight: '160px' }}
                />
              </div>
            )}
            {!isStreaming && (
              <button
                onClick={handleSendMessage}
                disabled={!draft.trim() || sendMessageMutation.isPending || hasInsufficientCredits}
                className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-500 text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
                aria-label="Send message"
              >
                {sendMessageMutation.isPending ? <Loader2 className="h-5 w-5 animate-spin" /> : <Send className="h-5 w-5" />}
              </button>
            )}
          </div>

          <div className="mt-3 flex flex-col gap-1 text-xs text-dark-500 sm:flex-row sm:items-center sm:justify-between">
            <p>Press Enter to send, Shift+Enter for a new line.</p>
            <p className="inline-flex items-center gap-1.5">
              <MessageSquare className="h-3.5 w-3.5" />
              Each message costs {requiredCredits} credits.
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
