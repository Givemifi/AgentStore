import { useState } from 'react';
import { Link } from 'react-router-dom';
import { KeyRound, ArrowLeft } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { authApi } from '../../api/client';

export default function ForgotPasswordPage() {
  const { t } = useTranslation('auth');
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await authApi.forgotPassword(email);
    } catch {
      // Always show success to prevent email enumeration
    }
    setSent(true);
    setLoading(false);
  };

  return (
    <div className="min-h-screen bg-dark-950 flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-primary-500 to-accent-purple flex items-center justify-center mx-auto mb-4">
            <KeyRound className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white">{t('forgotPassword.heading')}</h1>
          <p className="text-dark-400 mt-2">{t('forgotPassword.subtext')}</p>
        </div>

        <div className="bg-dark-900 border border-white/8 rounded-lg p-6">
          {sent ? (
            <div className="text-center py-4">
              <p className="text-dark-300 mb-4">
                {t('forgotPassword.sentMessage', { email }).split(email).map((part, i, arr) =>
                  i < arr.length - 1
                    ? <span key={i}>{part}<span className="text-white font-medium">{email}</span></span>
                    : <span key={i}>{part}</span>
                )}
              </p>
              <Link
                to="/login"
                className="inline-flex items-center gap-2 text-primary-400 hover:text-primary-300 transition-colors text-sm"
              >
                <ArrowLeft className="w-4 h-4" />
                {t('forgotPassword.backToLogin')}
              </Link>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-dark-300 mb-1.5">{t('email', { ns: 'common' })}</label>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full px-4 py-2.5 bg-dark-800 border border-white/8 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors"
                  placeholder={t('login.emailPlaceholder')}
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full py-2.5 px-4 bg-primary-500 hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
              >
                {loading ? t('forgotPassword.sending') : t('forgotPassword.sendLink')}
              </button>

              <div className="text-center">
                <Link
                  to="/login"
                  className="inline-flex items-center gap-2 text-dark-400 hover:text-white transition-colors text-sm"
                >
                  <ArrowLeft className="w-4 h-4" />
                  {t('forgotPassword.backToLogin')}
                </Link>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
