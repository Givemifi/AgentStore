import axios from 'axios';
import type { AuthResponse, MFARequiredResponse, AuthProviders, ActiveSession, ActivityLogEntry, PasskeyCredential, ImpersonationResponse, TenantMember, TenantDetail, TenantListItem, UserListItem, Message, AboutInfo, SystemLog, ConfigVar, UserDetail, UserMembershipDetail, DeletePreflightResponse, Plan, EntitlementKeyInfo, PublicPlansResponse, CreditBundle, SystemNode, SystemMetric, FinancialTransaction, DailyMetricPoint, IntegrationCheck, APIKey, Webhook, WebhookDelivery, WebhookEventTypeInfo, BrandingConfig, MediaItem, CustomPage, Promotion, EligibleProduct, Announcement, UsageSummary, Invitation, FunnelData, CohortRow, EngagementData, KPIData, CustomEventData, EventTypeSummary, EventDefinition, SankeyData, Agent, ChatMessage, Conversation, ChatRequest, ChatResponse, ModelProvider, ModelConfig, LaunchReadinessResponse } from '../types';
import { getStoredValue, removeStoredValue, setStoredValue, storageKeys } from '../utils/storageKeys';

const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
});

// Public (unauthenticated) API client — for catalog/market endpoints
const publicApi = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
});

// Auth header management
export function setAuthToken(token: string | null) {
  if (token) {
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  } else {
    delete api.defaults.headers.common['Authorization'];
  }
}

export function setTenantHeader(tenantId: string | null) {
  if (tenantId) {
    api.defaults.headers.common['X-Tenant-ID'] = tenantId;
  } else {
    delete api.defaults.headers.common['X-Tenant-ID'];
  }
}

// Silent token refresh: on 401, attempt to use the refresh token to get a new
// access token and retry the original request. Only falls back to redirect if
// the refresh itself fails.
let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

function subscribeToRefresh(cb: (token: string) => void) {
  refreshSubscribers.push(cb);
}

function onRefreshComplete(newToken: string) {
  refreshSubscribers.forEach(cb => cb(newToken));
  refreshSubscribers = [];
}

function onRefreshFailed() {
  refreshSubscribers = [];
}

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const originalRequest = error.config;
    const isAuthRoute = originalRequest?.url?.includes('/auth/login') || originalRequest?.url?.includes('/auth/refresh');

    if (error.response?.status !== 401 || isAuthRoute || originalRequest?._retry) {
      return Promise.reject(error);
    }

    const refreshToken = getStoredValue(storageKeys.refreshToken);
    if (!refreshToken) {
      removeStoredValue(storageKeys.accessToken);
      delete api.defaults.headers.common['Authorization'];
      window.location.href = '/login';
      return Promise.reject(error);
    }

    if (isRefreshing) {
      // Another request is already refreshing — queue this one
      return new Promise((resolve) => {
        subscribeToRefresh((newToken: string) => {
          originalRequest.headers['Authorization'] = `Bearer ${newToken}`;
          originalRequest._retry = true;
          resolve(api(originalRequest));
        });
      });
    }

    isRefreshing = true;
    originalRequest._retry = true;

    try {
      const { data } = await api.post<AuthResponse>('/auth/refresh', { refreshToken });
      setStoredValue(storageKeys.accessToken, data.accessToken);
      setStoredValue(storageKeys.refreshToken, data.refreshToken);
      setAuthToken(data.accessToken);
      onRefreshComplete(data.accessToken);
      originalRequest.headers['Authorization'] = `Bearer ${data.accessToken}`;
      return api(originalRequest);
    } catch {
      onRefreshFailed();
      removeStoredValue(storageKeys.accessToken);
      removeStoredValue(storageKeys.refreshToken);
      delete api.defaults.headers.common['Authorization'];
      window.location.href = '/login';
      return Promise.reject(error);
    } finally {
      isRefreshing = false;
    }
  }
);

// 503 interceptor (system not initialized)
api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 503 && error.response?.data?.redirect === '/setup') {
      window.location.href = '/setup';
    }
    return Promise.reject(error);
  }
);

// --- Bootstrap ---
export const bootstrapApi = {
  status: () => api.get<{ initialized: boolean }>('/bootstrap/status').then(r => r.data),
  setup: (data: { org: string; name: string; email: string; password: string }) =>
    api.post<{ initialized: boolean }>('/bootstrap/setup', data).then(r => r.data),
};

