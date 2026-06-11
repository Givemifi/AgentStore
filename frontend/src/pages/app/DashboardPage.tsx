import { useMemo, useState } from 'react';
import DOMPurify from 'dompurify';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Landmark, MessageCircle, PenLine, Scale, Search, TowerControl, Zap, Headphones, Globe2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation('app');
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
  const [selectedCategory, setSelectedCategory] = useState(() => t('dashboard.recommended'));

  const categories = useMemo(() => {
    const uniqueCategories = Array.from(new Set((agents ?? []).map((agent) => agent.category).filter(Boolean))).sort();
    return [t('dashboard.recommended'), ...uniqueCategories];
  }, [agents, t]);

  const filteredAgents = useMemo(() => {
    const normalizedSearch = searchTerm.trim().toLowerCase();
    return (agents ?? []).filter((agent) => {
      const matchesSearch = normalizedSearch === '' || [
        agent.name,
        agent.category,
        agent.description,
        ...(agent.suggestedPrompts ?? []),
      ].some((value) => value.toLowerCase().includes(normalizedSearch));
      const matchesCategory = selectedCategory === t('dashboard.recommended') || agent.category === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  }, [agents, searchTerm, selectedCategory, t]);

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
    <div className="space-y-6">
      {/* Page header — Linear style: title + description + search, flat */}
      <section className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div className="max-w-2xl flex-1">
          <h1 className="text-[15px] font-semibold text-white">
            {t('dashboard.marketplace')}
          </h1>
          <p className="mt-1 text-[13px] text-dark-400">
            {t('dashboard.subtext')}
          </p>
          <label className="sr-only" htmlFor="agent-search">{t('dashboard.searchLabel')}</label>
          <div className="mt-4 flex max-w-xl items-center gap-2 rounded-md border border-white/8 bg-dark-800 px-3 h-9 transition-colors focus-within:border-primary-500">
            <Search className="h-4 w-4 text-dark-500 shrink-0" />
            <input
              id="agent-search"
              type="search"
              role="searchbox"
              aria-label={t('dashboard.searchLabel')}
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
              placeholder={t('dashboard.searchPlaceholder')}
              className="min-w-0 flex-1 bg-transparent text-[13px] text-white placeholder:text-dark-500 focus:outline-none"
            />
          </div>
        </div>

        <div className="w-full lg:w-[300px] shrink-0">
          <CreditExplainer
            balance={usageSummary ? availableCreditsValue : null}
            perMessageCost={commonMessageCost}
          />
        </div>
      </section>

      {sanitizedDashboardHtml ? (
        <section
          role="region"
          aria-label="Custom dashboard content"
          className="rounded-lg border border-white/8 bg-dark-900 p-5 text-dark-100"
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
        <div className="flex flex-wrap gap-1.5">
          {categories.map((category) => (
            <button
              key={category}
              type="button"
              onClick={() => setSelectedCategory(category)}
              aria-pressed={selectedCategory === category}
              className={`rounded-md px-2.5 h-7 text-[13px] font-medium transition-colors ${
                selectedCategory === category
                  ? 'bg-primary-500 text-white'
                  : 'border border-white/8 text-dark-400 hover:border-white/14 hover:text-white'
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
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
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
          title={t('dashboard.noAgentsSearch')}
          description={t('dashboard.noAgentsSearchDesc')}
          action={{ label: t('dashboard.clearFilters'), onClick: () => { setSearchTerm(''); setSelectedCategory(t('dashboard.recommended')); } }}
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
  const { t } = useTranslation('app');
  const badge = getExpertBadge(agent);
  const ExpertIcon = getExpertIcon(agent);

  return (
    <div className="group flex h-full flex-col rounded-lg border border-white/8 bg-dark-900 p-4 transition-colors hover:border-white/14">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <div
            className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md"
            style={{ backgroundColor: `${agent.color}1a` }}
          >
            <ExpertIcon className="h-4 w-4" style={{ color: agent.color }} />
          </div>
          <div className="min-w-0">
            <h3 className="text-[13px] font-semibold text-white">{agent.name}</h3>
            <p className="mt-0.5 text-xs text-dark-500">{agent.category}</p>
          </div>
        </div>

        <span className={`rounded-md border px-1.5 py-0.5 text-[11px] font-medium ${badge.className}`}>
          {badge.label}
        </span>
      </div>

      <p className="mt-3 text-[13px] leading-5 text-dark-400 line-clamp-2">{agent.description}</p>

      <div className="mt-3 space-y-0.5">
        {agent.suggestedPrompts?.slice(0, 2).map((example) => (
          <button
            key={example}
            type="button"
            onClick={() => onPromptClick(example)}
            className="flex w-full items-center gap-1.5 rounded-md px-1.5 py-1 text-left text-xs text-dark-400 transition-colors hover:bg-white/5 hover:text-white"
            aria-label={`Try prompt: ${example}`}
          >
            <span className="h-1 w-1 shrink-0 rounded-full bg-primary-400" />
            <span className="truncate">{example}</span>
          </button>
        ))}
      </div>

      <div className="mt-auto flex items-center justify-between border-t border-white/8 pt-3 mt-4">
        <div className="flex items-center gap-1.5 text-xs text-dark-400">
          <Zap className="h-3 w-3 text-primary-400" />
          <span>{t('dashboard.creditsPerMessage', { count: agent.creditCost?.textMessageCredits ?? 1 })}</span>
        </div>

        <button
          onClick={onStartChat}
          className="inline-flex h-7 items-center gap-1 rounded-md bg-primary-500 px-2.5 text-xs font-medium text-white transition-colors hover:bg-primary-600"
          aria-label={`Start with ${agent.name}`}
        >
          {t('dashboard.start')}
          <ArrowRight className="h-3 w-3" />
        </button>
      </div>
    </div>
  );
}

function AgentGridSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 6 }).map((_, index) => (
        <div key={index} className="rounded-lg border border-white/8 bg-dark-900 p-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-start gap-3">
              <SkeletonBlock className="h-8 w-8 rounded-md" />
              <div className="space-y-2">
                <SkeletonBlock className="h-3.5 w-28" />
                <SkeletonBlock className="h-3 w-16" />
              </div>
            </div>
            <SkeletonBlock className="h-5 w-14 rounded-md" />
          </div>

          <div className="mt-4 space-y-2">
            <SkeletonBlock className="h-3.5 w-full" />
            <SkeletonBlock className="h-3.5 w-5/6" />
          </div>

          <div className="mt-4 space-y-1.5">
            <SkeletonBlock className="h-3 w-full" />
            <SkeletonBlock className="h-3 w-11/12" />
          </div>

          <div className="mt-4 flex items-center justify-between border-t border-white/8 pt-3">
            <SkeletonBlock className="h-3 w-24" />
            <SkeletonBlock className="h-7 w-16 rounded-md" />
          </div>
        </div>
      ))}
    </div>
  );
}

function InlineErrorBanner({ message, onRetry, isRetrying }: { message: string; onRetry: () => void; isRetrying: boolean }) {
  const { t } = useTranslation('app');
  return (
    <ErrorState
      title={t('dashboard.unableToLoad')}
      message={message}
      retryLabel={t('retry', { ns: 'common' })}
      onRetry={onRetry}
      isRetrying={isRetrying}
    />
  );
}

function EmptyExpertsState() {
  const { t } = useTranslation('app');
  return (
    <EmptyState
      icon={MessageCircle}
      title={t('dashboard.noAgentsYet')}
      description={t('dashboard.noAgentsYetDesc')}
    />
  );
}

function SkeletonBlock({ className }: { className: string }) {
  return <div className={`animate-pulse rounded-md bg-dark-800 ${className}`} />;
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
