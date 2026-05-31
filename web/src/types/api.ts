export interface ApiResponse<T = unknown> {
  success?: boolean;
  message?: string;
  data?: T;
  error?: string;
  token?: string;
  user?: UserBrief;
  items?: T[];
  total?: number;
  page?: number;
  size?: number;
}

export interface UserBrief {
  id: number;
  username: string;
  email: string;
  display_name: string;
  is_admin?: boolean;
}

export interface Team {
  id: number;
  name: string;
  slug: string;
  owner_id: number;
  balance: number;
  status: string;
  created_at: string;
  updated_at: string;
  members?: TeamMember[];
  invitations?: TeamInvitation[];
}

export interface AdminTeamItem {
  id: number;
  name: string;
  slug: string;
  owner_id: number;
  balance: number;
  status: string;
  member_count: number;
  created_at: string;
}

export interface TeamMember {
  id: number;
  team_id: number;
  user_id: number;
  role: 'owner' | 'member';
  joined_at: string;
  api_key_mask?: string;
  quota_allocated: number;
  quota_used: number;
  user?: UserBrief;
}

export interface TeamInvitation {
  id: number;
  team_id: number;
  email: string;
  name: string;
  inviter_id: number;
  status: 'pending' | 'accepted';
  created_at: string;
}

export interface APIKeySummary {
  team_id: number;
  team_name: string;
  team_slug: string;
  role: string;
  has_key: boolean;
  key_status: number;
  api_key_mask: string;
}

export interface QuotaInfo {
  quota_allocated: number;
  quota_used: number;
  quota_remain: number;
  has_key: boolean;
}

export interface LogItem {
  id: number;
  user_id: number;
  created_at: number;
  type: number;
  content: string;
  username: string;
  token_name: string;
  model_name: string;
  quota: number;
  prompt_tokens: number;
  completion_tokens: number;
  use_time: number;
  is_stream: boolean;
  channel: number;
  channel_name: string;
  token_id: number;
  group: string;
  ip: string;
  request_id: string;
  other: string;
}

export interface PaginatedLogs {
  items: LogItem[];
  total: number;
  page: number;
  page_size: number;
}

export interface LogsQuery {
  page?: number;
  page_size?: number;
  token_id?: number;
  token_name?: string;
  model_name?: string;
  start_timestamp?: number;
  end_timestamp?: number;
}

export interface LogKey {
  token_id: number;
  name: string;
}

export interface SiteSetting {
  key: string;
  value: string;
  comment: string;
}

export interface LoginLog {
  id: number;
  user_id: number;
  username: string;
  ip: string;
  user_agent: string;
  success: boolean;
  reason: string;
  created_at: string;
}

export interface UserAPIKey {
  id: number;
  user_id: number;
  name: string;
  key?: string;       // 仅创建/单独获取时返回完整密钥
  key_mask?: string;  // 列表返回脱敏密钥
  status: number;     // 1: 启用, 2: 禁用
  created_at: string;
  updated_at: string;
}

export interface RechargeLog {
  id: number;
  team_id: number;
  team_name: string;
  operator_id: number;
  operator_name: string;
  amount: number;
  balance_before: number;
  balance_after: number;
  remark: string;
  ip: string;
  created_at: string;
}
