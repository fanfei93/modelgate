import client from './client';
import type { ApiResponse, Team, QuotaInfo, LogItem, PaginatedLogs, LogsQuery, LogKey, UserAPIKey } from '../types/api';

// --- 用户级别 API Key（不关联团队，支持多 Key） ---
export async function createMyKey(name: string) {
  return client.post<ApiResponse<{ id: number; key: string; key_mask: string; name: string; status: number }>>('/me/keys', { name });
}

export async function listMyKeys() {
  return client.get<ApiResponse<UserAPIKey[]>>('/me/keys');
}

export async function getMyKey(keyId: number) {
  return client.get<ApiResponse<{ id: number; key: string; key_mask: string; name: string; status: number }>>(`/me/keys/${keyId}`);
}

export async function toggleMyKey(keyId: number) {
  return client.put<ApiResponse<{ key_status: number }>>(`/me/keys/${keyId}/toggle`);
}

export async function deleteMyKey(keyId: number) {
  return client.delete<ApiResponse>(`/me/keys/${keyId}`);
}

// --- 团队 CRUD ---
export async function createTeam(name: string) {
  return client.post<ApiResponse<Team>>('/teams', { name });
}

export async function getTeams() {
  return client.get<ApiResponse<Team[]>>('/teams');
}

export async function getTeam(slug: string) {
  return client.get<ApiResponse<Team>>(`/teams/${slug}`);
}

export async function deleteTeam(slug: string) {
  return client.delete<ApiResponse>(`/teams/${slug}`);
}

// --- 成员 API Key（保留旧接口，TeamDetail 中设置配额时会用到内部逻辑） ---

// --- 成员管理 ---
export async function addMembers(slug: string, entries: { name: string; email: string }[]) {
  return client.post<ApiResponse<{ added: number; failed: string[] }>>(`/teams/${slug}/members`, { entries });
}

export async function removeMember(slug: string, memberId: number) {
  return client.delete<ApiResponse>(`/teams/${slug}/members/${memberId}`);
}

export async function cancelInvitation(slug: string, invitationId: number) {
  return client.delete<ApiResponse>(`/teams/${slug}/invitations/${invitationId}`);
}

// --- 成员额度管理 ---
export async function getMemberQuota(slug: string, memberId: number) {
  return client.get<ApiResponse<QuotaInfo>>(`/teams/${slug}/members/${memberId}/quota`);
}

export async function setMemberQuota(slug: string, memberId: number, amount: number) {
  return client.put<ApiResponse>(`/teams/${slug}/members/${memberId}/quota`, { amount });
}

export async function revokeMemberQuota(slug: string, memberId: number) {
  return client.delete<ApiResponse>(`/teams/${slug}/members/${memberId}/quota`);
}

// --- 成员日志 ---
export async function getMemberLogs(slug: string, query?: LogsQuery) {
  const params = new URLSearchParams();
  if (query?.page) params.set('page', String(query.page));
  if (query?.page_size) params.set('page_size', String(query.page_size));
  if (query?.token_id) params.set('token_id', String(query.token_id));
  if (query?.token_name) params.set('token_name', query.token_name);
  if (query?.model_name) params.set('model_name', query.model_name);
  if (query?.start_timestamp) params.set('start_timestamp', String(query.start_timestamp));
  if (query?.end_timestamp) params.set('end_timestamp', String(query.end_timestamp));
  const qs = params.toString();
  return client.get<ApiResponse<PaginatedLogs>>(`/teams/${slug}/members/me/logs${qs ? '?' + qs : ''}`);
}

export async function getMemberLogKeys(slug: string) {
  return client.get<ApiResponse<LogKey[]>>(`/teams/${slug}/members/me/log-keys`);
}
