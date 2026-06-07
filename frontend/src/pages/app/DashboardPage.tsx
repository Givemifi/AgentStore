import { useMemo, useState } from 'react';
import DOMPurify from 'dompurify';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Landmark, MessageCircle, PenLine, Scale, Search, Sparkles, TowerControl, Zap, Headphones, Globe2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { agentsApi, brandingApi, usageApi } from '../../api/client';
import { useTenant } from '../../contexts/TenantContext';
import { CreditExplainer, EmptyState, ErrorState } from '../../components/app';
import type { Agent } from '../../types';
import { getErrorMessage } from '../../utils/errors';

const expertBadges: Record<string, { label: string; className: string }> = {
  'legal-expert': {
    label: 'Legal',
    className: 'border-violet-400/30 bg-violet-500/10 text-violet-200',
  },
  'tax-advisor': {
    label: 'Business',
    className: 'border-emerald-400/30 bg-emerald-500/10 text-emerald-200',
  },
  'marketing-copywriter': {
    label: 'Popular',
    className: 'border-rose-400/30 bg-rose-500/10 text-rose-200',
  },
  'customer-support': {
    label: 'Support',
    className: 'border-blue-400/30 bg-blue-500/10 text-blue-200',
  },
  'telecom-business': {
    label: 'Business',
    className: 'border-amber-400/30 bg-amber-500/10 text-amber-200',
  },
  'cross-border-ecommerce': {
    label: 'Business',
    className: 'border-cyan-400/30 bg-cyan-500/10 text-cyan-200',
  },
};

const expertIconMap = {
  'legal-expert': Scale,
  'tax-advisor': Landmark,
  'marketing-copywriter': PenLine,
  'customer-support': Headphones,
  'telecom-business': TowerControl,
  'cross-border-ecommerce': Globe2,
};

