import { useState, useEffect } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { MailCheck, MailX, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { authApi } from '../../api/client';

export default function VerifyEmailPage() {
  const { t } = useTranslation('auth');
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!token) {
      setStatus('error');
      setMessage(t('verifyEmail.missingToken'));
      return;
    }

    authApi.verifyEmail(token)
      .then(() => {
        setStatus('success');
        setMessage(t('verifyEmail.successMessage'));
      })
      .catch((err) => {
        setStatus('error');
        setMessage(err.response?.data?.error || t('verifyEmail.defaultError'));
      });
  }, [token, t]);

  return (
    <div className="min-h-screen bg-dark-950 flex items-center justify-center px-4">
      <div className="w-full max-w-md text-center">
        {status === 'loading' && (
          <>
            <Loader2 className="w-12 h-12 text-primary-500 animate-spin mx-auto mb-4" />
            <h1 className="text-xl font-bold text-white">{t('verifyEmail.verifying')}</h1>
          </>
        )}

        {status === 'success' && (
          <div className="bg-dark-900 border border-white/8 rounded-lg p-8">
            <div className="w-14 h-14 rounded-xl bg-accent-emerald/20 flex items-center justify-center mx-auto mb-4">
              <MailCheck className="w-7 h-7 text-accent-emerald" />
            </div>
            <h1 className="text-xl font-bold text-white mb-2">{t('verifyEmail.successHeading')}</h1>
            <p className="text-dark-400 mb-6">{message}</p>
            <Link
              to="/login"
              className="inline-block py-2.5 px-6 bg-primary-500 hover:bg-primary-600 transition-all"
            >
              {t('verifyEmail.continueToLogin')}
            </Link>
          </div>
        )}

        {status === 'error' && (
          <div className="bg-dark-900 border border-white/8 rounded-lg p-8">
            <div className="w-14 h-14 rounded-xl bg-red-500/20 flex items-center justify-center mx-auto mb-4">
              <MailX className="w-7 h-7 text-red-400" />
            </div>
            <h1 className="text-xl font-bold text-white mb-2">{t('verifyEmail.failedHeading')}</h1>
            <p className="text-dark-400 mb-6">{message}</p>
            <Link
              to="/login"
              className="inline-block py-2.5 px-6 bg-dark-800 border border-white/8 text-white font-medium rounded-lg hover:bg-dark-700 transition-all"
            >
              {t('verifyEmail.backToLogin')}
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
