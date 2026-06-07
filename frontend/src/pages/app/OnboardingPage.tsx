import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CheckCircle, ChevronRight, MessageCircle, Sparkles, Target, Zap } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../contexts/AuthContext';
import { authApi, agentsApi, usageApi } from '../../api/client';
import LoadingSpinner from '../../components/LoadingSpinner';
import type { Agent } from '../../types';

type Step = 'goal' | 'agent' | 'prompt' | 'credits';

const goals = ['Marketing', 'Legal', 'Tax', 'Customer support', 'E-commerce', 'Strategy', 'Other'];

export default function OnboardingPage() {
  const navigate = useNavigate();
  const { refreshUser } = useAuth();
  const [step, setStep] = useState<Step>('goal');
  const [selectedGoal, setSelectedGoal] = useState('');
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
  const [selectedPrompt, setSelectedPrompt] = useState('');
  const [loading, setLoading] = useState(false);

  const { data: agents = [], isLoading: agentsLoading } = useQuery({ queryKey: ['onboarding-agents'], queryFn: agentsApi.list, retry: false });
  const { data: usageSummary } = useQuery({ queryKey: ['onboarding-usage-summary'], queryFn: usageApi.summary, retry: false });

  const recommendedAgents = useMemo(() => {
    if (!selectedGoal || selectedGoal === 'Other') return agents.slice(0, 3);
    const normalizedGoal = selectedGoal.toLowerCase();
    const matches = agents.filter((agent) => [agent.category, agent.name, agent.description].some((value) => value.toLowerCase().includes(normalizedGoal)));
    return (matches.length > 0 ? matches : agents).slice(0, 3);
  }, [agents, selectedGoal]);

  const totalCredits = (usageSummary?.subscriptionCredits ?? 0) + (usageSummary?.purchasedCredits ?? 0);

  const navigateWithFallback = async (destination: string) => {
    setLoading(true);
    try {
      await authApi.completeOnboarding();
      try {
        await refreshUser();
      } catch {
        // completeOnboarding succeeded but refreshUser failed — hard navigate to avoid stale state
        window.location.href = destination;
        return;
      }
      navigate(destination);
    } catch {
      // completeOnboarding itself failed — still try to let the user through via soft navigate
      setLoading(false);
      navigate(destination);
    }
  };

  const goToDestination = () => {
    const destination = selectedAgent
      ? `/chat/${selectedAgent.slug}${selectedPrompt ? `?${new URLSearchParams({ prompt: selectedPrompt }).toString()}` : ''}`
      : '/dashboard';
    return navigateWithFallback(destination);
  };

  const goToMarketplace = () => navigateWithFallback('/dashboard');

  const chooseGoal = (goal: string) => {
    setSelectedGoal(goal);
    setStep('agent');
  };

  const chooseAgent = (agent: Agent) => {
    setSelectedAgent(agent);
    setSelectedPrompt(agent.suggestedPrompts?.[0] ?? '');
    setStep('prompt');
  };

  return (
    <div className="min-h-screen bg-dark-950 px-4 py-10">
      <div className="mx-auto w-full max-w-3xl">
        <div className="mb-8 text-center">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-500/15 text-primary-300">
            <Sparkles className="h-7 w-7" />
          </div>
          <h1 className="mt-5 text-3xl font-bold text-white">Welcome to AgentStore</h1>
          <p className="mt-2 text-sm text-dark-400">Choose a goal, pick an Agent, and start with a useful prompt.</p>
        </div>

        <div className="mb-8 flex items-center justify-center gap-2">
          {(['goal', 'agent', 'prompt', 'credits'] as Step[]).map((item, index) => {
            const labels: Record<Step, string> = { goal: 'Goal', agent: 'Agent', prompt: 'Prompt', credits: 'Credits' };
            const activeIndex = ['goal', 'agent', 'prompt', 'credits'].indexOf(step);
            const isActive = index === activeIndex;
            return (
              <div key={item} className="flex items-center gap-2">
                <div
                  className={`rounded-full px-3 py-1.5 text-sm font-medium ${isActive ? 'bg-primary-500/20 text-primary-300' : 'bg-dark-800 text-dark-500'}`}
                  aria-current={isActive ? 'step' : undefined}
                >
                  {labels[item]}
                </div>
                {index < 3 ? <ChevronRight className="h-4 w-4 text-dark-600" /> : null}
              </div>
            );
          })}
        </div>

        {step === 'goal' && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="flex items-center gap-2 text-xl font-bold text-white"><Target className="h-5 w-5 text-primary-300" />What do you want to accomplish first?</h2>
            <p className="mt-2 text-sm text-dark-400">We will recommend a practical Agent and a first prompt.</p>
            <div className="mt-6 grid gap-3 sm:grid-cols-2">
              {goals.map((goal) => (
                <button key={goal} type="button" onClick={() => chooseGoal(goal)} className="rounded-2xl border border-dark-800 bg-dark-950/60 px-4 py-4 text-left font-semibold text-white transition-colors hover:border-primary-500/40 hover:bg-dark-900">
                  {goal}
                </button>
              ))}
            </div>
          </section>
        )}

        {step === 'agent' && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="flex items-center gap-2 text-xl font-bold text-white"><MessageCircle className="h-5 w-5 text-primary-300" />Pick your first Agent</h2>
            <p className="mt-2 text-sm text-dark-400">
              {selectedGoal && selectedGoal !== 'Other' ? `Recommended for ${selectedGoal}` : 'Suggested Agents to get started'}
            </p>

            {agentsLoading ? (
              <div className="mt-8 flex justify-center">
                <LoadingSpinner />
              </div>
            ) : recommendedAgents.length === 0 ? (
              <div className="mt-8 rounded-2xl border border-dark-800 bg-dark-950/60 p-6 text-center">
                <p className="text-dark-400">No Agents are published yet</p>
                <button
                  type="button"
                  onClick={goToMarketplace}
                  disabled={loading}
                  className="mt-4 inline-flex items-center gap-2 rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600 disabled:opacity-60"
                >
                  Go to marketplace
                </button>
              </div>
            ) : (
              <div className="mt-6 grid gap-4 sm:grid-cols-2">
                {recommendedAgents.map((agent) => (
                  <button
                    key={agent.id}
                    type="button"
                    onClick={() => chooseAgent(agent)}
                    aria-label={`Choose ${agent.name}`}
                    className="rounded-2xl border border-dark-800 bg-dark-950/60 p-4 text-left transition-colors hover:border-primary-500/40 hover:bg-dark-900"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-lg font-semibold text-white">{agent.name}</span>
                      <span className="rounded-full bg-primary-500/10 px-2 py-0.5 text-xs font-medium text-primary-300">{agent.category}</span>
                    </div>
                    <p className="mt-2 text-sm text-dark-400 line-clamp-2">{agent.description}</p>
                    <div className="mt-3 flex flex-wrap gap-1">
                      {agent.suggestedPrompts?.slice(0, 2).map((prompt, i) => (
                        <span key={i} className="rounded-lg bg-dark-800 px-2 py-1 text-xs text-dark-300">{prompt}</span>
                      ))}
                    </div>
                  </button>
                ))}
              </div>
            )}

            <button
              type="button"
              onClick={() => setStep('goal')}
              className="mt-6 text-sm text-dark-400 hover:text-dark-300"
            >
              ← Choose a different goal
            </button>
          </section>
        )}

        {step === 'prompt' && selectedAgent && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="flex items-center gap-2 text-xl font-bold text-white"><Sparkles className="h-5 w-5 text-primary-300" />Choose your first prompt</h2>
            <p className="mt-2 text-sm text-dark-400">
              Or type your own message after starting the chat.
            </p>

            <div className="mt-6 space-y-3">
              {selectedAgent.suggestedPrompts?.map((prompt, index) => (
                <button
                  key={index}
                  type="button"
                  onClick={() => {
                    setSelectedPrompt(prompt);
                    setStep('credits');
                  }}
                  aria-label={prompt}
                  className={`w-full rounded-xl border px-4 py-3 text-left transition-colors ${
                    selectedPrompt === prompt
                      ? 'border-primary-500 bg-primary-500/10 text-white'
                      : 'border-dark-800 bg-dark-950/60 text-dark-200 hover:border-primary-500/40 hover:bg-dark-900'
                  }`}
                >
                  {prompt}
                </button>
              ))}
            </div>

            <div className="mt-6 flex items-center gap-3">
              <button
                type="button"
                onClick={() => {
                  setSelectedPrompt('');
                  setStep('credits');
                }}
                className="text-sm text-dark-400 hover:text-dark-300"
              >
                Skip prompt
              </button>
              <span className="h-px flex-1 bg-dark-800" />
              <button
                type="button"
                onClick={() => setStep('agent')}
                className="text-sm text-dark-400 hover:text-dark-300"
              >
                ← Choose a different Agent
              </button>
            </div>
          </section>
        )}

        {step === 'credits' && (
          <section className="rounded-3xl border border-dark-800 bg-dark-900/60 p-6">
            <h2 className="flex items-center gap-2 text-xl font-bold text-white"><Zap className="h-5 w-5 text-primary-300" />You're ready to start!</h2>
            <p className="mt-2 text-sm text-dark-400">
              You start with <span className="font-semibold text-primary-300">{totalCredits.toLocaleString()} credits.</span>
            </p>

            <div className="mt-6 rounded-2xl border border-accent-emerald/20 bg-accent-emerald/10 p-4">
              <div className="flex items-center gap-3">
                <CheckCircle className="h-5 w-5 text-accent-emerald" />
                <p className="text-sm text-accent-emerald">Credits are charged only after a successful response.</p>
              </div>
            </div>

            {selectedAgent && (
              <div className="mt-6 rounded-2xl border border-dark-800 bg-dark-950/60 p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-dark-500">Your first chat</p>
                <p className="mt-2 font-medium text-white">
                  {selectedAgent.name}
                  {selectedPrompt && <span className="text-dark-400"> — "{selectedPrompt}"</span>}
                </p>
              </div>
            )}

            <div className="mt-6 flex gap-3">
              <button
                type="button"
                onClick={() => setStep('prompt')}
                className="flex-1 rounded-xl border border-dark-700 bg-dark-900 px-4 py-3 font-semibold text-dark-200 transition-colors hover:bg-dark-800"
              >
                Back
              </button>
              <button
                type="button"
                onClick={goToDestination}
                disabled={loading}
                className="flex-1 rounded-xl bg-primary-500 px-4 py-3 font-semibold text-white transition-colors hover:bg-primary-600 disabled:opacity-60"
              >
                {loading ? <LoadingSpinner size="sm" /> : 'Start chatting'}
              </button>
            </div>
          </section>
        )}
      </div>
    </div>
  );
}