// --- Auth ---
export const authApi = {
  register: (data: { email: string; password: string; displayName: string; invitationToken?: string; refCode?: string }) =>
    api.post<AuthResponse>('/auth/register', data).then(r => r.data),
  login: (data: { email: string; password: string }) =>
    api.post<AuthResponse | MFARequiredResponse>('/auth/login', data).then(r => r.data),
  logout: (refreshToken?: string) =>
    api.post('/auth/logout', { refreshToken }),
  refresh: (refreshToken: string) =>
    api.post<AuthResponse>('/auth/refresh', { refreshToken }).then(r => r.data),
  getMe: () =>
    api.get<{ user: import('../types').User; memberships: import('../types').MembershipInfo[] }>('/auth/me').then(r => r.data),
  verifyEmail: (token: string) =>
    api.post('/auth/verify-email', { token }).then(r => r.data),
  resendVerification: (email: string) =>
    api.post('/auth/resend-verification', { email }).then(r => r.data),
  forgotPassword: (email: string) =>
    api.post('/auth/forgot-password', { email }).then(r => r.data),
  resetPassword: (token: string, newPassword: string) =>
    api.post('/auth/reset-password', { token, newPassword }).then(r => r.data),
  changePassword: (currentPassword: string, newPassword: string) =>
    api.post('/auth/change-password', { currentPassword, newPassword }).then(r => r.data),
  acceptInvitation: (token: string) =>
    api.post('/auth/accept-invitation', { token }).then(r => r.data),

  // OAuth code exchange
  exchangeCode: (code: string) =>
    api.post<{ accessToken?: string; refreshToken?: string; mfaRequired?: boolean; mfaToken?: string }>('/auth/exchange-code', { code }).then(r => r.data),

  // Auth providers discovery
  getProviders: () =>
    api.get<AuthProviders>('/auth/providers').then(r => r.data),

  // MFA / TOTP
  mfaSetup: () =>
    api.post<{ secret: string; qrCodeUrl: string }>('/auth/mfa/setup').then(r => r.data),
  mfaVerifySetup: (code: string) =>
    api.post<{ recoveryCodes: string[] }>('/auth/mfa/verify-setup', { code }).then(r => r.data),
  mfaDisable: (code: string) =>
    api.post('/auth/mfa/disable', { code }).then(r => r.data),
  mfaChallenge: (mfaToken: string, code: string) =>
    api.post<AuthResponse>('/auth/mfa/challenge', { mfaToken, code }).then(r => r.data),
  mfaRegenerateCodes: (code: string) =>
    api.post<{ recoveryCodes: string[] }>('/auth/mfa/regenerate-codes', { code }).then(r => r.data),

  // Magic Link
  requestMagicLink: (email: string) =>
    api.post('/auth/magic-link', { email }).then(r => r.data),
  verifyMagicLink: (token: string) =>
    api.post<AuthResponse | MFARequiredResponse>('/auth/magic-link/verify', { token }).then(r => r.data),

  // Passkeys / WebAuthn
  passkeyRegisterBegin: () =>
    api.post('/auth/passkeys/register/begin').then(r => r.data),
  passkeyRegisterFinish: (data: { name: string; credential: unknown }) =>
    api.post('/auth/passkeys/register/finish', data).then(r => r.data),
  passkeyLoginBegin: () =>
    api.post('/auth/passkeys/login/begin').then(r => r.data),
  passkeyLoginFinish: (credential: unknown) =>
    api.post<AuthResponse>('/auth/passkeys/login/finish', { credential }).then(r => r.data),
  listPasskeys: () =>
    api.get<{ passkeys: PasskeyCredential[] }>('/auth/passkeys').then(r => r.data),
  deletePasskey: (id: string) =>
    api.delete(`/auth/passkeys/${id}`).then(r => r.data),

  // Sessions
  listSessions: () =>
    api.get<{ sessions: ActiveSession[] }>('/auth/sessions').then(r => r.data),
  revokeSession: (id: string) =>
    api.delete(`/auth/sessions/${id}`).then(r => r.data),
  revokeAllSessions: () =>
    api.delete('/auth/sessions').then(r => r.data),

  // Preferences
  updatePreferences: (data: { themePreference?: string }) =>
    api.patch('/auth/preferences', data).then(r => r.data),

  // Onboarding
  completeOnboarding: () =>
    api.post('/auth/complete-onboarding').then(r => r.data),

  // Account management
  deleteAccount: (password: string) =>
    api.post('/auth/delete-account', { password }).then(r => r.data),
  exportData: () =>
    api.get('/auth/export-data', { responseType: 'blob' }).then(r => r.data),
};

