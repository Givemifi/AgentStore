export const storageKeys = {
  accessToken: 'agentstore_access_token',
  refreshToken: 'agentstore_refresh_token',
  activeTenant: 'agentstore_active_tenant',
  theme: 'agentstore_theme',
  impersonating: 'agentstore_impersonating',
  sessionId: 'agentstore_session_id',
} as const;

const legacyKeys: Record<string, string> = {
  [storageKeys.accessToken]: 'lastsaas_access_token',
  [storageKeys.refreshToken]: 'lastsaas_refresh_token',
  [storageKeys.activeTenant]: 'lastsaas_active_tenant',
  [storageKeys.theme]: 'lastsaas_theme',
  [storageKeys.impersonating]: 'lastsaas_impersonating',
  [storageKeys.sessionId]: 'lastsaas_session_id',
};

export function getStoredValue(key: string) {
  const current = localStorage.getItem(key) ?? sessionStorage.getItem(key);
  if (current !== null) return current;
  const legacy = legacyKeys[key];
  if (!legacy) return null;
  return localStorage.getItem(legacy) ?? sessionStorage.getItem(legacy);
}

export function setStoredValue(key: string, value: string, storage: Storage = localStorage) {
  storage.setItem(key, value);
  const legacy = legacyKeys[key];
  if (legacy) storage.removeItem(legacy);
}

export function removeStoredValue(key: string, storage: Storage = localStorage) {
  storage.removeItem(key);
  const legacy = legacyKeys[key];
  if (legacy) storage.removeItem(legacy);
}