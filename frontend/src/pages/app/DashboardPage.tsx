import DOMPurify from 'dompurify';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Landmark, MessageCircle, PenLine, Scale, Sparkles, TowerControl, Wallet, Zap, Headphones, Globe2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { agentsApi, brandingApi, usageApi } from '../../api/client';
import { useTenant } from '../../contexts/TenantContext';
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
              <h1 className="mt-4 flex items-center gap-3 text-3xl font-bold text-white sm:text-4xl">
                <MessageCircle className="h-8 w-8 text-primary-300" />
                Launch your AgentStore
              </h1>
              <p className="mt-3 max-w-2xl text-sm leading-7 text-dark-300 sm:text-base">
                Choose a tenant-published agent, start a conversation, and turn specialist AI workflows into a product your team can sell and monetize.
              </p>
            </div>

            <div className="min-w-full lg:min-w-[320px] lg:max-w-sm">
              <div className="rounded-2xl border border-white/8 bg-dark-900/70 p-5 backdrop-blur-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.24em] text-dark-400">
                      Current balance
                    </p>
                    {usageLoading ? (
                      <div className="mt-3 space-y-2">
                        <SkeletonBlock className="h-7 w-40" />
                        <SkeletonBlock className="h-4 w-48" />
                      </div>
                    ) : usageSummary ? (
                      <>
                        <p className="mt-3 text-2xl font-semibold text-white">
                          {formatCredits(availableCreditsValue)} credits available
                        </p>
                        <p className="mt-2 text-sm text-dark-400">
                          {formatCredits(usageSummary.subscriptionCredits)} subscription + {formatCredits(usageSummary.purchasedCredits)} purchased
                        </p>
                      </>
                    ) : (
                      <p className="mt-3 text-sm text-dark-400">
                        Credits will appear here when usage data is available.
                      </p>
                    )}
                  </div>
                  <div className="rounded-2xl border border-primary-400/15 bg-primary-500/10 p-3 text-primary-200">
                    <Wallet className="h-5 w-5" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {sanitizedDashboardHtml ? (
        <section
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

      {agentsLoading || !tenantReady ? (
        <AgentGridSkeleton />
      ) : agents && agents.length > 0 ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
          {agents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onStartChat={() => navigate(`/chat/${agent.slug}`)}
            />
          ))}
        </div>
      ) : agentsError ? null : (
        <EmptyExpertsState />
      )}
    </div>
  );
}

interface AgentCardProps {
  agent: Agent;
  onStartChat: () => void;
}

function AgentCard({ agent, onStartChat }: AgentCardProps) {
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

        <div className="mt-6 rounded-2xl border border-white/6 bg-dark-900/60 p-4">
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">
            Suggested prompts
          </p>
          <ul className="mt-3 space-y-2.5">
            {agent.suggestedPrompts?.slice(0, 3).map((example, index) => (
              <li key={index} className="flex items-start gap-2 text-sm text-dark-200">
                <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-primary-400" />
                <span>{example}</span>
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
            {agent.capabilities?.includes('image_generation') && (
              <span className="text-xs text-dark-400">Image</span>
            )}
            {agent.capabilities?.includes('video_generation') && (
              <span className="text-xs text-dark-400">Video</span>
            )}
          </div>

          <button
            onClick={onStartChat}
            className="inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600"
          >
            Start Chat
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

function InlineErrorBanner({
  message,
  onRetry,
  isRetrying,
}: {
  message: string;
  onRetry: () => void;
  isRetrying: boolean;
}) {
  return (
    <div className="flex flex-col gap-4 rounded-2xl border border-red-500/25 bg-red-500/10 p-4 text-sm text-red-200 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="font-semibold text-red-100">Unable to load agents right now.</p>
        <p className="mt-1 text-red-200/80">{message}</p>
      </div>
      <button
        onClick={onRetry}
        disabled={isRetrying}
        className="inline-flex items-center justify-center rounded-xl border border-red-400/30 bg-red-500/10 px-4 py-2 font-semibold text-red-100 transition-colors hover:bg-red-500/20 disabled:cursor-not-allowed disabled:opacity-70"
      >
        {isRetrying ? 'Retrying...' : 'Retry'}
      </button>
    </div>
  );
}

function EmptyExpertsState() {
  return (
    <div className="rounded-[26px] border border-dashed border-white/10 bg-dark-950/70 px-6 py-14 text-center">
      <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-white/8 bg-dark-900/80 text-primary-300">
        <MessageCircle className="h-6 w-6" />
      </div>
      <h2 className="mt-5 text-xl font-semibold text-white">No agents yet</h2>
      <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-dark-400">
        Check back soon or contact your administrator to publish agents.
      </p>
    </div>
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

function formatCredits(value: number) {
  return new Intl.NumberFormat().format(value);
}