// --- Tenant ---
export const tenantApi = {
  listMembers: () =>
    api.get<{ members: TenantMember[] }>('/tenant/members').then(r => r.data),
  inviteMember: (email: string, role: string) =>
    api.post('/tenant/members/invite', { email, role }).then(r => r.data),
  removeMember: (userId: string) =>
    api.delete(`/tenant/members/${userId}`).then(r => r.data),
  changeRole: (userId: string, role: string) =>
    api.patch(`/tenant/members/${userId}/role`, { role }).then(r => r.data),
  transferOwnership: (userId: string) =>
    api.post(`/tenant/members/${userId}/transfer-ownership`).then(r => r.data),
  getActivity: (params?: { page?: number; perPage?: number; action?: string; search?: string }) =>
    api.get<{ logs: ActivityLogEntry[]; total: number }>('/tenant/activity', { params }).then(r => r.data),
  updateSettings: (data: { name?: string }) =>
    api.patch('/tenant/settings', data).then(r => r.data),
};

// --- Messages ---
export const messagesApi = {
  list: () =>
    api.get<{ messages: Message[] }>('/messages').then(r => r.data),
  unreadCount: () =>
    api.get<{ count: number }>('/messages/unread-count').then(r => r.data),
  markRead: (id: string) =>
    api.patch(`/messages/${id}/read`).then(r => r.data),
};

