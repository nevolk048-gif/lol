export type Role = "SUPER_ADMIN" | "ADMIN" | "SUPPORT" | "ANALYST";

export type EntityStatus = "ACTIVE" | "INACTIVE" | "BLOCKED";

export type TransactionStatus =
  | "NEW"
  | "ASSIGNED"
  | "WAITING_PAYMENT"
  | "PAID"
  | "EXPIRED"
  | "CANCELLED";

export type RequisiteStatus = "ACTIVE" | "INACTIVE" | "EXHAUSTED";

export type DisputeStatus =
  | "NEW"
  | "UNDER_REVIEW"
  | "AWAITING_PROVIDER_RESPONSE"
  | "MERCHANT_WON"
  | "PROVIDER_WON"
  | "CLOSED";

export interface User {
  id: string;
  email: string;
  role: Role;
  status: EntityStatus;
  created_at: string;
  updated_at: string;
}

export interface Casino {
  id: string;
  name: string;
  api_key: string;
  webhook_url?: string;
  ip_whitelist?: string[];
  status: EntityStatus;
  is_sandbox: boolean;
  turnover?: number;
  transaction_count?: number;
  conversion_rate?: number;
  created_at: string;
  updated_at: string;
}

export interface Provider {
  id: string;
  name: string;
  api_key: string;
  webhook_url?: string;
  ip_whitelist?: string[];
  status: EntityStatus;
  is_sandbox: boolean;
  traffic_enabled?: boolean;
  traffic_disabled_reason?: string;
  traffic_disabled_at?: string;
  traffic_disabled_by?: string;
  turnover?: number;
  transaction_count?: number;
  conversion_rate?: number;
  avg_response_ms?: number;
  created_at: string;
  updated_at: string;
}

export interface Requisite {
  id: string;
  provider_id: string;
  bank_name: string;
  holder_name: string;
  account_number: string;
  currency: string;
  country: string;
  daily_limit: number;
  used_limit: number;
  status: RequisiteStatus;
  is_sandbox: boolean;
  created_at: string;
  updated_at: string;
}

export interface Transaction {
  id: string;
  external_id?: string;
  casino_id: string;
  provider_id?: string;
  requisite_id?: string;
  amount: number;
  currency: string;
  country: string;
  status: TransactionStatus;
  player_id?: string;
  is_sandbox: boolean;
  processing_ms?: number;
  created_at: string;
  updated_at: string;
  assigned_at?: string;
  paid_at?: string;
  casino_name?: string;
  provider_name?: string;
  requisite_bank?: string;
}

export interface RouteRule {
  id: string;
  priority: number;
  weight: number;
  country?: string;
  currency?: string;
  provider_id: string;
  provider_name?: string;
  status: EntityStatus;
  is_fallback: boolean;
  is_sandbox: boolean;
  created_at: string;
  updated_at: string;
}

export interface Dispute {
  id: string;
  transaction_id: string;
  provider_id: string;
  casino_id: string;
  status: DisputeStatus;
  reason: string;
  amount: number;
  currency: string;
  created_by?: string;
  resolved_by?: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
  provider_name?: string;
  casino_name?: string;
}

export interface DisputeMessage {
  id: string;
  dispute_id: string;
  sender_type: string;
  sender_id: string;
  message: string;
  attachments?: Record<string, unknown>;
  created_at: string;
}

export interface AuditLog {
  id: string;
  user_id?: string;
  action: string;
  entity_type: string;
  entity_id?: string;
  details?: Record<string, unknown>;
  ip_address: string;
  user_email?: string;
  created_at: string;
}

export interface IntegrationLog {
  id: string;
  endpoint: string;
  method: string;
  status_code: number;
  duration_ms: number;
  provider_id?: string;
  casino_id?: string;
  transaction_id?: string;
  error_message?: string;
  is_sandbox: boolean;
  created_at: string;
}

export interface DashboardStats {
  turnover_day: number;
  turnover_week: number;
  turnover_month: number;
  profit: number;
  transaction_count: number;
  active_providers: number;
  active_requisites: number;
  conversion_rate: number;
  avg_processing_ms: number;
}

export interface ChartPoint {
  label: string;
  value: number;
}

export interface DistributionPoint {
  name: string;
  value: number;
}

export interface RecentEvent {
  id: string;
  type: string;
  message: string;
  timestamp: string;
}

export interface DashboardData {
  stats: DashboardStats;
  turnover_hourly: ChartPoint[];
  turnover_daily: ChartPoint[];
  by_provider: DistributionPoint[];
  by_casino: DistributionPoint[];
  by_country: DistributionPoint[];
  recent_events: RecentEvent[];
}

export interface FinanceData {
  turnover: number;
  profit: number;
  commissions: number;
  payouts: number;
  profit_daily: ChartPoint[];
  profit_by_casino: DistributionPoint[];
  profit_by_provider: DistributionPoint[];
}

export interface MonitoringData {
  rps: number;
  active_connections: number;
  ws_connections: number;
  error_rate: number;
  avg_latency_ms: number;
  provider_load: DistributionPoint[];
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
  meta?: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

export interface WSEvent {
  type: string;
  payload: unknown;
  timestamp: string;
}
