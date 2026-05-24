import client from './client';
import type { ApiResponse, Team, APIKeySummary, QuotaInfo, LogItem } from '../types/api';

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

// --- 成员 API Key ---
export async function getMyApiKeys() {
  return client.get<ApiResponse<APIKeySummary[]>>('/me/api-keys');
}

export async function getMyKey(slug: string) {
  return client.get<ApiResponse<{ key: string }>>(`/teams/${slug}/members/me/key`);
}

export async function createMyKey(slug: string) {
  return client.post<ApiResponse<{ key: string }>>(`/teams/${slug}/members/me/key`);
}

export async function toggleMyKey(slug: string) {
  return client.put<ApiResponse<{ key_status: number }>>(`/teams/${slug}/members/me/key`);
}

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
export async function getMemberLogs(slug: string) {
  return client.get<ApiResponse<LogItem[]>>(`/teams/${slug}/members/me/logs`);
}