// --- Admin ---
export const adminApi = {
  getAbout: () =>
    api.get<AboutInfo>('/admin/about').then(r => r.data),
  getDashboard: () =>
    api.get<{ users: number; tenants: number; health: { healthy: boolean; issues: string[] } }>('/admin/dashboard').then(r => r.data),
  listTenants: (params?: { page?: number; limit?: number; search?: string; sort?: string; status?: string; billingStatus?: string }) =>
    api.get<{ tenants: TenantListItem[]; total: number; page: number; limit: number }>('/admin/tenants', { params }).then(r => r.data),
  getTenant: (id: string) =>
    api.get<{ tenant: TenantDetail; members: TenantMember[] }>(`/admin/tenants/${id}`).then(r => r.data),
  updateTenant: (id: string, data: { name?: string; billingWaived?: boolean; subscriptionCredits?: number; purchasedCredits?: number }) =>
    api.put(`/admin/tenants/${id}`, data).then(r => r.data),
  updateTenantStatus: (id: string, isActive: boolean) =>
    api.patch(`/admin/tenants/${id}/status`, { isActive }).then(r => r.data),
  listUsers: (params?: { page?: number; limit?: number; search?: string; sort?: string; status?: string }) =>
    api.get<{ users: UserListItem[]; total: number; page: number; limit: number }>('/admin/users', { params }).then(r => r.data),
  updateUserStatus: (id: string, isActive: boolean) =>
    api.patch(`/admin/users/${id}/status`, { isActive }).then(r => r.data),
  listLogs: (params?: { page?: number; perPage?: number; severity?: string; category?: string; search?: string; userId?: string; fromDate?: string; toDate?: string }) =>
    api.get<{ logs: SystemLog[]; total: number }>('/admin/logs', { params }).then(r => r.data),
  logSeverityCounts: (params?: { category?: string; fromDate?: string; toDate?: string }) =>
    api.get<{ counts: Record<string, number> }>('/admin/logs/severity-counts', { params }).then(r => r.data),
  exportLogsCSV: (params?: { severity?: string; category?: string; search?: string; fromDate?: string; toDate?: string }) =>
    api.get('/admin/logs/export', { params, responseType: 'blob' }).then(r => r.data),
  exportUsersCSV: (params?: { search?: string; status?: string }) =>
    api.get('/admin/users/export', { params, responseType: 'blob' }).then(r => r.data),
  exportTenantsCSV: (params?: { search?: string; status?: string; billingStatus?: string }) =>
    api.get('/admin/tenants/export', { params, responseType: 'blob' }).then(r => r.data),

  // LLM Configuration
  getLLMConfig: () =>
    api.get<{ id?: string; apiKey: string; baseURL: string; model: string; isActive: boolean }>('/admin/llm-config').then(r => r.data),
  updateLLMConfig: (data: { apiKey: string; baseURL: string; model: string; isActive: boolean }) =>
    api.put('/admin/llm-config', data).then(r => r.data),

  // Payment provider configuration
  getPaymentConfig: (provider: 'wechat' | 'alipay') =>
    api.get<Record<string, unknown>>(`/admin/payment-config/${provider}`).then(r => r.data),
  updatePaymentConfig: (provider: 'wechat' | 'alipay', data: Record<string, unknown>) =>
    api.put<{ status: string; warning?: string }>(`/admin/payment-config/${provider}`, data).then(r => r.data),

  listConfig: () =>
    api.get<{ configs: ConfigVar[] }>('/admin/config').then(r => r.data),
  getConfig: (name: string) =>
    api.get<ConfigVar>(`/admin/config/${name}`).then(r => r.data),
  updateConfig: (name: string, value: string, opts?: { description?: string; options?: string }) =>
    api.put<ConfigVar>(`/admin/config/${name}`, { value, ...opts }).then(r => r.data),
  createConfig: (data: { name: string; description: string; type: string; value: string; options?: string }) =>
    api.post<ConfigVar>('/admin/config', data).then(r => r.data),
  deleteConfig: (name: string) =>
    api.delete(`/admin/config/${name}`).then(r => r.data),
  getUser: (id: string) =>
    api.get<{ user: UserDetail; memberships: UserMembershipDetail[] }>(`/admin/users/${id}`).then(r => r.data),
  updateUser: (id: string, data: { email?: string; displayName?: string }) =>
    api.put(`/admin/users/${id}`, data).then(r => r.data),
  updateUserRole: (userId: string, tenantId: string, role: string) =>
    api.patch(`/admin/users/${userId}/role/${tenantId}`, { role }).then(r => r.data),
  preflightDeleteUser: (id: string) =>
    api.get<DeletePreflightResponse>(`/admin/users/${id}/preflight-delete`).then(r => r.data),
  deleteUser: (id: string, data?: { replacementOwners?: Record<string, string>; confirmTenantDeletions?: string[] }) =>
    api.delete(`/admin/users/${id}`, { data }).then(r => r.data),
  listPlans: () =>
    api.get<{ plans: Plan[] }>('/admin/plans').then(r => r.data),
  getPlan: (id: string) =>
    api.get<Plan>(`/admin/plans/${id}`).then(r => r.data),
  createPlan: (data: Omit<Plan, 'id' | 'isSystem' | 'isArchived' | 'createdAt' | 'updatedAt' | 'subscriberCount'>) =>
    api.post<Plan>('/admin/plans', data).then(r => r.data),
  updatePlan: (id: string, data: Omit<Plan, 'id' | 'isSystem' | 'isArchived' | 'createdAt' | 'updatedAt' | 'subscriberCount'>) =>
    api.put<Plan>(`/admin/plans/${id}`, data).then(r => r.data),
  deletePlan: (id: string) =>
    api.delete(`/admin/plans/${id}`).then(r => r.data),
  archivePlan: (id: string) =>
    api.post(`/admin/plans/${id}/archive`).then(r => r.data),
  unarchivePlan: (id: string) =>
    api.post(`/admin/plans/${id}/unarchive`).then(r => r.data),
  listEntitlementKeys: () =>
    api.get<{ keys: EntitlementKeyInfo[] }>('/admin/entitlement-keys').then(r => r.data),
  assignTenantPlan: (tenantId: string, planId?: string | null, billingWaived?: boolean) => {
    const body: Record<string, unknown> = {};
    if (planId !== undefined) body.planId = planId || '';
    if (billingWaived !== undefined) body.billingWaived = billingWaived;
    return api.patch(`/admin/tenants/${tenantId}/plan`, body).then(r => r.data);
  },
  listBundles: () =>
    api.get<{ bundles: CreditBundle[] }>('/admin/credit-bundles').then(r => r.data),
  createBundle: (data: Omit<CreditBundle, 'id' | 'createdAt' | 'updatedAt'>) =>
    api.post<CreditBundle>('/admin/credit-bundles', data).then(r => r.data),
  updateBundle: (id: string, data: Omit<CreditBundle, 'id' | 'createdAt' | 'updatedAt'>) =>
    api.put<CreditBundle>(`/admin/credit-bundles/${id}`, data).then(r => r.data),
  deleteBundle: (id: string) =>
    api.delete(`/admin/credit-bundles/${id}`).then(r => r.data),
  listHealthNodes: () =>
    api.get<{ nodes: SystemNode[] }>('/admin/health/nodes').then(r => r.data),
  getHealthMetrics: (params?: { node?: string; range?: string }) =>
    api.get<{ metrics: SystemMetric[]; from: string; to: string }>('/admin/health/metrics', { params }).then(r => r.data),
  getHealthCurrent: () =>
    api.get<{ metrics: SystemMetric[] }>('/admin/health/current').then(r => r.data),
  getHealthIntegrations: () =>
    api.get<{ integrations: IntegrationCheck[] }>('/admin/health/integrations').then(r => r.data),
  sendTestEmail: (to: string) =>
    api.post<{ success?: boolean; error?: string }>('/admin/health/test-email', { to }).then(r => r.data),
  listFinancialTransactions: (params?: { page?: number; perPage?: number; tenantId?: string; search?: string }) =>
    api.get<{ transactions: FinancialTransaction[]; total: number; page: number; perPage: number }>('/admin/financial/transactions', { params }).then(r => r.data),
  getFinancialMetrics: (params?: { range?: string; metric?: string }) =>
    api.get<{ data: DailyMetricPoint[] }>('/admin/financial/metrics', { params }).then(r => r.data),
  adminCancelSubscription: (tenantId: string, immediate: boolean) =>
    api.post(`/admin/tenants/${tenantId}/cancel-subscription`, { immediate }).then(r => r.data),
  adminUpdateSubscription: (tenantId: string, data: { currentPeriodEnd?: string }) =>
    api.patch(`/admin/tenants/${tenantId}/subscription`, data).then(r => r.data),

  // Promotions
  listPromotions: () =>
    api.get<{ promotions: Promotion[]; productNames: Record<string, string> }>('/admin/promotions').then(r => r.data),
  listEligibleProducts: () =>
    api.get<{ items: EligibleProduct[] }>('/admin/promotions/eligible-products').then(r => r.data),
  createPromotion: (data: { code: string; name?: string; percentOff?: number; amountOff?: number; currency?: string; maxRedemptions?: number; expiresAt?: string; appliesTo?: { type: string; id: string }[] }) =>
    api.post<{ id: string; code: string }>('/admin/promotions', data).then(r => r.data),
  updatePromotion: (data: { id: string; couponId: string; couponName?: string; active?: boolean }) =>
    api.post('/admin/promotions/update', data).then(r => r.data),
  deactivatePromotion: (id: string) =>
    api.post('/admin/promotions/deactivate', { id }).then(r => r.data),

  // Launch Readiness
  getLaunchReadiness: () =>
    api.get<LaunchReadinessResponse>('/admin/launch-readiness').then(r => r.data),

  // Announcements
  listAnnouncements: () =>
    api.get<{ announcements: Announcement[] }>('/admin/announcements').then(r => r.data),
  createAnnouncement: (data: { title: string; body: string; publish: boolean }) =>
    api.post<Announcement>('/admin/announcements', data).then(r => r.data),
  updateAnnouncement: (id: string, data: { title?: string; body?: string; publish?: boolean }) =>
    api.put(`/admin/announcements/${id}`, data).then(r => r.data),
  deleteAnnouncement: (id: string) =>
    api.delete(`/admin/announcements/${id}`).then(r => r.data),

  // API Keys
  listAPIKeys: () =>
    api.get<{ apiKeys: APIKey[] }>('/admin/api-keys').then(r => r.data),
  createAPIKey: (data: { name: string; authority: string }) =>
    api.post<{ apiKey: APIKey; rawKey: string }>('/admin/api-keys', data).then(r => r.data),
  deleteAPIKey: (id: string) =>
    api.delete(`/admin/api-keys/${id}`).then(r => r.data),

  // Webhooks
  listWebhooks: () =>
    api.get<{ webhooks: Webhook[] }>('/admin/webhooks').then(r => r.data),
  getWebhook: (id: string) =>
    api.get<{ webhook: Webhook; deliveries: WebhookDelivery[] }>(`/admin/webhooks/${id}`).then(r => r.data),
  createWebhook: (data: { name: string; description: string; url: string; events: string[] }) =>
    api.post<{ webhook: Webhook; secret: string }>('/admin/webhooks', data).then(r => r.data),
  updateWebhook: (id: string, data: { name: string; description: string; url: string; events: string[] }) =>
    api.put<{ webhook: Webhook }>(`/admin/webhooks/${id}`, data).then(r => r.data),
  deleteWebhook: (id: string) =>
    api.delete(`/admin/webhooks/${id}`).then(r => r.data),
  testWebhook: (id: string) =>
    api.post<{ delivery: WebhookDelivery }>(`/admin/webhooks/${id}/test`).then(r => r.data),
  regenerateWebhookSecret: (id: string) =>
    api.post<{ secret: string; secretPreview: string }>(`/admin/webhooks/${id}/regenerate-secret`).then(r => r.data),
  listWebhookEventTypes: () =>
    api.get<{ eventTypes: WebhookEventTypeInfo[] }>('/admin/webhooks/event-types').then(r => r.data),

  // Impersonation
  impersonateUser: (userId: string) =>
    api.post<ImpersonationResponse>(`/admin/users/${userId}/impersonate`).then(r => r.data),

  // Root Members
  listRootMembers: () =>
    api.get<{ members: TenantMember[]; invitations: Invitation[] }>('/admin/members').then(r => r.data),
  inviteRootMember: (email: string, role: string) =>
    api.post('/admin/members/invite', { email, role }).then(r => r.data),
  removeRootMember: (userId: string) =>
    api.delete(`/admin/members/${userId}`).then(r => r.data),
  changeRootMemberRole: (userId: string, role: string) =>
    api.patch(`/admin/members/${userId}/role`, { role }).then(r => r.data),
  cancelRootInvitation: (invitationId: string) =>
    api.delete(`/admin/members/invitations/${invitationId}`).then(r => r.data),
};

