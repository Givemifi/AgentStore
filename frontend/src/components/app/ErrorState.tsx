import { AlertCircle } from 'lucide-react';

interface ErrorStateProps {
  title: string;
  message: string;
  retryLabel?: string;
  onRetry?: () => void;
  isRetrying?: boolean;
  className?: string;
}

export default function ErrorState({
  title,
  message,
  retryLabel = 'Retry',
  onRetry,
  isRetrying = false,
  className = '',
}: ErrorStateProps) {
  return (
    <div role="alert" className={`flex flex-col gap-4 rounded-xl border border-red-500/25 bg-red-500/10 p-4 text-sm text-red-200 sm:flex-row sm:items-center sm:justify-between ${className}`}>
      <div className="flex gap-3">
        <AlertCircle className="mt-0.5 h-5 w-5 flex-shrink-0 text-red-300" />
        <div>
          <p className="font-semibold text-red-100">{title}</p>
          <p className="mt-1 text-red-200/80">{message}</p>
        </div>
      </div>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          disabled={isRetrying}
          className="inline-flex items-center justify-center rounded-xl border border-red-400/30 bg-red-500/10 px-4 py-2 font-semibold text-red-100 transition-colors hover:bg-red-500/20 disabled:cursor-not-allowed disabled:opacity-70"
        >
          {isRetrying ? 'Retrying...' : retryLabel}
        </button>
      ) : null}
    </div>
  );
}
