import { Link } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';

type EmptyStateAction =
  | { label: string; to: string; onClick?: never }
  | { label: string; onClick: () => void; to?: never };

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: EmptyStateAction;
  className?: string;
}

export default function EmptyState({ icon: Icon, title, description, action, className = '' }: EmptyStateProps) {
  const actionClassName = 'mt-6 inline-flex items-center justify-center rounded-xl bg-primary-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-600';

  return (
    <div className={`rounded-lg border border-dashed border-white/10 bg-dark-950/70 px-6 py-14 text-center ${className}`}>
      <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-xl border border-white/8 bg-dark-900/80 text-primary-300">
        <Icon className="h-6 w-6" />
      </div>
      <h2 className="mt-5 text-xl font-semibold text-white">{title}</h2>
      <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-dark-400">{description}</p>
      {action && typeof action.to === 'string' ? (
        <Link to={action.to} className={actionClassName}>{action.label}</Link>
      ) : action ? (
        <button type="button" onClick={action.onClick} className={actionClassName}>{action.label}</button>
      ) : null}
    </div>
  );
}
