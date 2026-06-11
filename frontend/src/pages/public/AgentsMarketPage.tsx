import { useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ArrowRight, MessageCircle, Search, Sparkles, Zap } from 'lucide-react';
import { publicAgentsApi } from '../../api/client';
import { useAuth } from '../../contexts/AuthContext';
import { LanguageSwitcher } from '../../components/LanguageSwitcher';
import { PublicFooter } from '../../components/PublicFooter';
import type { Agent } from '../../types';

export default function AgentsMarketPage() {
  const { t } = useTranslation('app');
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState(t('dashboard.recommended'));

  const { data: agents = [], isLoading } = useQuery({
    queryKey: ['public-agents'],
    queryFn: publicAgentsApi.list,
    staleTime: 5 * 60 * 1000,
  });

  const categories = useMemo(() => {
    const unique = Array.from(new Set(agents.map(a => a.category).filter(Boolean))).sort();
    return [t('dashboard.recommended'), ...unique];
  }, [agents, t]);

  const filtered = useMemo(() => {
    const q = searchTerm.trim().toLowerCase();
    return agents.filter(a => {
      const matchSearch = q === '' || [a.name, a.category, a.description].some(v => v?.toLowerCase().includes(q));
      const matchCat = selectedCategory === t('dashboard.recommended') || a.category === selectedCategory;
      return matchSearch && matchCat;
    });
  }, [agents, searchTerm, selectedCategory, t]);

  const handleStart = (agent: Agent) => {
    if (isAuthenticated) {
      navigate(`/chat/${agent.slug || agent.id}`);
    } else {
      navigate(`/signup?redirect=/chat/${agent.slug || agent.id}`);
    }
  };

  return (
    <div className="min-h-screen bg-dark-950">
      {/* Minimal header */}
      <header className="sticky top-0 z-40 bg-dark-900/80 backdrop-blur-xl border-b border-dark-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <Link to="/" className="font-semibold text-white text-lg">AgentStore</Link>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            {isAuthenticated ? (
              <Link to="/dashboard" className="text-sm text-primary-400 hover:text-primary-300 transition-colors">
                {t('nav.agents')}
              </Link>
            ) : (
              <div className="flex items-center gap-2">
                <Link to="/login" className="text-sm text-dark-400 hover:text-white transition-colors">
                  {t('login.signIn', { ns: 'auth' })}
                </Link>
                <Link to="/signup" className="px-4 py-2 text-sm font-medium bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors">
                  {t('login.signUp', { ns: 'auth' })}
                </Link>
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        {/* Hero search */}
        <section className="rounded-[28px] border border-white/8 bg-dark-950/70 p-1 shadow-[0_30px_80px_-40px_rgba(0,0,0,0.9)] mb-10">
          <div className="rounded-[24px] border border-primary-500/10 bg-[radial-gradient(circle_at_top_left,rgba(139,92,246,0.18),transparent_35%),linear-gradient(180deg,rgba(17,24,39,0.96),rgba(2,6,23,0.92))] px-6 py-8 sm:px-10 sm:py-10">
            <div className="inline-flex items-center gap-2 rounded-full border border-primary-400/20 bg-primary-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-primary-200">
              <Sparkles className="h-3.5 w-3.5" />
              {t('dashboard.marketplace')}
            </div>
            <h1 className="mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl lg:text-5xl">
              {t('dashboard.headline')}
            </h1>
            <p className="mt-4 max-w-2xl text-sm leading-7 text-dark-300 sm:text-base">
              {t('dashboard.subtext')}
            </p>
            <div className="mt-6 flex max-w-2xl items-center gap-3 rounded-2xl border border-white/10 bg-white px-4 py-3 text-dark-950 shadow-xl shadow-black/20">
              <Search className="h-5 w-5 text-dark-400 flex-shrink-0" />
              <input
                type="search"
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                placeholder={t('dashboard.searchPlaceholder')}
                aria-label={t('dashboard.searchLabel')}
                className="min-w-0 flex-1 bg-transparent text-sm text-dark-950 placeholder:text-dark-400 focus:outline-none"
              />
            </div>
          </div>
        </section>

        {/* Category chips */}
        {categories.length > 1 && (
          <div className="flex flex-wrap gap-2 mb-8">
            {categories.map(cat => (
              <button
                key={cat}
                type="button"
                onClick={() => setSelectedCategory(cat)}
                aria-pressed={selectedCategory === cat}
                className={`rounded-full px-4 py-2 text-sm font-semibold transition-colors ${
                  selectedCategory === cat
                    ? 'bg-primary-500 text-white'
                    : 'border border-white/8 bg-dark-900/70 text-dark-300 hover:border-primary-400/30 hover:text-white'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>
        )}

        {/* Agent grid */}
        {isLoading ? (
          <AgentGridSkeleton />
        ) : filtered.length > 0 ? (
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
            {filtered.map(agent => (
              <PublicAgentCard
                key={agent.id}
                agent={agent}
                onStart={() => handleStart(agent)}
                onViewDetail={() => navigate(`/agents/${agent.slug || agent.id}`)}
              />
            ))}
          </div>
        ) : (
          <div className="py-20 text-center">
            <MessageCircle className="mx-auto mb-4 h-12 w-12 text-dark-600" />
            <p className="text-dark-400">{searchTerm ? t('dashboard.noAgentsSearch') : t('dashboard.noAgentsYet')}</p>
            {searchTerm && (
              <button
                type="button"
                onClick={() => { setSearchTerm(''); setSelectedCategory(t('dashboard.recommended')); }}
                className="mt-3 text-sm text-primary-400 hover:text-primary-300 transition-colors"
              >
                {t('dashboard.clearFilters')}
              </button>
            )}
          </div>
        )}
      </main>
      <PublicFooter />
    </div>
  );
}

function PublicAgentCard({ agent, onStart, onViewDetail }: { agent: Agent; onStart: () => void; onViewDetail: () => void }) {
  const { t } = useTranslation('app');
  return (
    <div className="group rounded-[26px] border border-white/8 bg-gradient-to-br from-white/10 via-primary-500/5 to-transparent p-[1px] shadow-[0_24px_60px_-40px_rgba(0,0,0,0.95)] transition-transform duration-200 hover:-translate-y-1 hover:from-primary-400/30 hover:via-primary-500/10 hover:to-white/10">
      <div className="flex h-full flex-col rounded-[25px] bg-dark-950/95 p-6">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-4">
            <div
              className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/6"
              style={{ backgroundColor: `${agent.color || '#7C3AED'}20` }}
            >
              <MessageCircle className="h-6 w-6" style={{ color: agent.color || '#7C3AED' }} />
            </div>
            <div className="min-w-0">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">{agent.category}</p>
              <h3 className="mt-2 text-xl font-semibold text-white">{agent.name}</h3>
            </div>
          </div>
          <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-semibold text-dark-200">
            {agent.category}
          </span>
        </div>

        <p className="mt-5 text-sm leading-6 text-dark-300">{agent.description}</p>

        {agent.suggestedPrompts && agent.suggestedPrompts.length > 0 && (
          <div className="mt-6 rounded-2xl border border-white/6 bg-dark-900/60 p-4">
            <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">
              {t('dashboard.suggestedPrompts')}
            </p>
            <ul className="mt-3 space-y-2.5">
              {agent.suggestedPrompts.slice(0, 3).map((prompt, i) => (
                <li key={i}>
                  <button
                    type="button"
                    onClick={() => onStart()}
                    className="flex w-full items-start gap-2 rounded-xl px-2 py-1.5 text-left text-sm text-dark-200 transition-colors hover:bg-dark-800 hover:text-white"
                  >
                    <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-primary-400 flex-shrink-0" />
                    <span>{prompt}</span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="mt-6 flex items-center justify-between border-t border-white/6 pt-5">
          <div className="flex items-center gap-2 text-sm text-dark-300">
            <Zap className="h-4 w-4 text-primary-300" />
            <span>{t('dashboard.creditsPerMessage', { count: agent.creditCost?.textMessageCredits ?? 1 })}</span>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={onViewDetail}
              className="text-xs text-dark-400 hover:text-dark-200 transition-colors"
            >
              Details
            </button>
            <button
              onClick={onStart}
              className="inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600"
            >
              {t('dashboard.start')}
              <ArrowRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function AgentGridSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="rounded-[26px] border border-white/8 bg-dark-950/80 p-6 animate-pulse">
          <div className="flex items-start gap-4">
            <div className="h-12 w-12 rounded-2xl bg-dark-800" />
            <div className="space-y-2 flex-1">
              <div className="h-3 w-16 rounded bg-dark-800" />
              <div className="h-5 w-32 rounded bg-dark-800" />
            </div>
          </div>
          <div className="mt-6 space-y-2">
            <div className="h-4 w-full rounded bg-dark-800" />
            <div className="h-4 w-3/4 rounded bg-dark-800" />
          </div>
        </div>
      ))}
    </div>
  );
}