// --- Plans (public, authenticated) ---
export const plansApi = {
  list: () =>
    api.get<PublicPlansResponse>('/plans').then(r => r.data),
};

// --- Credit Bundles (public, authenticated) ---
export const bundlesApi = {
  list: () =>
    api.get<{ bundles: CreditBundle[] }>('/credit-bundles').then(r => r.data),
};

// --- Announcements (public, authenticated) ---
export const announcementsApi = {
  list: () =>
    api.get<{ announcements: Announcement[] }>('/announcements').then(r => r.data),
};

// --- Usage Metering ---
export const usageApi = {
  record: (data: { type: string; quantity: number; metadata?: Record<string, unknown> }) =>
    api.post<{ id: string; type: string; quantity: number }>('/usage/record', data).then(r => r.data),
  summary: () =>
    api.get<UsageSummary>('/usage/summary').then(r => r.data),
};

// --- AI Agents / Expert Chat ---
export const agentsApi = {
  list: () =>
    api.get<Agent[]>('/chat/agents').then(r => r.data),
  get: (agentId: string) =>
    api.get<Agent>(`/chat/agents/${agentId}`).then(r => r.data),
};

// Public catalog API — no authentication required
export const publicAgentsApi = {
  list: () =>
    publicApi.get<{ agents: Agent[] }>('/public/agents').then(r => r.data.agents),
  get: (slug: string) =>
    publicApi.get<Agent>(`/public/agents/${slug}`).then(r => r.data),
};

