import type { AuthResponse, LeaderboardEntry, PlayerProfile, ShopItem } from '../types';

const TOKEN_KEY = 'skywalker_token';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

async function apiFetch<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((opts.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(path, { ...opts, headers });
  const contentType = res.headers.get('content-type') || '';
  if (!contentType.includes('application/json')) {
    const text = await res.text();
    throw new Error(text || `Request failed: ${res.status}`);
  }
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || `Request failed: ${res.status}`);
  }
  return data as T;
}

export async function register(username: string, email: string, password: string): Promise<AuthResponse> {
  const resp = await apiFetch<AuthResponse>('/api/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password }),
  });
  setToken(resp.token);
  return resp;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const resp = await apiFetch<AuthResponse>('/api/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  setToken(resp.token);
  return resp;
}

export function logout() {
  clearToken();
}

export async function getProfile(): Promise<PlayerProfile> {
  return apiFetch<PlayerProfile>('/api/profile');
}

export async function getShopItems(): Promise<ShopItem[]> {
  return apiFetch<ShopItem[]>('/api/shop/items');
}

export async function buyItem(itemId: string): Promise<{ ok: boolean; coins: number }> {
  return apiFetch<{ ok: boolean; coins: number }>('/api/shop/buy', {
    method: 'POST',
    body: JSON.stringify({ item_id: itemId }),
  });
}

export async function equipItem(itemId: string, slot: string): Promise<{ ok: boolean }> {
  return apiFetch<{ ok: boolean }>('/api/loadout/equip', {
    method: 'POST',
    body: JSON.stringify({ item_id: itemId, slot }),
  });
}

export async function getLeaderboard(): Promise<LeaderboardEntry[]> {
  return apiFetch<LeaderboardEntry[]>('/api/leaderboard');
}
