import { beforeEach, describe, expect, it } from 'vitest';
import { getStoredValue, removeStoredValue, setStoredValue, storageKeys } from './storageKeys';

describe('storage keys', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  it('uses the AgentStore namespace for all known keys', () => {
    expect(Object.values(storageKeys)).toEqual([
      'agentstore_access_token',
      'agentstore_refresh_token',
      'agentstore_active_tenant',
      'agentstore_theme',
      'agentstore_impersonating',
      'agentstore_session_id',
    ]);
  });

  it('reads current values from localStorage first', () => {
    localStorage.setItem(storageKeys.accessToken, 'local-access');
    sessionStorage.setItem(storageKeys.accessToken, 'session-access');

    expect(getStoredValue(storageKeys.accessToken)).toBe('local-access');
  });

  it('falls back to current sessionStorage values', () => {
    sessionStorage.setItem(storageKeys.refreshToken, 'session-refresh');

    expect(getStoredValue(storageKeys.refreshToken)).toBe('session-refresh');
  });

  it('sets values under the requested current key', () => {
    setStoredValue(storageKeys.activeTenant, 'tenant-1');

    expect(localStorage.getItem(storageKeys.activeTenant)).toBe('tenant-1');
  });

  it('sets values in the selected storage area', () => {
    setStoredValue(storageKeys.sessionId, 'session-1', sessionStorage);

    expect(localStorage.getItem(storageKeys.sessionId)).toBeNull();
    expect(sessionStorage.getItem(storageKeys.sessionId)).toBe('session-1');
  });

  it('removes only the requested current key from the selected storage area', () => {
    localStorage.setItem(storageKeys.theme, 'dark');
    sessionStorage.setItem(storageKeys.theme, 'light');

    removeStoredValue(storageKeys.theme, sessionStorage);

    expect(localStorage.getItem(storageKeys.theme)).toBe('dark');
    expect(sessionStorage.getItem(storageKeys.theme)).toBeNull();
  });
});
