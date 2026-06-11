import { useState, useMemo, useEffect } from 'react';
import { Settings } from 'lucide-react';
import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useTenant } from '../../contexts/TenantContext';
import { useAuth } from '../../contexts/AuthContext';
import { useBranding } from '../../contexts/BrandingContext';
import ProfileTab from './settings/ProfileTab';
import SecurityTab from './settings/SecurityTab';
import SessionsTab from './settings/SessionsTab';
import BillingTab from './settings/BillingTab';
import AgentsTab from './settings/AgentsTab';
import ModelSettingsTab from './settings/ModelSettingsTab';

type SettingsTab = 'profile' | 'security' | 'sessions' | 'billing' | 'agents' | 'models';

export default function SettingsPage() {
  const { user } = useAuth();
  const { isRootTenant, role } = useTenant();
  const { branding } = useBranding();
  const { t } = useTranslation('app');
  const location = useLocation();
  const passkeysEnabled = branding?.authProviders?.passkeys ?? false;
  const mfaConfigEnabled = branding?.authProviders?.mfa ?? false;
  const showMfaSection = mfaConfigEnabled || user?.totpEnabled;
  const showSecurityTab = passkeysEnabled || showMfaSection;
  const canManageMarketplaceSupply = isRootTenant && (role === 'owner' || role === 'admin');

  // Resolve active tab from pathname
  const pathTab = location.pathname.endsWith('/agents') ? 'agents' : location.pathname.endsWith('/models') ? 'models' : undefined;

  const tabs = useMemo(() => [
    { key: 'profile' as const, label: t('settings.tabs.profile') },
    ...(showSecurityTab ? [{ key: 'security' as const, label: t('settings.tabs.security') }] : []),
    { key: 'sessions' as const, label: t('settings.tabs.sessions') },
    { key: 'billing' as const, label: t('settings.tabs.billing') },
    ...(canManageMarketplaceSupply ? [
      { key: 'agents' as const, label: t('settings.tabs.agents') },
      { key: 'models' as const, label: t('settings.tabs.models') },
    ] : []),
  ], [showSecurityTab, canManageMarketplaceSupply, t]);

  const [tab, setTab] = useState<SettingsTab>(() => {
    if (pathTab && tabs.some(t => t.key === pathTab)) return pathTab as SettingsTab;
    return 'profile';
  });

  // Sync URL tab after tenant/role-gated tabs become available.
  useEffect(() => {
    if (pathTab && tabs.some(t => t.key === pathTab) && tab !== pathTab) {
      setTab(pathTab as SettingsTab);
    }
  }, [pathTab, tabs, tab]);

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
          <Settings className="w-7 h-7 text-primary-400" />
          {t('settings.heading')}
        </h1>
        <p className="text-dark-400 mt-1">{t('settings.subtext')}</p>
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-1 mb-6 bg-dark-900/50 border border-dark-800 rounded-xl p-1 max-w-md" role="tablist">
        {tabs.map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            role="tab"
            aria-selected={tab === t.key}
            className={`flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
              tab === t.key ? 'bg-dark-700 text-white' : 'text-dark-400 hover:text-dark-300'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'profile' && <ProfileTab />}
      {tab === 'security' && <SecurityTab />}
      {tab === 'sessions' && <SessionsTab />}
      {tab === 'billing' && <BillingTab />}
      {tab === 'agents' && <AgentsTab />}
      {tab === 'models' && <ModelSettingsTab />}
    </div>
  );
}
