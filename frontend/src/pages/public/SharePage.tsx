import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ArrowRight, MessageCircle } from 'lucide-react';
import { shareApi } from '../../api/client';
import { useAuth } from '../../contexts/AuthContext';
import { LanguageSwitcher } from '../../components/LanguageSwitcher';
import { MarkdownMessage } from '../../components/app';
import LoadingSpinner from '../../components/LoadingSpinner';

export default function SharePage() {
  const { token } = useParams<{ token: string }>();
  const { isAuthenticated } = useAuth();
  const { t } = useTranslation('app');

  const { data: share, isLoading, isError } = useQuery({
    queryKey: ['share', token],
    queryFn: () => shareApi.getPublic(token!),
    enabled: !!token,
    staleTime: 60 * 1000,
    retry: false,
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-dark-950 flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (isError || !share) {
    return (
      <div className="min-h-screen bg-dark-950 flex items-center justify-center text-center px-4">
        <div>
          <MessageCircle className="mx-auto mb-4 h-12 w-12 text-dark-600" />
          <h1 className="text-xl font-bold text-white mb-2">Conversation not found</h1>
          <p className="text-dark-400 mb-6 text-sm">This shared conversation may have been revoked or the link is invalid.</p>
          <Link to="/agents" className="text-primary-400 hover:text-primary-300 transition-colors text-sm">
            Browse AI agents →
          </Link>
        </div>
      </div>
    );
  }

  const ctaHref = isAuthenticated
    ? `/chat/${share.agentId}`
    : `/signup?redirect=${encodeURIComponent(`/chat/${share.agentId}`)}`;

  return (
    <div className="min-h-screen bg-dark-950">
      {/* Header */}
      <header className="sticky top-0 z-40 bg-dark-900/80 backdrop-blur-xl border-b border-dark-800">
        <div className="max-w-3xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link to="/" className="font-semibold text-white">AgentStore</Link>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            {!isAuthenticated && (
              <Link to="/signup" className="px-4 py-2 text-sm font-medium bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors">
                {t('login.signUp', { ns: 'auth' })}
              </Link>
            )}
          </div>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 sm:px-6 py-8">
        {/* Agent banner */}
        <div className="mb-6 flex items-center gap-3 rounded-2xl border border-white/8 bg-dark-900/60 px-4 py-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary-500/20">
            <MessageCircle className="h-5 w-5 text-primary-300" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-xs text-dark-400">Conversation with</p>
            <p className="font-semibold text-white truncate">{share.agentName}</p>
          </div>
          {share.title && (
            <p className="hidden sm:block text-sm text-dark-400 italic truncate max-w-[200px]">"{share.title}"</p>
          )}
        </div>

        {/* Messages */}
        <div className="space-y-4 mb-8">
          {share.messages.map((msg, i) => (
            <div key={i} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-[85%] rounded-2xl px-4 py-3 sm:max-w-[75%] ${
                  msg.role === 'user'
                    ? 'rounded-br-md bg-primary-500 text-white'
                    : 'rounded-bl-md bg-dark-800 text-dark-100'
                }`}
              >
                {msg.role === 'assistant' ? (
                  <MarkdownMessage content={msg.content} />
                ) : (
                  <p className="whitespace-pre-wrap break-words text-sm leading-6">{msg.content}</p>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* CTA */}
        <div className="rounded-2xl border border-primary-500/20 bg-primary-500/5 p-6 text-center">
          <p className="text-white font-semibold mb-2">
            {isAuthenticated ? `Chat with ${share.agentName}` : `Try ${share.agentName} yourself`}
          </p>
          <p className="text-dark-400 text-sm mb-4">
            {isAuthenticated
              ? 'Continue this conversation or start a new one.'
              : 'Create a free account and start chatting with AI agents.'}
          </p>
          <Link
            to={ctaHref}
            className="inline-flex items-center gap-2 rounded-xl bg-primary-500 px-6 py-3 font-semibold text-white transition-colors hover:bg-primary-600"
          >
            {isAuthenticated ? 'Start chatting' : 'Get started free'}
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </main>
    </div>
  );
}
