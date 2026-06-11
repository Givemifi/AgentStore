import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Rocket } from 'lucide-react';
import { bootstrapApi } from '../api/client';

export default function BootstrapPage() {
  const { t } = useTranslation('auth');
  const navigate = useNavigate();

  const [form, setForm] = useState({
    org: '',
    name: '',
    email: '',
    password: '',
    confirm: '',
  });
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm(prev => ({ ...prev, [e.target.name]: e.target.value }));
    setError('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (form.password !== form.confirm) {
      setError(t('setup.passwordMismatch'));
      return;
    }

    setSubmitting(true);
    try {
      await bootstrapApi.setup({
        org: form.org,
        name: form.name,
        email: form.email,
        password: form.password,
      });
      setDone(true);
      setTimeout(() => navigate('/login'), 1500);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        'Setup failed. Please try again.';
      if (msg.includes('already')) {
        setError(t('setup.alreadyDone'));
      } else {
        setError(msg);
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (done) {
    return (
      <div className="min-h-screen bg-dark-950 flex items-center justify-center px-4">
        <div className="text-center space-y-4">
          <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary-500 to-accent-purple flex items-center justify-center mx-auto">
            <Rocket className="w-8 h-8 text-white" />
          </div>
          <p className="text-white text-lg font-medium">{t('setup.successMessage')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-dark-950 flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary-500 to-accent-purple flex items-center justify-center mx-auto mb-4">
            <Rocket className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white">{t('setup.heading')}</h1>
          <p className="text-dark-400 mt-2 text-sm">{t('setup.subtext')}</p>
        </div>

        <div className="bg-dark-900/50 backdrop-blur-sm border border-dark-800 rounded-2xl p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Organization */}
            <div>
              <label className="block text-sm font-medium text-dark-300 mb-1">
                {t('setup.orgLabel')}
              </label>
              <input
                type="text"
                name="org"
                value={form.org}
                onChange={handleChange}
                placeholder={t('setup.orgPlaceholder')}
                required
                className="w-full px-3 py-2 rounded-lg bg-dark-800 border border-dark-700 text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 text-sm"
              />
            </div>

            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-dark-300 mb-1">
                {t('setup.nameLabel')}
              </label>
              <input
                type="text"
                name="name"
                value={form.name}
                onChange={handleChange}
                placeholder={t('setup.namePlaceholder')}
                required
                className="w-full px-3 py-2 rounded-lg bg-dark-800 border border-dark-700 text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 text-sm"
              />
            </div>

            {/* Email */}
            <div>
              <label className="block text-sm font-medium text-dark-300 mb-1">
                {t('setup.emailLabel')}
              </label>
              <input
                type="email"
                name="email"
                value={form.email}
                onChange={handleChange}
                placeholder={t('setup.emailPlaceholder')}
                required
                className="w-full px-3 py-2 rounded-lg bg-dark-800 border border-dark-700 text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 text-sm"
              />
            </div>

            {/* Password */}
            <div>
              <label className="block text-sm font-medium text-dark-300 mb-1">
                {t('setup.passwordLabel')}
              </label>
              <input
                type="password"
                name="password"
                value={form.password}
                onChange={handleChange}
                placeholder={t('setup.passwordPlaceholder')}
                required
                className="w-full px-3 py-2 rounded-lg bg-dark-800 border border-dark-700 text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 text-sm"
              />
            </div>

            {/* Confirm Password */}
            <div>
              <label className="block text-sm font-medium text-dark-300 mb-1">
                {t('setup.confirmLabel')}
              </label>
              <input
                type="password"
                name="confirm"
                value={form.confirm}
                onChange={handleChange}
                placeholder={t('setup.confirmPlaceholder')}
                required
                className="w-full px-3 py-2 rounded-lg bg-dark-800 border border-dark-700 text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 text-sm"
              />
            </div>

            {error && (
              <p className="text-red-400 text-sm">{error}</p>
            )}

            <button
              type="submit"
              disabled={submitting}
              className="w-full py-2.5 px-4 bg-gradient-to-r from-primary-600 to-primary-500 text-white font-medium rounded-lg hover:from-primary-500 hover:to-primary-400 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
            >
              {submitting ? t('setup.submitting') : t('setup.submit')}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