export const tenantAgentsApi = {
  list: () => api.get<Agent[]>('/tenant/agents').then(r => r.data),
  get: (id: string) => api.get<Agent>(`/tenant/agents/${id}`).then(r => r.data),
  create: (data: Partial<Agent>) => api.post<Agent>('/tenant/agents', data).then(r => r.data),
  update: (id: string, data: Partial<Agent>) => api.put<Agent>(`/tenant/agents/${id}`, data).then(r => r.data),
  publish: (id: string) => api.post<{ status: string }>(`/tenant/agents/${id}/publish`).then(r => r.data),
  archive: (id: string) => api.post<{ status: string }>(`/tenant/agents/${id}/archive`).then(r => r.data),
  delete: (id: string) => api.delete(`/tenant/agents/${id}`).then(r => r.data),
};

export const tenantModelsApi = {
  listProviders: () => api.get<ModelProvider[]>('/tenant/model-providers').then(r => r.data),
  getProvider: (id: string) => api.get<ModelProvider>(`/tenant/model-providers/${id}`).then(r => r.data),
  createProvider: (data: Partial<ModelProvider>) => api.post<ModelProvider>('/tenant/model-providers', data).then(r => r.data),
  updateProvider: (id: string, data: Partial<ModelProvider>) => api.put<ModelProvider>(`/tenant/model-providers/${id}`, data).then(r => r.data),
  deleteProvider: (id: string) => api.delete(`/tenant/model-providers/${id}`).then(r => r.data),
  testProvider: (id: string) => api.post<{ status: string; message?: string }>(`/tenant/model-providers/${id}/test`).then(r => r.data),
  listModels: () => api.get<ModelConfig[]>('/tenant/model-configs').then(r => r.data),
  getModel: (id: string) => api.get<ModelConfig>(`/tenant/model-configs/${id}`).then(r => r.data),
  createModel: (data: Partial<ModelConfig>) => api.post<ModelConfig>('/tenant/model-configs', data).then(r => r.data),
  updateModel: (id: string, data: Partial<ModelConfig>) => api.put<ModelConfig>(`/tenant/model-configs/${id}`, data).then(r => r.data),
  deleteModel: (id: string) => api.delete(`/tenant/model-configs/${id}`).then(r => r.data),
  updateDefaults: (data: { defaultTextModelConfigId?: string; defaultImageModelConfigId?: string; defaultVideoModelConfigId?: string }) => api.post('/tenant/model-defaults', data).then(r => r.data),
};

