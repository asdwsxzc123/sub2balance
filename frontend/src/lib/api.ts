import type {
  PasswordResetConfirmResult,
  PasswordResetQueryResult,
  PasswordResetSettings,
  SystemLatest,
  SystemVersion,
  UpgradeResult,
} from '@/types/api';

const API_BASE = '/api';

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

function getToken(): string | null {
  return localStorage.getItem('token');
}

async function request<T>(method: string, endpoint: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = getToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${endpoint}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const isJson = res.headers.get('content-type')?.includes('application/json');
  const payload = isJson ? await res.json().catch(() => null) : null;

  if (!res.ok) {
    const message = (payload && typeof payload === 'object' && 'error' in payload ? String(payload.error) : null)
      || res.statusText
      || `HTTP ${res.status}`;
    throw new ApiError(message, res.status);
  }

  return payload as T;
}

export const api = {
  get: <T>(endpoint: string) => request<T>('GET', endpoint),
  post: <T>(endpoint: string, body?: unknown) => request<T>('POST', endpoint, body),
  put: <T>(endpoint: string, body?: unknown) => request<T>('PUT', endpoint, body),
  delete: <T>(endpoint: string) => request<T>('DELETE', endpoint),
};

export function passwordResetQuery(email: string) {
  return api.post<PasswordResetQueryResult>('/password-reset/query', { email });
}

export function passwordResetConfirm(email: string) {
  return api.post<PasswordResetConfirmResult>('/password-reset', { email });
}

export function getPasswordResetSettings() {
  return api.get<PasswordResetSettings>('/admin/settings/password-reset');
}

export function updatePasswordResetSettings(dailyLimit: number) {
  return api.put<PasswordResetSettings>('/admin/settings/password-reset', {
    daily_limit: dailyLimit,
  });
}

export function getSystemVersion() {
  return api.get<SystemVersion>('/admin/system/version');
}

export function getSystemLatest() {
  return api.get<SystemLatest>('/admin/system/latest');
}

export function triggerSystemUpgrade(version = '') {
  return api.post<UpgradeResult>('/admin/system/upgrade', { version });
}
