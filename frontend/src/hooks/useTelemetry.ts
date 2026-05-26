import { useCallback, useRef } from 'react';
import { telemetryApi } from '../api/client';

function getSessionId(): string {
  const SESSION_KEY = 'agentstore_session_id';
  let id = sessionStorage.getItem(SESSION_KEY);
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem(SESSION_KEY, id);
    // Also check legacy key
    const legacyId = sessionStorage.getItem('lastsaas_session_id');
    if (legacyId && legacyId !== id) {
      sessionStorage.removeItem('lastsaas_session_id');
    }
  }
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
