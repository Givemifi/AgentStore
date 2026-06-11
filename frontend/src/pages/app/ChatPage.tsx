import { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useNavigate, useParams, useSearchParams, Link, useLocation } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, ArrowLeft, Check, Copy, FileText, Globe2, Headphones, Landmark, Loader2, Menu, MessageSquare, Mic, Paperclip, Pencil, PenLine, Plus, RefreshCw, Scale, Send, Share2, Square, ThumbsDown, ThumbsUp, TowerControl, Trash2, X, Zap } from 'lucide-react';
import axios from 'axios';
import { useTranslation } from 'react-i18next';
import { agentsApi, chatApi, usageApi, feedbackApi, type ChatStreamEvent, shareApi } from '../../api/client';
import { useTenant } from '../../contexts/TenantContext';
import type { ChatMessage, Conversation } from '../../types';
import { ErrorState, MarkdownMessage } from '../../components/app';
import LoadingSpinner from '../../components/LoadingSpinner';
import ConfirmModal from '../../components/ConfirmModal';
import { getErrorMessage } from '../../utils/errors';
import { useSpeechRecognition } from '../../hooks/useSpeechRecognition';
import {
  prepareAttachment,
  composeMessagePayload,
  MAX_ATTACHMENTS,
  type PreparedAttachment,
} from '../../utils/attachments';

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
        <div key={item} className="rounded-xl border border-white/8 bg-dark-900 p-4">
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
      <div className="max-w-[80%] rounded-xl rounded-bl-md bg-dark-800 px-4 py-4">
        <div className="h-4 w-full animate-pulse rounded bg-dark-700" />
        <div className="mt-2 h-4 w-4/5 animate-pulse rounded bg-dark-700" />
      </div>
      <div className="ml-auto max-w-[70%] rounded-xl rounded-br-md bg-dark-800 px-4 py-4">
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
  onRename,
  onDelete,
}: {
  conversation: Conversation;
  isActive: boolean;
  onClick: () => void;
  onRename: (id: string, title: string) => void;
  onDelete: (conversation: Conversation) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(conversation.title);

  const commit = () => {
    const next = title.trim();
    setEditing(false);
    if (next && next !== conversation.title) {
      onRename(conversation.id, next);
    } else {
      setTitle(conversation.title);
    }
  };

  if (editing) {
    return (
      <div className={`w-full rounded-md border px-3 py-2.5 ${isActive ? 'border-primary-500/60 bg-primary-500/10' : 'border-white/8 bg-dark-900'}`}>
        <input
          autoFocus
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') { e.preventDefault(); commit(); }
            if (e.key === 'Escape') { setTitle(conversation.title); setEditing(false); }
          }}
          maxLength={200}
          className="w-full rounded bg-dark-800 px-2 py-1 text-sm text-white outline-none ring-1 ring-primary-500/40"
        />
      </div>
    );
  }

  return (
    <div
      className={`group relative w-full rounded-md border px-3 py-2.5 transition-colors ${
        isActive
          ? 'border-primary-500/60 bg-primary-500/10'
          : 'border-white/8 bg-dark-900 hover:border-white/8 hover:bg-dark-900'
      }`}
    >
      <button onClick={onClick} className="block w-full pr-12 text-left">
        <p className="truncate text-sm font-medium text-white">{conversation.title}</p>
        <p className="mt-1 text-xs text-dark-400">Updated {formatConversationTime(conversation.updatedAt)}</p>
      </button>
      <div className="absolute right-2 top-2 flex gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
        <button
          onClick={() => { setTitle(conversation.title); setEditing(true); }}
          className="rounded p-1 text-dark-400 hover:bg-white/10 hover:text-white"
          aria-label="Rename conversation"
          title="Rename"
        >
          <Pencil className="h-3.5 w-3.5" />
        </button>
        <button
          onClick={() => onDelete(conversation)}
          className="rounded p-1 text-dark-400 hover:bg-white/10 hover:text-red-400"
          aria-label="Delete conversation"
          title="Delete"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
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
  onRename,
  onDelete,
}: {
  agent: { name: string; category: string };
  conversations: Conversation[] | undefined;
  conversationsLoading: boolean;
  conversationsError: unknown;
  conversationId: string | null;
  startNewChat: () => void;
  openConversation: (nextConversationId: string) => void;
  onRename: (id: string, title: string) => void;
  onDelete: (conversation: Conversation) => void;
}) {
  return (
    <>
      <div className="border-b border-white/8 p-4">
        <button
          onClick={startNewChat}
          className="flex w-full items-center justify-center gap-2 rounded-md bg-primary-500 h-8 text-[13px] font-medium text-white transition-colors hover:bg-primary-600"
        >
          <Plus className="h-4 w-4" />
          New chat
        </button>
        <div className="mt-4 rounded-xl border border-white/8 bg-dark-950/60 p-4">
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
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-200">
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
                onRename={onRename}
                onDelete={onDelete}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-white/8 bg-dark-950/40 p-5 text-sm text-dark-400">
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
  const { t } = useTranslation('app');
  const [shareState, setShareState] = useState<'idle' | 'sharing' | 'copied'>('idle');
  const tenantReady = !!activeTenant;
  const [searchParams, setSearchParams] = useSearchParams();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

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

  // Attachment state (images sent as base64 to LLM vision; documents parsed to text)
  const [attachments, setAttachments] = useState<PreparedAttachment[]>([]);
  const [isProcessingFiles, setIsProcessingFiles] = useState(false);

  // Streaming state
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const streamingContentRef = useRef('');
  // Ref to startNewChat so callbacks defined before it (e.g. delete mutation) can call it.
  const startNewChatRef = useRef<(() => void) | null>(null);
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

  // Per-message feedback (thumbs up/down) for the active conversation.
  const { data: feedbackMap } = useQuery({
    queryKey: ['feedback', conversationId],
    queryFn: () => feedbackApi.forConversation(conversationId!),
    enabled: shouldFetchMessages,
    retry: false,
  });

  const feedbackMutation = useMutation({
    mutationFn: ({ messageId, rating }: { messageId: string; rating: 1 | -1; comment?: string }) =>
      feedbackApi.set(messageId, rating, undefined),
    onSettled: () => {
      if (conversationId) void queryClient.invalidateQueries({ queryKey: ['feedback', conversationId] });
    },
  });
  const removeFeedbackMutation = useMutation({
    mutationFn: (messageId: string) => feedbackApi.remove(messageId),
    onSettled: () => {
      if (conversationId) void queryClient.invalidateQueries({ queryKey: ['feedback', conversationId] });
    },
  });

  const handleFeedback = useCallback((messageId: string, rating: 1 | -1) => {
    const current = feedbackMap?.[messageId]?.rating;
    if (current === rating) {
      removeFeedbackMutation.mutate(messageId);
    } else {
      feedbackMutation.mutate({ messageId, rating });
    }
  }, [feedbackMap, feedbackMutation, removeFeedbackMutation]);

  // Conversation management: rename / delete.
  const conversationsQueryKey = ['conversations', agentId, activeTenant?.tenantId];
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null);
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null);

  const renameMutation = useMutation({
    mutationFn: ({ id, title }: { id: string; title: string }) => chatApi.renameConversation(id, title),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: conversationsQueryKey }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => chatApi.deleteConversation(id),
    onSuccess: (_data, deletedId) => {
      void queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
      queryClient.removeQueries({ queryKey: ['messages', deletedId] });
      setPendingDelete(null);
      if (conversationId === deletedId) {
        startNewChatRef.current?.();
      }
    },
  });

  const handleRenameConversation = useCallback((id: string, title: string) => {
    renameMutation.mutate({ id, title });
  }, [renameMutation]);

  const handleCopyMessage = useCallback((messageId: string, content: string) => {
    void navigator.clipboard.writeText(content).then(() => {
      setCopiedMessageId(messageId);
      setTimeout(() => setCopiedMessageId((cur) => (cur === messageId ? null : cur)), 1500);
    });
  }, []);

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

  // --- Voice input (push-to-talk) ---
  const handleVoiceTranscript = useCallback((text: string) => {
    setDraft((current) => {
      const trimmed = current.trimEnd();
      return trimmed ? `${trimmed} ${text}` : text;
    });
  }, []);

  const {
    supported: voiceSupported,
    listening: isListening,
    interim: voiceInterim,
    error: voiceError,
    start: startVoice,
    stop: stopVoice,
  } = useSpeechRecognition(handleVoiceTranscript);

  useEffect(() => {
    if (voiceError) {
      setComposerNotice({ tone: 'error', message: voiceError });
    }
  }, [voiceError]);

  // --- File attachments ---
  const handlePickFiles = useCallback(async (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return;
    const files = Array.from(fileList);

    setComposerNotice(null);
    setIsProcessingFiles(true);
    try {
      for (const file of files) {
        // Enforce the total cap before processing each file.
        let reachedCap = false;
        setAttachments((current) => {
          if (current.length >= MAX_ATTACHMENTS) {
            reachedCap = true;
          }
          return current;
        });
        if (reachedCap) {
          setComposerNotice({ tone: 'warning', message: `最多上传 ${MAX_ATTACHMENTS} 个附件` });
          break;
        }
        try {
          const prepared = await prepareAttachment(file);
          setAttachments((current) =>
            current.length >= MAX_ATTACHMENTS ? current : [...current, prepared],
          );
        } catch (error) {
          setComposerNotice({ tone: 'error', message: getErrorMessage(error) });
        }
      }
    } finally {
      setIsProcessingFiles(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  }, []);

  const removeAttachment = useCallback((id: string) => {
    setAttachments((current) => current.filter((a) => a.id !== id));
  }, []);

  // On mobile, shrink the chat container to the visual viewport so the
  // soft keyboard doesn't push the header off-screen.
  // We write a CSS custom property --chat-h that the container reads via inline style.
  const chatContainerRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const viewport = window.visualViewport;
    if (!viewport) return;

    const update = () => {
      // Available height = visual viewport height minus the Layout header (64px)
      // and the mobile bottom nav area (padding-bottom: 7rem = 112px).
      const available = viewport.height - 64 - 112;
      if (chatContainerRef.current) {
        chatContainerRef.current.style.height = `${Math.max(available, 240)}px`;
      }
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    update();
    viewport.addEventListener('resize', update);
    viewport.addEventListener('scroll', update);
    return () => {
      viewport.removeEventListener('resize', update);
      viewport.removeEventListener('scroll', update);
    };
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
    setAttachments([]);
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
  useEffect(() => {
    startNewChatRef.current = startNewChat;
  });

  const openConversation = (nextConversationId: string) => {
    setIsConversationDrawerOpen(false);
    setSearchParams({ conversationId: nextConversationId });
    setPendingUserMessage(null);
    setComposerNotice(null);
    setAttachments([]);
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

  const handleShareConversation = async () => {
    if (!conversationId || shareState !== 'idle') return;
    setShareState('sharing');
    try {
      const { token } = await shareApi.create(conversationId);
      const shareUrl = `${window.location.origin}/share/${token}`;
      await navigator.clipboard.writeText(shareUrl);
      setShareState('copied');
      setTimeout(() => setShareState('idle'), 2500);
    } catch {
      setShareState('idle');
    }
  };

  const sendStreamMessage = useCallback(async (
    message: string,
    currentConversationId: string | null,
    agentIdToUse: string,
    images?: string[],
    // Original attachment objects saved before clearing the composer, so they
    // can be restored when the stream fails (e.g. proxy timeout or LLM error).
    originalAttachments?: PreparedAttachment[],
  ) => {
    // Guard against missing agentId
    if (!agentIdToUse) {
      return;
    }

    const conversationId = currentConversationId || `new-${Date.now()}`;
    const messageId = `assistant-${Date.now()}`;
    const imageAttachments = images && images.length > 0 ? images : undefined;

    setIsStreaming(true);
    setStreamingContent('');
    streamingContentRef.current = '';

    abortControllerRef.current = new AbortController();

    const userMsg: ChatMessage = {
      id: `user-${Date.now()}`,
      tenantId: '',
      userId: '',
      conversationId,
      agentId: agentIdToUse,
      role: 'user',
      content: message,
      creditsCharged: 0,
      // Attach image data URLs in-memory so the bubble can render thumbnails.
      // These are NOT persisted to the DB (too large); we store attachmentCount instead.
      attachments: imageAttachments,
      attachmentCount: imageAttachments ? imageAttachments.length : 0,
      createdAt: new Date().toISOString(),
    };
    setPendingUserMessage(userMsg);
    setDraft('');

    // Track whether the stream ended normally (message_done or error event).
    // If the stream closes without either, the connection was cut (e.g. proxy
    // timeout) and we need to recover the composer state.
    let streamFinishedNormally = false;

    const restoreComposer = () => {
      setDraft(message);
      if (originalAttachments && originalAttachments.length > 0) {
        setAttachments(originalAttachments);
      }
    };

    try {
      await chatApi.stream(
        {
          agentId: agentIdToUse,
          conversationId: currentConversationId || undefined,
          message,
          attachments: imageAttachments,
        },
        (event: ChatStreamEvent) => {
          if (event.event === 'delta') {
            streamingContentRef.current += event.data.text;
            setStreamingContent(streamingContentRef.current);
          } else if (event.event === 'message_done') {
            streamFinishedNormally = true;
            setIsStreaming(false);
            setCreditBalanceOverride(event.data.remainingCredits);

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

            if (!currentConversationId) {
              setSearchParams({ conversationId: event.data.conversationId });
              setCreatedConversationIds(prev => [...prev, event.data.conversationId]);
            }

            queryClient.invalidateQueries({ queryKey: ['conversations', agentId, activeTenant?.tenantId] });
            queryClient.invalidateQueries({ queryKey: ['usage-summary', activeTenant?.tenantId] });

            setStreamingContent('');
            streamingContentRef.current = '';
            setPendingUserMessage(null);
          } else if (event.event === 'error') {
            streamFinishedNormally = true;
            setIsStreaming(false);
            setComposerNotice({ tone: 'error', message: event.data.message });
            restoreComposer();
            setPendingUserMessage(null);
            setStreamingContent('');
            streamingContentRef.current = '';
          }
        },
        abortControllerRef.current.signal
      );

      // Stream ended without a terminal event — connection was cut (e.g. proxy
      // timeout while the LLM was still thinking). Recover the composer so the
      // user can retry without losing their message or attachments.
      if (!streamFinishedNormally) {
        setIsStreaming(false);
        setComposerNotice({ tone: 'error', message: 'Connection interrupted. Please try again.' });
        restoreComposer();
        setPendingUserMessage(null);
        setStreamingContent('');
        streamingContentRef.current = '';
      }
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
        restoreComposer();
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

  const handleRegenerateMessage = useCallback(async (message: ChatMessage) => {
    if (!conversationId || isStreaming || sendMessageMutation.isPending || hasInsufficientCredits) {
      return;
    }
    if (message.role !== 'assistant' || message.id.startsWith('assistant-') || message.id.startsWith('user-')) {
      return; // only persisted assistant messages can be regenerated
    }

    setComposerNotice(null);
    setIsStreaming(true);
    setStreamingContent('');
    streamingContentRef.current = '';
    abortControllerRef.current = new AbortController();

    // Optimistically drop the old assistant reply so it is visibly replaced.
    const snapshot = queryClient.getQueryData<ChatMessage[]>(['messages', conversationId]);
    queryClient.setQueryData<ChatMessage[]>(['messages', conversationId], (current) =>
      (current ?? []).filter((m) => m.id !== message.id),
    );

    let finishedNormally = false;
    try {
      await chatApi.regenerate(
        { conversationId, messageId: message.id },
        (event: ChatStreamEvent) => {
          if (event.event === 'delta') {
            streamingContentRef.current += event.data.text;
            setStreamingContent(streamingContentRef.current);
          } else if (event.event === 'message_done') {
            finishedNormally = true;
            setIsStreaming(false);
            setCreditBalanceOverride(event.data.remainingCredits);
            const assistantMsg: ChatMessage = {
              id: event.data.messageId,
              tenantId: '',
              userId: '',
              conversationId: event.data.conversationId,
              agentId: message.agentId,
              role: 'assistant',
              content: streamingContentRef.current,
              creditsCharged: event.data.creditsCharged,
              model: event.data.model,
              createdAt: new Date().toISOString(),
            };
            queryClient.setQueryData<ChatMessage[]>(['messages', conversationId], (current) => [
              ...(current ?? []),
              assistantMsg,
            ]);
            queryClient.invalidateQueries({ queryKey: ['usage-summary', activeTenant?.tenantId] });
            setStreamingContent('');
            streamingContentRef.current = '';
          } else if (event.event === 'error') {
            finishedNormally = true;
            setIsStreaming(false);
            setComposerNotice({ tone: 'error', message: event.data.message });
            // Restore the original reply on failure (credits were refunded server-side).
            if (snapshot) queryClient.setQueryData<ChatMessage[]>(['messages', conversationId], snapshot);
            setStreamingContent('');
            streamingContentRef.current = '';
          }
        },
        abortControllerRef.current.signal,
      );
      if (!finishedNormally) {
        setIsStreaming(false);
        setComposerNotice({ tone: 'error', message: 'Connection interrupted. Please try again.' });
        if (snapshot) queryClient.setQueryData<ChatMessage[]>(['messages', conversationId], snapshot);
        setStreamingContent('');
        streamingContentRef.current = '';
      }
    } catch (error) {
      setIsStreaming(false);
      if (!isExpectedStreamAbort(error)) {
        setComposerNotice(getSendNotice(error));
      }
      if (snapshot) queryClient.setQueryData<ChatMessage[]>(['messages', conversationId], snapshot);
      setStreamingContent('');
      streamingContentRef.current = '';
    }
  }, [conversationId, isStreaming, sendMessageMutation.isPending, hasInsufficientCredits, queryClient, activeTenant?.tenantId]);

  const handleSendMessage = async () => {
    // Compose message text (with parsed document text) and image data URLs.
    const { message, images } = composeMessagePayload(draft, attachments);

    // Guard: require text or at least one attachment, and a valid agentId.
    if ((!message && images.length === 0) || sendMessageMutation.isPending || isStreaming || !resolvedAgentId || hasInsufficientCredits) {
      return;
    }

    setComposerNotice(null);

    // Snapshot attachments before clearing, so they can be restored on error.
    const snapshotAttachments = [...attachments];

    // Clear attachments and draft after preparing the payload
    // (images variable already has the base64 data extracted)
    setAttachments([]);
    setDraft('');

    // Use streaming API with explicitly resolved agentId.
    // Pass images and the original attachments snapshot so errors can restore them.
    await sendStreamMessage(message, conversationId, resolvedAgentId, images.length > 0 ? images : undefined, snapshotAttachments);
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
      <div className="mx-auto flex max-w-2xl flex-col items-center justify-center rounded-xl border border-white/8 bg-dark-900 px-6 py-16 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-dark-800 text-dark-300">
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
      <div className="mx-auto flex max-w-2xl flex-col items-center justify-center rounded-xl border border-white/8 bg-dark-900 px-6 py-16 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-dark-800 text-dark-300">
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
    <div ref={chatContainerRef} className="flex h-[calc(100dvh-13rem)] min-h-0 flex-col gap-4 lg:h-[calc(100dvh-8rem)] lg:min-h-[40rem] lg:flex-row">
      <aside className="hidden w-full flex-col rounded-xl border border-white/8 bg-dark-900 lg:flex lg:w-80 lg:min-w-80">
        <ConversationSidebarContent
          agent={agent}
          conversations={conversations}
          conversationsLoading={conversationsLoading}
          conversationsError={conversationsError}
          conversationId={conversationId}
          startNewChat={startNewChat}
          openConversation={openConversation}
          onRename={handleRenameConversation}
          onDelete={setPendingDelete}
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
            className="relative z-10 flex h-full w-[min(22rem,86vw)] flex-col border-r border-white/8 bg-dark-950 shadow-2xl"
          >
            <div className="flex items-center justify-between border-b border-white/8 px-4 py-3">
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
              onRename={handleRenameConversation}
              onDelete={setPendingDelete}
            />
          </aside>
        </div>
      )}

      <section className="flex min-h-0 min-w-0 flex-1 flex-col rounded-xl border border-white/8 bg-dark-900">
        <div className="border-b border-white/8 p-4 sm:p-5">
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
                className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl"
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

            <div className="flex items-center gap-2 self-start">
              {conversationId && (
                <button
                  type="button"
                  onClick={handleShareConversation}
                  disabled={shareState !== 'idle'}
                  className="inline-flex items-center gap-1.5 rounded-xl border border-white/8 bg-dark-950/60 px-3 py-2 text-sm text-dark-300 transition-colors hover:border-primary-500/30 hover:text-white disabled:opacity-50"
                  title={shareState === 'copied' ? 'Link copied!' : 'Share conversation'}
                >
                  <Share2 className="h-4 w-4" />
                  <span className="hidden sm:inline">{shareState === 'copied' ? 'Copied!' : shareState === 'sharing' ? '...' : 'Share'}</span>
                </button>
              )}
              <div className="inline-flex items-center gap-1.5 rounded-xl border border-white/8 bg-dark-950/60 px-3 py-2 text-sm text-dark-200">
                <Zap className="h-4 w-4 text-primary-400" />
                <span className="font-medium">{remainingCredits.toLocaleString()}</span>
                <span className="text-dark-500">credits left</span>
              </div>
            </div>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-5">
          {isValidatingConversationSelection ? (
            <MessageSkeleton />
          ) : !hasActiveThread ? (
            <div className="flex h-full flex-col items-center justify-center text-center">
              <div
                className="flex h-18 w-18 items-center justify-center rounded-xl"
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
                        className="rounded-xl border border-white/8 bg-dark-950/60 px-4 py-4 text-left text-sm text-dark-200 transition-colors hover:border-white/8 hover:bg-dark-900 hover:text-white"
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
              <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-dark-800 text-dark-300">
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
              retryLabel={t('retry', { ns: 'common' })}
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
                    className={`max-w-[85%] rounded-lg px-3 py-2.5 sm:max-w-[75%] ${
                      message.role === 'user'
                        ? 'bg-primary-500 text-white'
                        : 'bg-dark-900 border border-white/8 text-dark-100'
                    }`}
                  >
                    {message.role === 'assistant' ? (
                      <MarkdownMessage content={message.content} />
                    ) : (
                      <div className="space-y-2">
                        {/* Image attachments: show thumbnails if data available, count badge otherwise */}
                        {(message.attachments && message.attachments.length > 0) ? (
                          <div className={`flex flex-wrap gap-2 ${message.content ? 'mb-2' : ''}`}>
                            {message.attachments.map((src, i) => (
                              <img
                                key={i}
                                src={src}
                                alt={`Attachment ${i + 1}`}
                                className="h-32 max-w-[180px] rounded-lg object-cover ring-2 ring-white/20"
                              />
                            ))}
                          </div>
                        ) : (message.attachmentCount ?? 0) > 0 ? (
                          <div className="flex items-center gap-1.5 text-sm text-primary-100">
                            <Paperclip className="h-3.5 w-3.5 shrink-0" />
                            <span>{message.attachmentCount} images</span>
                          </div>
                        ) : null}
                        {message.content && (
                          <p className="whitespace-pre-wrap break-words text-sm leading-6">{message.content}</p>
                        )}
                      </div>
                    )}
                    {((message.creditsCharged ?? 0) > 0) && (
                      <p className={`mt-2 text-xs ${message.role === 'user' ? 'text-primary-100' : 'text-dark-500'}`}>
                        {message.creditsCharged ?? 0} credits
                      </p>
                    )}
                    {message.role === 'assistant' && !assistantStatus && message.content && !message.id.startsWith('temp-') && (
                      <div className="mt-2 flex items-center gap-1">
                        <button
                          type="button"
                          onClick={() => handleFeedback(message.id, 1)}
                          className={`rounded-md p-1 transition-colors hover:bg-white/8 ${feedbackMap?.[message.id]?.rating === 1 ? 'text-emerald-400' : 'text-dark-500'}`}
                          aria-label={t('chat.feedback.helpful', { defaultValue: 'Helpful' })}
                          title={t('chat.feedback.helpful', { defaultValue: 'Helpful' })}
                        >
                          <ThumbsUp className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleFeedback(message.id, -1)}
                          className={`rounded-md p-1 transition-colors hover:bg-white/8 ${feedbackMap?.[message.id]?.rating === -1 ? 'text-red-400' : 'text-dark-500'}`}
                          aria-label={t('chat.feedback.notHelpful', { defaultValue: 'Not helpful' })}
                          title={t('chat.feedback.notHelpful', { defaultValue: 'Not helpful' })}
                        >
                          <ThumbsDown className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleCopyMessage(message.id, message.content)}
                          className={`rounded-md p-1 transition-colors hover:bg-white/8 ${copiedMessageId === message.id ? 'text-emerald-400' : 'text-dark-500'}`}
                          aria-label={t('chat.actions.copy', { defaultValue: 'Copy' })}
                          title={t('chat.actions.copy', { defaultValue: 'Copy' })}
                        >
                          {copiedMessageId === message.id ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                        </button>
                        <button
                          type="button"
                          onClick={() => void handleRegenerateMessage(message)}
                          disabled={isStreaming || hasInsufficientCredits}
                          className="rounded-md p-1 text-dark-500 transition-colors hover:bg-white/8 hover:text-white disabled:opacity-40"
                          aria-label={t('chat.actions.regenerate', { defaultValue: 'Regenerate' })}
                          title={t('chat.actions.regenerate', { defaultValue: 'Regenerate' })}
                        >
                          <RefreshCw className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    )}
                    {assistantStatus && (
                      <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-white/6 pt-3">
                        <span className={`rounded-full border px-2.5 py-1 text-xs font-semibold ${assistantStatus.className}`}>
                          {assistantStatus.label}
                        </span>
                        <button
                          type="button"
                          onClick={() => handleRetryAssistantMessage(index)}
                          className="rounded-lg border border-white/8 px-2.5 py-1 text-xs font-medium text-dark-200 transition-colors hover:border-primary-400/40 hover:text-white"
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
                  <div className="max-w-[85%] rounded-lg border border-white/8 bg-dark-900 px-3 py-2.5 text-dark-100 sm:max-w-[75%]">
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

        <div
          className="border-t border-white/8 p-4 sm:p-5"
          style={{ paddingBottom: 'calc(1rem + env(safe-area-inset-bottom))' }}
        >
          {activeComposerNotice && (
            <div
              className={`mb-3 rounded-xl border px-4 py-3 text-sm ${
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

          {/* Attachment previews */}
          {attachments.length > 0 && (
            <div className="mb-3 flex flex-wrap gap-2">
              {attachments.map((attachment) => (
                <div
                  key={attachment.id}
                  className="group relative flex items-center gap-2 rounded-xl border border-white/8 bg-dark-800 p-2"
                >
                  {attachment.kind === 'image' ? (
                    <img
                      src={attachment.dataUrl}
                      alt={attachment.name}
                      className="h-14 w-14 rounded-lg object-cover"
                    />
                  ) : (
                    <div className="flex h-14 w-14 items-center justify-center rounded-lg bg-dark-900 text-primary-400">
                      <FileText className="h-6 w-6" />
                    </div>
                  )}
                  <div className="max-w-[8rem] pr-1">
                    <p className="truncate text-xs font-medium text-white">{attachment.name}</p>
                    <p className="text-[11px] text-dark-500">
                      {attachment.kind === 'image' ? '图片' : '文档'}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeAttachment(attachment.id)}
                    className="absolute -right-2 -top-2 flex h-6 w-6 items-center justify-center rounded-full bg-dark-700 text-dark-200 shadow transition-colors hover:bg-red-500 hover:text-white"
                    aria-label={`移除附件 ${attachment.name}`}
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Voice listening indicator */}
          {isListening && (
            <div className="mb-3 flex items-center gap-2 rounded-xl border border-primary-500/30 bg-primary-500/10 px-4 py-2.5 text-sm text-primary-100">
              <span className="relative flex h-3 w-3">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary-400 opacity-75" />
                <span className="relative inline-flex h-3 w-3 rounded-full bg-primary-500" />
              </span>
              <span>{voiceInterim || '正在聆听… 松开按钮结束'}</span>
            </div>
          )}

          <p className="mb-3 text-xs text-dark-400">
            This message costs {requiredCredits} credits. Failed responses are not charged.
          </p>

          {/* Hidden file input */}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*,.pdf,.doc,.docx,.txt,.md"
            multiple
            className="hidden"
            onChange={(event) => void handlePickFiles(event.target.files)}
          />

          <div className="flex items-end gap-2 sm:gap-3">
            {/* Attach button */}
            {!isStreaming && (
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={attachments.length >= MAX_ATTACHMENTS || isProcessingFiles}
                className="flex h-12 min-h-[48px] w-12 min-w-[48px] shrink-0 items-center justify-center rounded-xl bg-dark-800 text-dark-300 transition-colors hover:bg-dark-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                aria-label="上传图片或文档"
                title="上传图片或文档"
              >
                {isProcessingFiles ? <Loader2 className="h-5 w-5 animate-spin" /> : <Paperclip className="h-5 w-5" />}
              </button>
            )}

            {/* Voice button (push-to-talk) */}
            {!isStreaming && voiceSupported && (
              <button
                type="button"
                onPointerDown={(event) => {
                  event.preventDefault();
                  startVoice();
                }}
                onPointerUp={() => stopVoice()}
                onPointerLeave={() => isListening && stopVoice()}
                onPointerCancel={() => stopVoice()}
                disabled={sendMessageMutation.isPending}
                className={`flex h-12 min-h-[48px] w-12 min-w-[48px] shrink-0 select-none items-center justify-center rounded-xl transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                  isListening
                    ? 'bg-primary-500 text-white'
                    : 'bg-dark-800 text-dark-300 hover:bg-dark-700 hover:text-white'
                }`}
                style={{ touchAction: 'none' }}
                aria-label="按住说话"
                title="按住说话"
              >
                <Mic className="h-5 w-5" />
              </button>
            )}

            {isStreaming ? (
              <button
                onClick={handleStopStreaming}
                className="flex h-12 min-h-[48px] w-12 min-w-[48px] items-center justify-center rounded-xl bg-red-500 text-white transition-colors hover:bg-red-600"
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
                  className="min-h-[48px] w-full resize-none rounded-xl border border-white/8 bg-dark-800 px-4 py-3 pr-4 text-base text-white placeholder-dark-500 focus:border-primary-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-70 sm:text-sm"
                  style={{ maxHeight: '160px' }}
                />
              </div>
            )}
            {!isStreaming && (
              <button
                onClick={handleSendMessage}
                disabled={(!draft.trim() && attachments.length === 0) || sendMessageMutation.isPending || hasInsufficientCredits}
                className="flex h-12 min-h-[48px] w-12 min-w-[48px] shrink-0 items-center justify-center rounded-xl bg-primary-500 text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
                aria-label="Send message"
              >
                {sendMessageMutation.isPending ? <Loader2 className="h-5 w-5 animate-spin" /> : <Send className="h-5 w-5" />}
              </button>
            )}
          </div>

          <div className="mt-3 flex flex-col gap-1 text-xs text-dark-500 sm:flex-row sm:items-center sm:justify-between">
            <p className="hidden sm:block">Press Enter to send, Shift+Enter for a new line.</p>
            <p className="inline-flex items-center gap-1.5">
              <MessageSquare className="h-3.5 w-3.5" />
              Each message costs {requiredCredits} credits.
            </p>
          </div>
        </div>
      </section>

      <ConfirmModal
        open={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        onConfirm={() => { if (pendingDelete) deleteMutation.mutate(pendingDelete.id); }}
        title={t('chat.deleteConversation.title', { defaultValue: 'Delete conversation?' })}
        message={t('chat.deleteConversation.message', { defaultValue: 'This will permanently delete this conversation and all its messages. This cannot be undone.' })}
        confirmLabel={t('chat.deleteConversation.confirm', { defaultValue: 'Delete' })}
        confirmVariant="danger"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
