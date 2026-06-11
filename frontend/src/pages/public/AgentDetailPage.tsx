import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, MessageCircle, Zap } from 'lucide-react';
import { publicAgentsApi } from '../../api/client';
import { useAuth } from '../../contexts/AuthContext';
import { LanguageSwitcher } from '../../components/LanguageSwitcher';
import LoadingSpinner from '../../components/LoadingSpinner';

export default function AgentDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation('app');
  const { isAuthenticated } = useAuth();

  const { data: agent, isLoading, isError } = useQuery({
    queryKey: ['public-agent', slug],
    queryFn: () => publicAgentsApi.get(slug!),
    enabled: !!slug,
    staleTime: 5 * 60 * 1000,
  });

  const handleStart = () => {
    const target = `/chat/${agent?.slug || agent?.id}`;
    if (isAuthenticated) {
      navigate(target);
    } else {
      navigate(`/signup?redirect=${encodeURIComponent(target)}`);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-dark-950 flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (isError || !agent) {
    return (
      <div className="min-h-screen bg-dark-950 flex items-center justify-center text-center px-4">
        <div>
          <MessageCircle className="mx-auto mb-4 h-12 w-12 text-dark-600" />
          <h1 className="text-xl font-bold text-white mb-2">Agent not found</h1>
          <Link to="/agents" className="text-primary-400 hover:text-primary-300 transition-colors text-sm">
            ← Browse all agents
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-dark-950">
      {/* Minimal header */}
      <header className="sticky top-0 z-40 bg-dark-900/80 border-b border-white/8">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
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

      <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        {/* Back link */}
        <Link
          to="/agents"
          className="inline-flex items-center gap-2 text-sm text-dark-400 hover:text-white transition-colors mb-8"
        >
          <ArrowLeft className="h-4 w-4" />
          Browse all agents
        </Link>

        {/* Agent hero */}
        <div className="rounded-lg border border-white/8 bg-dark-900/60 p-8">
          <div className="flex items-start gap-5">
            <div
              className="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-xl border border-white/6"
              style={{ backgroundColor: `${agent.color || '#7C3AED'}20` }}
            >
              <MessageCircle className="h-8 w-8" style={{ color: agent.color || '#7C3AED' }} />
            </div>
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-dark-400">{agent.category}</p>
              <h1 className="mt-1 text-2xl font-bold text-white sm:text-3xl">{agent.name}</h1>
              <p className="mt-2 text-dark-300">{agent.description}</p>
            </div>
          </div>

          {/* Credit cost */}
          <div className="mt-6 flex items-center gap-3 rounded-xl border border-white/6 bg-dark-950/60 px-4 py-3 w-fit">
            <Zap className="h-4 w-4 text-primary-300" />
            <span className="text-sm text-dark-300">
              {t('dashboard.creditsPerMessage', { count: agent.creditCost?.textMessageCredits ?? 1 })}
            </span>
          </div>

          {/* Welcome message */}
          {agent.welcomeMessage && (
            <div className="mt-6 rounded-xl border border-primary-500/20 bg-primary-500/5 p-4">
              <p className="text-sm text-primary-200 italic">"{agent.welcomeMessage}"</p>
            </div>
          )}

          {/* Suggested prompts */}
          {agent.suggestedPrompts && agent.suggestedPrompts.length > 0 && (
            <div className="mt-8">
              <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-dark-400 mb-4">
                {t('dashboard.suggestedPrompts')}
              </h2>
              <div className="space-y-3">
                {agent.suggestedPrompts.map((prompt, i) => (
                  <button
                    key={i}
                    type="button"
                    onClick={handleStart}
                    className="w-full rounded-xl border border-white/8 bg-dark-900 px-4 py-3 text-left text-sm text-dark-200 transition-colors hover:border-primary-500/30 hover:text-white"
                  >
                    {prompt}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* CTA */}
          <div className="mt-8 pt-6 border-t border-white/6">
            <button
              type="button"
              onClick={handleStart}
              className="w-full rounded-xl bg-primary-500 px-6 py-3.5 text-base font-semibold text-white transition-colors hover:bg-primary-600"
            >
              {isAuthenticated ? `Start chatting with ${agent.name}` : `Sign up to chat with ${agent.name}`}
            </button>
            {!isAuthenticated && (
              <p className="mt-3 text-center text-sm text-dark-400">
                Already have an account?{' '}
                <Link to={`/login?redirect=${encodeURIComponent(`/chat/${agent.slug || agent.id}`)}`} className="text-primary-400 hover:text-primary-300 transition-colors">
                  Sign in
                </Link>
              </p>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
