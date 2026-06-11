import type { ReactNode } from 'react';

type AlertVariant = 'error' | 'success' | 'info';

interface AlertProps {
  variant?: AlertVariant;
  children: ReactNode;
  className?: string;
}

const variantClasses: Record<AlertVariant, string> = {
  error: 'border-red-500/30 text-red-400',
  success: 'border-accent-emerald/30 text-accent-emerald',
  info: 'border-primary-500/30 text-primary-400',
};

export default function Alert({ variant = 'error', children, className = '' }: AlertProps) {
  return (
    <div className={`px-3 py-2 border rounded-md text-[13px] ${variantClasses[variant]} ${className}`}>
      {children}
    </div>
  );
}
