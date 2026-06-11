import { forwardRef, type TextareaHTMLAttributes } from 'react';

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
}

const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ label, error, className = '', id, ...props }, ref) => {
    const textareaId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);
    return (
      <div>
        {label && (
          <label htmlFor={textareaId} className="block text-[13px] font-medium text-dark-300 mb-1.5">
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          id={textareaId}
          className={`w-full px-2.5 py-2 bg-dark-800 border border-white/8 rounded-md text-white text-[13px] placeholder-dark-500 transition-colors focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500/30 resize-none disabled:opacity-50 disabled:cursor-not-allowed ${error ? 'border-red-500/60' : ''} ${className}`}
          {...props}
        />
        {error && <p className="text-xs text-red-400 mt-1">{error}</p>}
      </div>
    );
  }
);

Textarea.displayName = 'Textarea';
export default Textarea;
