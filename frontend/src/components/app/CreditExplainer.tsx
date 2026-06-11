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
    <div className={`rounded-lg border border-white/8 bg-dark-900 p-4 ${className}`}>
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-dark-500">
        <Zap className="h-3.5 w-3.5 text-primary-400" />
        Credits
      </div>
      {balance === null ? (
        <p className="mt-2 text-[13px] text-dark-400">Credits will appear when usage data is available.</p>
      ) : (
        <p className="mt-2 text-xl font-semibold text-white">
          {balance.toLocaleString()}{' '}
          <span className="text-[13px] font-normal text-dark-400">{balanceCreditUnit} available</span>
        </p>
      )}
      <p className="mt-1.5 text-[13px] text-dark-400">Most selected Agents cost {perMessageCost} {creditUnit}/message.</p>
      <p className="mt-1.5 inline-flex items-center gap-1.5 text-xs text-accent-emerald">
        <ShieldCheck className="h-3 w-3" />
        Credits are charged only after a successful response.
      </p>
    </div>
  );
}
