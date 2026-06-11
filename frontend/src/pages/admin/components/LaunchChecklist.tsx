import { Link } from 'react-router-dom';
import { AlertTriangle, CheckCircle, ChevronRight, CircleDot } from 'lucide-react';
import type { LaunchReadinessItem, LaunchReadinessStatus } from '../../../types';

export type { LaunchReadinessItem };

// Integration display name mapping - used by DashboardPage for integration warnings
export const INTEGRATION_DISPLAY_NAMES: Record<string, string> = {
  stripe: 'Stripe',
  resend: 'Resend',
  mongodb: 'MongoDB',
  google_oauth: 'Google Login',
  github_oauth: 'GitHub Login',
  microsoft_oauth: 'Microsoft Login',
  webauthn: 'Passkeys',
  saml_sso: 'SSO/SAML',
};

const statusConfig: Record<LaunchReadinessStatus, { label: string; className: string; icon: typeof CheckCircle }> = {
  complete: { label: 'Complete', className: 'border-accent-emerald/20 bg-accent-emerald/10 text-accent-emerald', icon: CheckCircle },
  warning: { label: 'Needs attention', className: 'border-yellow-500/20 bg-yellow-500/10 text-yellow-400', icon: AlertTriangle },
  pending: { label: 'Review', className: 'border-white/8 bg-dark-800 text-dark-300', icon: CircleDot },
};

interface LaunchChecklistProps {
  items: LaunchReadinessItem[];
}

export default function LaunchChecklist({ items }: LaunchChecklistProps) {
  const completedCount = items.filter((item) => item.status === 'complete').length;

  return (
    <section className="mb-8 rounded-xl border border-white/8 bg-dark-900/60 p-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-xl font-bold text-white">Launch Checklist</h2>
          <p className="mt-1 text-sm text-dark-400">Know exactly what remains before users can rely on AgentStore.</p>
        </div>
        <p className="text-sm font-semibold text-primary-300">{completedCount}/{items.length} ready</p>
      </div>

      <div className="mt-5 grid gap-3 lg:grid-cols-2">
        {items.map((item) => {
          const config = statusConfig[item.status];
          const Icon = config.icon;
          return (
            <Link key={item.id} to={item.actionPath} className="group rounded-xl border border-white/8 bg-dark-950/50 p-4 transition-colors hover:border-primary-500/30 hover:bg-dark-900">
              <div className="flex items-start gap-3">
                <div className={`rounded-xl border p-2 ${config.className}`}>
                  <Icon className="h-4 w-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="font-semibold text-white">{item.label}</h3>
                    <ChevronRight className="h-4 w-4 text-dark-600 transition-colors group-hover:text-primary-300" />
                  </div>
                  <p className="mt-1 text-xs font-semibold uppercase tracking-[0.16em] text-dark-500">{config.label}</p>
                  <p className="mt-2 text-sm leading-5 text-dark-400">{item.description}</p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </section>
  );
}