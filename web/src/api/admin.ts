import client from './client';
import type { ApiResponse, AdminTeamItem } from '../types/api';

export async function listTeams() {
  return client.get<ApiResponse<AdminTeamItem[]>>('/admin/teams');
}

export async function rechargeTeam(slug: string, amount: number) {
  return client.post<ApiResponse>(`/admin/teams/${slug}/recharge`, { amount });
}
