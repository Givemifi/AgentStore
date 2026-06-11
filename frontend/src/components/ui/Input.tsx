import { forwardRef, type InputHTMLAttributes } from 'react';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, className = '', id, ...props }, ref) => {
    const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);
    return (
      <div>
        {label && (
          <label htmlFor={inputId} className="block text-[13px] font-medium text-dark-300 mb-1.5">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          className={`w-full h-8 px-2.5 bg-dark-800 border border-white/8 rounded-md text-white text-[13px] placeholder-dark-500 transition-colors focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500/30 disabled:opacity-50 disabled:cursor-not-allowed ${error ? 'border-red-500/60' : ''} ${className}`}
          {...props}
        />
        {error && <p className="text-xs text-red-400 mt-1">{error}</p>}
      </div>
    );
  }
);

Input.displayName = 'Input';
export default Input;
