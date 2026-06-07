import { useCallback, useRef } from 'react';
import { telemetryApi } from '../api/client';
import { getStoredValue, setStoredValue, storageKeys } from '../utils/storageKeys';

export function getSessionId(): string {
  const key = storageKeys.sessionId;
  // getStoredValue checks the current AgentStore key in localStorage/sessionStorage.
  // setStoredValue writes the current session key to sessionStorage.
  let id = getStoredValue(key);
  if (!id) {
    id = crypto.randomUUID();
  }
  setStoredValue(key, id, sessionStorage);
  return id;
}

export function useTelemetry() {
  const lastPageView = useRef<string>('');
  const lastPageViewTime = useRef<number>(0);

  const trackPageView = useCallback((page: string) => {
    const now = Date.now();
    // Debounce: no duplicate page view within 5 seconds
    if (page === lastPageView.current && now - lastPageViewTime.current < 5000) {
      return;
    }
    lastPageView.current = page;
    lastPageViewTime.current = now;

    const sessionId = getSessionId();
    telemetryApi.trackAnonymous({ sessionId, event: 'page.view', properties: { page } }).catch(() => {});
  }, []);

  const trackEvent = useCallback((event: string, properties?: Record<string, unknown>) => {
    telemetryApi.trackEvent({ event, properties }).catch(() => {});
  }, []);

  return { trackPageView, trackEvent };
}