// Streaming chat types
export type ChatStreamEvent =
  | { event: 'message_start'; data: { conversationId: string; messageId: string } }
  | { event: 'delta'; data: { text: string } }
  | { event: 'message_done'; data: { conversationId: string; messageId: string; creditsCharged: number; remainingCredits: number; model: string } }
  | { event: 'error'; data: { message: string } };

export type StreamChatError = Error & {
  response?: {
    status?: number;
    data?: unknown;
  };
};

export async function streamChat(data: ChatRequest, onEvent: (event: ChatStreamEvent) => void, signal?: AbortSignal) {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(api.defaults.headers.common.Authorization ? { Authorization: String(api.defaults.headers.common.Authorization) } : {}),
      ...(api.defaults.headers.common['X-Tenant-ID'] ? { 'X-Tenant-ID': String(api.defaults.headers.common['X-Tenant-ID']) } : {}),
    },
    body: JSON.stringify(data),
    signal,
  });
  if (!response.ok || !response.body) {
    const text = await response.text().catch(() => '');
    let data: unknown = text;

    if (text) {
      try {
        data = JSON.parse(text);
      } catch {
        data = text;
      }
    }

    const error = new Error('Unable to start chat stream') as StreamChatError;
    error.response = {
      status: response.status,
      data,
    };
    throw error;
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split('\n\n');
    buffer = events.pop() ?? '';
    for (const raw of events) {
      const lines = raw.split('\n');
      const event = lines.find((line) => line.startsWith('event: '))?.slice(7) as ChatStreamEvent['event'] | undefined;
      const dataLine = lines.find((line) => line.startsWith('data: '));
      if (!event || !dataLine) continue;
      onEvent({ event, data: JSON.parse(dataLine.slice(6)) } as ChatStreamEvent);
    }
  }
}

export const chatApi = {
  send: (data: ChatRequest) =>
    api.post<ChatResponse>('/chat', data).then(r => r.data),
  stream: streamChat,
  conversations: (agentId?: string) =>
    api.get<Conversation[]>('/chat/conversations', { params: { agentId } }).then(r => r.data),
  messages: (conversationId: string) =>
    api.get<ChatMessage[]>(`/chat/conversations/${conversationId}/messages`).then(r => r.data),
};

export const shareApi = {
  create: (conversationId: string) =>
    api.post<{ token: string; createdAt: string }>(`/chat/conversations/${conversationId}/share`).then(r => r.data),
  list: () =>
    api.get<{ shares: Array<{ token: string; agentName: string; title: string; createdAt: string }> }>('/chat/share').then(r => r.data),
  revoke: (token: string) =>
    api.delete(`/chat/share/${token}`).then(r => r.data),
  getPublic: (token: string) =>
    publicApi.get<{ token: string; agentId: string; agentName: string; title: string; messages: Array<{ role: string; content: string }>; createdAt: string }>(`/public/share/${token}`).then(r => r.data),
};

// --- Billing ---
export const billingApi = {
  checkout: (data: { planId?: string; bundleId?: string; billingInterval?: string; seatQuantity?: number; removeBillingWaiver?: boolean; paymentMethod?: 'stripe' | 'wechat_h5' | 'alipay' }) =>
    api.post<{ checkoutUrl?: string; waived?: boolean; outTradeNo?: string }>('/billing/checkout', data).then(r => r.data),
  portal: () =>
    api.post<{ portalUrl: string }>('/billing/portal').then(r => r.data),
  listTransactions: (params?: { page?: number; perPage?: number }) =>
    api.get<{ transactions: FinancialTransaction[]; total: number; page: number; perPage: number }>('/billing/transactions', { params }).then(r => r.data),
  getInvoice: (id: string) =>
    api.get<{ transaction: FinancialTransaction; tenant: { name: string } }>(`/billing/transactions/${id}/invoice`).then(r => r.data),
  getInvoicePDF: (id: string) =>
    api.get(`/billing/transactions/${id}/invoice/pdf`, { responseType: 'blob' }).then(r => r.data),
  cancel: () =>
    api.post<{ message: string; currentPeriodEnd?: string }>('/billing/cancel').then(r => r.data),
  getConfig: () =>
    api.get<{ publishableKey: string; paymentMethods: string[] }>('/billing/config').then(r => r.data),
  getPaymentStatus: (outTradeNo: string) =>
    api.get<{ status: 'pending' | 'completed' | 'failed'; provider: string; credits: number; outTradeNo: string }>('/billing/payment/status', { params: { outTradeNo } }).then(r => r.data),
};

