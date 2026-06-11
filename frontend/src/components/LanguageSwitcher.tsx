import { useTranslation } from 'react-i18next';
import { Globe } from 'lucide-react';

const SUPPORTED_LANGS = [
  { code: 'en', label: 'English' },
  { code: 'zh', label: '中文' },
];

interface LanguageSwitcherProps {
  /** compact: icon + current lang code only (for header). full: dropdown with labels. */
  variant?: 'compact' | 'full';
  className?: string;
}

export function LanguageSwitcher({ variant = 'compact', className = '' }: LanguageSwitcherProps) {
  const { i18n } = useTranslation();
  const currentLang = i18n.resolvedLanguage?.startsWith('zh') ? 'zh' : 'en';

  const toggle = () => {
    const next = currentLang === 'en' ? 'zh' : 'en';
    void i18n.changeLanguage(next);
    document.documentElement.lang = next;
  };

  if (variant === 'compact') {
    return (
      <button
        type="button"
        onClick={toggle}
        className={`flex items-center gap-1.5 text-dark-400 hover:text-white transition-colors text-sm ${className}`}
        aria-label={`Switch language to ${currentLang === 'en' ? '中文' : 'English'}`}
        title={`Switch to ${currentLang === 'en' ? '中文' : 'English'}`}
      >
        <Globe className="w-4 h-4" />
        <span className="hidden sm:inline font-medium uppercase">{currentLang}</span>
      </button>
    );
  }

  return (
    <div className={`flex items-center gap-1 ${className}`}>
      <Globe className="w-4 h-4 text-dark-400" />
      {SUPPORTED_LANGS.map(({ code, label }) => (
        <button
          key={code}
          type="button"
          onClick={() => {
            void i18n.changeLanguage(code);
            document.documentElement.lang = code;
          }}
          className={`px-2 py-1 rounded text-sm transition-colors ${
            currentLang === code
              ? 'text-white font-medium'
              : 'text-dark-400 hover:text-dark-200'
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  );
}
