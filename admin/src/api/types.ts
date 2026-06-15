export interface Admin {
  id: string;
  email: string;
  created_at: string;
  last_login_at?: string;
}

export interface Product {
  id: string;
  slug: string;
  name: string;
  short_desc: string;
  long_desc?: string;
  image_url?: string;
  deeplink_url?: string;
  category: string;
  problems: string[];
  plants: string[];
  priority: number;
  active: boolean;
  price_rub?: number | null;
  created_at: string;
  updated_at: string;
}

export interface ProductInput {
  slug: string;
  name: string;
  short_desc: string;
  long_desc: string;
  image_url: string;
  deeplink_url: string;
  category: string;
  problems: string[];
  plants: string[];
  priority: number;
  active: boolean;
  price_rub: number | null;
}

export interface Overview {
  users_total: number;
  users_new_7d: number;
  messages_total: number;
  messages_7d: number;
  tokens_in_30d: number;
  tokens_out_30d: number;
  cost_usd_30d: number;
  taps_30d: number;
  catalog_total: number;
  catalog_active: number;
  errors_24h: number;

  dau: number;
  wau: number;
  mau: number;

  feedback_up_30d: number;
  feedback_down_30d: number;
  feedback_coverage_30d: number;

  answers_complete_7d: number;
  answers_failed_7d: number;
  answers_cancelled_7d: number;

  accounts_deleted_total: number;
  media_purge_pending: number;
  media_purge_oldest_hours: number;

  cost_mtd: number;
  cost_forecast_month: number;
  cost_prev_month: number;
}

export interface FeedbackPoint {
  day: string;
  up: number;
  down: number;
}

export interface MessageStatusPoint {
  day: string;
  complete: number;
  failed: number;
  cancelled: number;
}

export interface ProviderStat {
  provider: string;
  logins: number;
  users: number;
}

export interface CostUser {
  user: string;
  requests: number;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
}

export interface ErrorRouteStat {
  route: string;
  status: number;
  count: number;
}

export interface ErrorDayPoint {
  day: string;
  count: number;
}

export interface ErrorBreakdown {
  by_route: ErrorRouteStat[];
  by_day: ErrorDayPoint[];
}

export interface DayPoint {
  day: string;
  users: number;
  messages: number;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
}

export interface TapStat {
  slug: string;
  taps: number;
}

// Named ServerError (not ErrorEvent) to avoid shadowing the DOM global ErrorEvent.
export interface ServerError {
  id: number;
  source: string;
  route: string;
  method: string;
  status: number;
  request_id?: string;
  message?: string;
  created_at: string;
}

export interface AuditEntry {
  id: number;
  action: string;
  entity?: string;
  entity_id?: string;
  ip?: string;
  created_at: string;
}

export interface Session {
  id: string;
  ip?: string;
  user_agent?: string;
  created_at: string;
  last_seen_at: string;
  current: boolean;
}
