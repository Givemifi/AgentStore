import type { ReactNode } from 'react';

type BadgeVariant = 'success' | 'danger' | 'warning' | 'info' | 'neutral';

interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
  className?: string;
}

const variantClasses: Record<BadgeVariant, string> = {
  success: 'border-accent-emerald/30 text-accent-emerald',
  danger: 'border-red-500/30 text-red-400',
  warning: 'border-amber-500/30 text-amber-400',
  info: 'border-primary-500/30 text-primary-400',
  neutral: 'border-white/8 text-dark-400',
};

export default function Badge({ variant = 'neutral', children, className = '' }: BadgeProps) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 border rounded-md text-xs font-medium ${variantClasses[variant]} ${className}`}>
      {children}
    </span>
  );
}
