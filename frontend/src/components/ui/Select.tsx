import { forwardRef, type SelectHTMLAttributes } from 'react';

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
}

const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, className = '', children, id, ...props }, ref) => {
    const selectId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);
    return (
      <div>
        {label && (
          <label htmlFor={selectId} className="block text-[13px] font-medium text-dark-300 mb-1.5">
            {label}
          </label>
        )}
        <select
          ref={ref}
          id={selectId}
          className={`w-full h-8 px-2.5 bg-dark-800 border border-white/8 rounded-md text-white text-[13px] transition-colors focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500/30 disabled:opacity-50 disabled:cursor-not-allowed ${error ? 'border-red-500/60' : ''} ${className}`}
          {...props}
        >
          {children}
        </select>
        {error && <p className="text-xs text-red-400 mt-1">{error}</p>}
      </div>
    );
  }
);

Select.displayName = 'Select';
export default Select;
