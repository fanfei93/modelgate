import client from './client';
import type { ApiResponse, AdminTeamItem, SiteSetting, LoginLog, RechargeLog, AdminUserItem } from '../types/api';

export async function listTeams() {
  return client.get<ApiResponse<AdminTeamItem[]>>('/admin/teams');
}

export async function rechargeTeam(slug: string, amount: number, remark = '') {
  return client.post<ApiResponse>(`/admin/teams/${slug}/recharge`, { amount, remark });
}

export async function listSettings() {
  return client.get<ApiResponse<SiteSetting[]>>('/admin/settings');
}

export async function updateSetting(key: string, value: string) {
  return client.put<ApiResponse>(`/admin/settings/${key}`, { value });
}

export async function listLoginLogs(page = 1, pageSize = 20) {
  return client.get<ApiResponse<LoginLog[]>>('/admin/login-logs', {
    params: { page, page_size: pageSize },
  });
}

export async function listRechargeLogs(page = 1, pageSize = 20) {
  return client.get<ApiResponse<RechargeLog[]>>('/admin/recharge-logs', {
    params: { page, page_size: pageSize },
  });
}

// 公开接口 - 获取站点配置（无需认证）
export async function getSiteSettings() {
  return client.get<ApiResponse<SiteSetting[]>>('/site-settings');
}

export async function listUsers(page = 1, pageSize = 20, keyword = '') {
  return client.get<ApiResponse<AdminUserItem[]>>('/admin/users', {
    params: { page, page_size: pageSize, keyword },
  });
}

export async function updateUserStatus(id: number, status: string) {
  return client.put<ApiResponse>(`/admin/users/${id}/status`, { status });
}