// --- Branding (public, no auth) ---
export const brandingApi = {
  get: () =>
    api.get<BrandingConfig>('/branding').then(r => r.data),
  getPublicPages: () =>
    api.get<{ pages: CustomPage[] }>('/branding/pages').then(r => r.data),
  getPublicPage: (slug: string) =>
    api.get<CustomPage>(`/branding/page/${slug}`).then(r => r.data),
};

// --- Branding Admin ---
export const brandingAdminApi = {
  update: (data: Partial<BrandingConfig>) =>
    api.put('/admin/branding', data).then(r => r.data),
  uploadAsset: (key: 'logo' | 'favicon', file: File) => {
    const form = new FormData();
    form.append('key', key);
    form.append('file', file);
    return api.post('/admin/branding/asset', form, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data);
  },
  deleteAsset: (key: 'logo' | 'favicon') =>
    api.delete(`/admin/branding/asset/${key}`).then(r => r.data),
  listMedia: () =>
    api.get<{ media: MediaItem[] }>('/admin/branding/media').then(r => r.data),
  uploadMedia: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return api.post<MediaItem>('/admin/branding/media', form, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data);
  },
  deleteMedia: (key: string) =>
    api.delete(`/admin/branding/media/${key}`).then(r => r.data),
  listPages: () =>
    api.get<{ pages: CustomPage[] }>('/admin/branding/pages').then(r => r.data),
  createPage: (data: Partial<CustomPage>) =>
    api.post<CustomPage>('/admin/branding/pages', data).then(r => r.data),
  updatePage: (id: string, data: Partial<CustomPage>) =>
    api.put(`/admin/branding/pages/${id}`, data).then(r => r.data),
  deletePage: (id: string) =>
    api.delete(`/admin/branding/pages/${id}`).then(r => r.data),
};

// --- PM Dashboard ---
export const pmApi = {
  getFunnel: (params?: { range?: string }) =>
    api.get<FunnelData>('/admin/pm/funnel', { params }).then(r => r.data),
  getRetention: (params?: { granularity?: string; periods?: number }) =>
    api.get<{ granularity: string; periods: number; cohorts: CohortRow[] }>('/admin/pm/retention', { params }).then(r => r.data),
  getEngagement: (params?: { range?: string }) =>
    api.get<EngagementData>('/admin/pm/engagement', { params }).then(r => r.data),
  getKPIs: () =>
    api.get<KPIData>('/admin/pm/kpis').then(r => r.data),
  getCustomEvents: (params?: { name?: string; range?: string }) =>
    api.get<CustomEventData>('/admin/pm/events', { params }).then(r => r.data),
  listEventTypes: () =>
    api.get<{ eventTypes: EventTypeSummary[] }>('/admin/pm/events/types').then(r => r.data),
  listEventDefinitions: (params?: { range?: string }) =>
    api.get<{ definitions: EventDefinition[] }>('/admin/pm/event-definitions', { params }).then(r => r.data),
  createEventDefinition: (data: { name: string; description: string; parentId?: string | null }) =>
    api.post<EventDefinition>('/admin/pm/event-definitions', data).then(r => r.data),
  updateEventDefinition: (id: string, data: { name: string; description: string; parentId?: string | null }) =>
    api.put<EventDefinition>(`/admin/pm/event-definitions/${id}`, data).then(r => r.data),
  deleteEventDefinition: (id: string) =>
    api.delete(`/admin/pm/event-definitions/${id}`).then(r => r.data),
  getSankeyData: (params?: { range?: string }) =>
    api.get<SankeyData>('/admin/pm/event-definitions/sankey', { params }).then(r => r.data),
};

// --- Telemetry ---
export const telemetryApi = {
  trackAnonymous: (data: { sessionId: string; event: string; properties?: Record<string, unknown> }) =>
    api.post('/telemetry/track', data).then(r => r.data),
  trackEvent: (data: { event: string; properties?: Record<string, unknown> }) =>
    api.post('/telemetry/events', data).then(r => r.data),
  trackBatch: (events: { event: string; properties?: Record<string, unknown> }[]) =>
    api.post('/telemetry/events/batch', { events }).then(r => r.data),
};

export default api;
