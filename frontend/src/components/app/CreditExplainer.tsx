import { ShieldCheck, Zap } from 'lucide-react';

interface CreditExplainerProps {
  balance: number | null;
  perMessageCost?: number;
  className?: string;
}

export default function CreditExplainer({ balance, perMessageCost = 1, className = '' }: CreditExplainerProps) {
  const creditUnit = perMessageCost === 1 ? 'credit' : 'credits';
  const balanceCreditUnit = balance === 1 ? 'credit' : 'credits';

  return (
    <div className={`rounded-2xl border border-white/8 bg-dark-900/70 p-5 backdrop-blur-sm ${className}`}>
      <div className="flex items-start gap-4">
        <div className="rounded-2xl border border-primary-400/15 bg-primary-500/10 p-3 text-primary-200">
          <Zap className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-dark-400">Credits</p>
          {balance === null ? (
            <p className="mt-3 text-sm text-dark-400">Credits will appear when usage data is available.</p>
          ) : (
            <p className="mt-3 text-2xl font-semibold text-white">{balance.toLocaleString()} {balanceCreditUnit} available</p>
          )}
          <p className="mt-2 text-sm text-dark-300">Most selected Agents cost {perMessageCost} {creditUnit}/message.</p>
          <p className="mt-2 inline-flex items-center gap-1.5 text-xs text-accent-emerald">
            <ShieldCheck className="h-3.5 w-3.5" />
            Credits are charged only after a successful response.
          </p>
        </div>
      </div>
    </div>
  );
}
