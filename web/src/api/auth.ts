import client from './client';
import type { ApiResponse, UserBrief } from '../types/api';

export async function register(username: string, email: string, password: string) {
  return client.post<ApiResponse>('/auth/register', { username, email, password });
}

export async function login(username: string, password: string) {
  return client.post<ApiResponse>('/auth/login', { username, password });
}

export async function getMe() {
  return client.get<ApiResponse<UserBrief>>('/auth/me');
}
