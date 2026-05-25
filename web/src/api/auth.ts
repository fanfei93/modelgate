import client from './client';
import type { ApiResponse, UserBrief } from '../types/api';

export async function register(username: string, email: string, password: string, code: string) {
  return client.post<ApiResponse>('/auth/register', { username, email, password, code });
}

export async function login(username: string, password: string) {
  return client.post<ApiResponse>('/auth/login', { username, password });
}

export async function getMe() {
  return client.get<ApiResponse<UserBrief>>('/auth/me');
}

export async function sendVerificationCode(email: string) {
  return client.post<ApiResponse>('/auth/send-verification-code', { email });
}
