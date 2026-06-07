export const storageKeys = {
  accessToken: 'agentstore_access_token',
  refreshToken: 'agentstore_refresh_token',
  activeTenant: 'agentstore_active_tenant',
  theme: 'agentstore_theme',
  impersonating: 'agentstore_impersonating',
  sessionId: 'agentstore_session_id',
} as const;

export function getStoredValue(key: string) {
  return localStorage.getItem(key) ?? sessionStorage.getItem(key);
}

export function setStoredValue(key: string, value: string, storage: Storage = localStorage) {
  storage.setItem(key, value);
}

export function removeStoredValue(key: string, storage: Storage = localStorage) {
  storage.removeItem(key);
}
