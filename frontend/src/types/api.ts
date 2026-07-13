export type Role = 'admin' | 'staff';

export interface User {
  id: number;
  email: string;
  role: Role;
  created_at: string;
  updated_at: string;
}

export interface AuthUser {
  id: number;
  email: string;
  role: Role;
}

export interface LoginResponse {
  token: string;
  user: AuthUser;
}

export type ConversionStatus = 'pending' | 'approved' | 'rejected';
export type RequestType = 'balance' | 'switch' | 'bind' | 'unbind';

export interface ConversionRequest {
  id: number;
  request_type: RequestType;
  user_email: string;
  sub2api_user_id: number;
  subscription_id: number;
  group_name: string;
  original_amount: number;
  consumed_amount: number;
  conversion_amount: number;
  final_amount: number | null;
  target_group_id: number | null;
  target_group_name: string | null;
  validity_days: number | null;
  status: ConversionStatus;
  submitted_by: number;
  submitted_by_user?: User;
  reviewed_by: number | null;
  reviewed_by_user?: User;
  review_note: string;
  reviewed_at: string | null;
  created_at: string;
  updated_at: string;
}

export type OriginalSource = 'mapping' | 'parsed' | 'limit';

export interface QueryResult {
  user_email: string;
  sub2api_user_id: number;
  subscription_id: number;
  group_id: number;
  group_name: string;
  platform?: string;
  original_amount: number;
  original_source?: OriginalSource;
  currency?: string;
  consumed_amount: number;
  conversion_amount: number;
  status: string;
  expires_at?: string;
}

export interface GroupPriceRow {
  group_id: number;
  group_name: string;
  platform: string;
  daily_limit_usd: number | null;
  parsed_price?: number;
  price?: number;
  currency?: string;
  note?: string;
}

export interface AvailableGroup {
  id: number;
  name: string;
  platform: string;
  daily_limit_usd: number | null;
}

export interface Sub2APIUser {
  id: number;
  email: string;
  balance: number;
}

export interface QueryByEmailResult {
  user: Sub2APIUser;
  subscriptions: QueryResult[];
}

export interface Sub2APISettings {
  base_url: string;
  api_key_masked: string;
  configured: boolean;
}

export interface PasswordResetSettings {
  daily_limit: number;
}

export interface PasswordResetAccount {
  id: number;
  email: string;
  username: string;
  status: string;
  created_at: string;
}

export interface PasswordResetQueryResult {
  user: PasswordResetAccount;
}

export interface PasswordResetConfirmResult {
  email: string;
  user_id: number;
  new_password: string;
}

export interface AuditLog {
  id: number;
  request_id: number | null;
  user_id: number;
  user?: User;
  action: string;
  details: string;
  created_at: string;
}
