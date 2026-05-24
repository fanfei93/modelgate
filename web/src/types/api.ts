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
}

export interface Team {
  id: number;
  name: string;
  slug: string;
  new_api_user_id: number;
  owner_id: number;
  balance: number;
  status: string;
  created_at: string;
  updated_at: string;
  members?: TeamMember[];
  invitations?: TeamInvitation[];
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