export default function DashboardPage() {
  const navigate = useNavigate();
  const { activeTenant } = useTenant();
  const tenantReady = !!activeTenant;

  const {
    data: agents,
    isLoading: agentsLoading,
    error: agentsError,
    refetch: refetchAgents,
    isFetching: agentsFetching,
  } = useQuery({
    queryKey: ['agents', activeTenant?.tenantId],
    queryFn: () => agentsApi.list(),
    enabled: tenantReady,
    retry: false,
  });

  const {
    data: usageSummary,
    isLoading: usageLoading,
    refetch: refetchUsage,
  } = useQuery({
    queryKey: ['usage-summary', activeTenant?.tenantId],
    queryFn: () => usageApi.summary(),
    enabled: tenantReady,
    retry: false,
  });

  const { data: branding } = useQuery({
    queryKey: ['branding'],
    queryFn: () => brandingApi.get(),
    retry: false,
  });

  const sanitizedDashboardHtml = branding?.dashboardHtml
    ? DOMPurify.sanitize(branding.dashboardHtml).trim()
    : '';

  const availableCredits = usageSummary
    ? usageSummary.subscriptionCredits + usageSummary.purchasedCredits
    : null;
  const availableCreditsValue = availableCredits ?? 0;

  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('Recommended');

  const categories = useMemo(() => {
    const uniqueCategories = Array.from(new Set((agents ?? []).map((agent) => agent.category).filter(Boolean))).sort();
    return ['Recommended', ...uniqueCategories];
  }, [agents]);

  const filteredAgents = useMemo(() => {
    const normalizedSearch = searchTerm.trim().toLowerCase();
    return (agents ?? []).filter((agent) => {
      const matchesSearch = normalizedSearch === '' || [
        agent.name,
        agent.category,
        agent.description,
        ...(agent.suggestedPrompts ?? []),
      ].some((value) => value.toLowerCase().includes(normalizedSearch));
      const matchesCategory = selectedCategory === 'Recommended' || agent.category === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  }, [agents, searchTerm, selectedCategory]);

  const commonMessageCost = agents && agents.length > 0
    ? Math.max(1, Math.min(...agents.map((agent) => agent.creditCost?.textMessageCredits ?? 1)))
    : 1;

  const startWithPrompt = (agent: Agent, prompt: string) => {
    const params = new URLSearchParams({ prompt });
    navigate(`/chat/${agent.slug}?${params.toString()}`);
  };

  const handleRetry = () => {
    void refetchAgents();
    void refetchUsage();
  };

  return (
    <div className="space-y-8">
      <section className="rounded-[28px] border border-white/8 bg-dark-950/70 p-1 shadow-[0_30px_80px_-40px_rgba(0,0,0,0.9)]">
        <div className="rounded-[24px] border border-primary-500/10 bg-[radial-gradient(circle_at_top_left,rgba(139,92,246,0.18),transparent_35%),linear-gradient(180deg,rgba(17,24,39,0.96),rgba(2,6,23,0.92))] px-6 py-7 sm:px-8 sm:py-9 lg:px-10">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl">
              <div className="inline-flex items-center gap-2 rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-primary-200">
                <Sparkles className="h-3.5 w-3.5" />
                Agent Marketplace
              </div>
              <h1 className="mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl lg:text-5xl">
                Find the right AI agent for your next task.
              </h1>
              <p className="mt-4 max-w-2xl text-sm leading-7 text-dark-300 sm:text-base">
                Search practical business Agents, try a suggested prompt, and only pay credits after a successful response.
              </p>
              <label className="sr-only" htmlFor="agent-search">Search Agents</label>
              <div className="mt-6 flex max-w-2xl items-center gap-3 rounded-2xl border border-white/10 bg-white px-4 py-3 text-dark-950 shadow-xl shadow-black/20">
                <Search className="h-5 w-5 text-dark-400" />
                <input
                  id="agent-search"
                  type="search"
                  role="searchbox"
                  aria-label="Search Agents"
                  value={searchTerm}
                  onChange={(event) => setSearchTerm(event.target.value)}
                  placeholder="Search legal, tax, marketing, support..."
                  className="min-w-0 flex-1 bg-transparent text-sm text-dark-950 placeholder:text-dark-400 focus:outline-none"
                />
              </div>
            </div>

            <div className="min-w-full lg:min-w-[340px] lg:max-w-sm">
              <CreditExplainer
                balance={usageSummary ? availableCreditsValue : null}
                perMessageCost={commonMessageCost}
              />
            </div>
          </div>
        </div>
      </section>

      {sanitizedDashboardHtml ? (
        <section
          role="region"
          aria-label="Custom dashboard content"
          className="rounded-[26px] border border-white/8 bg-dark-950/80 p-6 text-dark-100"
          dangerouslySetInnerHTML={{ __html: sanitizedDashboardHtml }}
        />
      ) : null}

      {agentsError ? (
        <InlineErrorBanner
          message={getErrorMessage(agentsError)}
          onRetry={handleRetry}
          isRetrying={agentsFetching}
        />
      ) : null}

      {categories.length > 1 ? (
        <div className="flex flex-wrap gap-2">
          {categories.map((category) => (
            <button
              key={category}
              type="button"
              onClick={() => setSelectedCategory(category)}
              aria-pressed={selectedCategory === category}
              className={`rounded-full px-4 py-2 text-sm font-semibold transition-colors ${
                selectedCategory === category
                  ? 'bg-primary-500 text-white'
                  : 'border border-white/8 bg-dark-900/70 text-dark-300 hover:border-primary-400/30 hover:text-white'
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      ) : null}

      {agentsLoading || !tenantReady ? (
        <AgentGridSkeleton />
      ) : agents && agents.length > 0 && filteredAgents.length > 0 ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
          {filteredAgents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onStartChat={() => navigate(`/chat/${agent.slug}`)}
              onPromptClick={(prompt) => startWithPrompt(agent, prompt)}
            />
          ))}
        </div>
      ) : agents && agents.length > 0 ? (
        <EmptyState
          icon={Search}
          title="No Agents match your search"
          description="Try a different keyword or category to find the right Agent for your task."
          action={{ label: 'Clear filters', onClick: () => { setSearchTerm(''); setSelectedCategory('Recommended'); } }}
        />
      ) : agentsError ? null : (
        <EmptyExpertsState />
      )}
    </div>
  );
}

interface AgentCardProps {
  agent: Agent;
  onStartChat: () => void;
  onPromptClick: (prompt: string) => void;
}

function AgentCard({ agent, onStartChat, onPromptClick }: AgentCardProps) {
  const badge = getExpertBadge(agent);
  const ExpertIcon = getExpertIcon(agent);

  return (
    <div className="group rounded-[26px] border border-white/8 bg-gradient-to-br from-white/10 via-primary-500/5 to-transparent p-[1px] shadow-[0_24px_60px_-40px_rgba(0,0,0,0.95)] transition-transform duration-200 hover:-translate-y-1 hover:from-primary-400/30 hover:via-primary-500/10 hover:to-white/10">
      <div className="flex h-full flex-col rounded-[25px] bg-dark-950/95 p-6">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-4">
            <div
              className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/6"
              style={{ backgroundColor: `${agent.color}20` }}
            >
              <ExpertIcon className="h-6 w-6" style={{ color: agent.color }} />
            </div>
            <div className="min-w-0">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">
                {agent.category}
              </p>
              <h3 className="mt-2 text-xl font-semibold text-white">{agent.name}</h3>
            </div>
          </div>

          <span className={`rounded-full border px-3 py-1 text-xs font-semibold ${badge.className}`}>
            {badge.label}
          </span>
        </div>

        <p className="mt-5 text-sm leading-6 text-dark-300">{agent.description}</p>

        <p className="mt-3 text-xs font-semibold uppercase tracking-[0.18em] text-primary-300">
          Best for {agent.category} teams
        </p>

        <div className="mt-6 rounded-2xl border border-white/6 bg-dark-900/60 p-4">
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">
            Suggested prompts
          </p>
          <ul className="mt-3 space-y-2.5">
            {agent.suggestedPrompts?.slice(0, 3).map((example) => (
              <li key={example}>
                <button
                  type="button"
                  onClick={() => onPromptClick(example)}
                  className="flex w-full items-start gap-2 rounded-xl px-2 py-1.5 text-left text-sm text-dark-200 transition-colors hover:bg-dark-800 hover:text-white"
                  aria-label={`Try prompt: ${example}`}
                >
                  <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-primary-400" />
                  <span>{example}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="mt-6 flex items-center justify-between border-t border-white/6 pt-5">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 text-sm text-dark-300">
              <Zap className="h-4 w-4 text-primary-300" />
              <span>{agent.creditCost?.textMessageCredits ?? 1} credits/message</span>
            </div>
            {agent.capabilities?.includes('text_chat') && (
              <span className="text-xs text-dark-400">Text chat</span>
            )}
          </div>

          <button
            onClick={onStartChat}
            className="inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600"
            aria-label={`Start with ${agent.name}`}
          >
            Start
            <ArrowRight className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

function AgentGridSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 6 }).map((_, index) => (
        <div key={index} className="rounded-[26px] border border-white/8 bg-dark-950/80 p-6">
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-4">
              <SkeletonBlock className="h-12 w-12 rounded-2xl" />
              <div className="space-y-3">
                <SkeletonBlock className="h-3 w-16" />
                <SkeletonBlock className="h-6 w-40" />
              </div>
            </div>
            <SkeletonBlock className="h-7 w-20 rounded-full" />
          </div>

          <div className="mt-6 space-y-3">
            <SkeletonBlock className="h-4 w-full" />
            <SkeletonBlock className="h-4 w-5/6" />
          </div>

          <div className="mt-6 rounded-2xl border border-white/6 bg-dark-900/60 p-4">
            <SkeletonBlock className="h-3 w-28" />
            <div className="mt-4 space-y-3">
              <SkeletonBlock className="h-4 w-full" />
              <SkeletonBlock className="h-4 w-11/12" />
              <SkeletonBlock className="h-4 w-4/5" />
            </div>
          </div>

          <div className="mt-6 flex items-center justify-between border-t border-white/6 pt-5">
            <SkeletonBlock className="h-4 w-28" />
            <SkeletonBlock className="h-10 w-28 rounded-xl" />
          </div>
        </div>
      ))}
    </div>
  );
}

function InlineErrorBanner({ message, onRetry, isRetrying }: { message: string; onRetry: () => void; isRetrying: boolean }) {
  return (
    <ErrorState
      title="Unable to load agents right now."
      message={message}
      retryLabel="Retry"
      onRetry={onRetry}
      isRetrying={isRetrying}
    />
  );
}

function EmptyExpertsState() {
  return (
    <EmptyState
      icon={MessageCircle}
      title="No agents yet"
      description="This workspace has not published any Agents yet. Check back soon or contact your administrator to publish the first Agent."
    />
  );
}

function SkeletonBlock({ className }: { className: string }) {
  return <div className={`animate-pulse rounded-xl bg-dark-800 ${className}`} />;
}

function getExpertBadge(agent: Agent) {
  return expertBadges[agent.id] ?? {
    label: agent.category,
    className: 'border-white/10 bg-white/5 text-dark-200',
  };
}

function getExpertIcon(agent: Agent) {
  return expertIconMap[agent.id as keyof typeof expertIconMap] ?? MessageCircle;
}
