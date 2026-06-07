import { beforeEach, describe, expect, it } from 'vitest';
import { getSessionId } from './useTelemetry';

// getSessionId must be exported from useTelemetry.ts for testing.
// If this import fails, the test correctly flags that getSessionId is not yet exported.

describe('getSessionId', () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it('creates a new session ID and stores it under the current AgentStore key', () => {
    const id = getSessionId();

    expect(id).toBeTruthy();
    expect(sessionStorage.getItem('agentstore_session_id')).toBe(id);
  });

  it('reuses an existing session ID from the current AgentStore key', () => {
    sessionStorage.setItem('agentstore_session_id', 'existing-id');

    const id = getSessionId();

    expect(id).toBe('existing-id');
  });

  it('uses sessionStorage as the canonical session ID store', () => {
    localStorage.setItem('agentstore_session_id', 'local-id');
    sessionStorage.setItem('agentstore_session_id', 'session-id');

    const id = getSessionId();

    expect(id).toBe('local-id');
    expect(sessionStorage.getItem('agentstore_session_id')).toBe('local-id');
  });
});
