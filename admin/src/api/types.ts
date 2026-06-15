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

// --- Рост (growth analytics) ---

export interface Activation {
  signups: number;
  activated: number;
  median_hours: number;
}

export interface RetentionCohort {
  week: string;
  size: number;
  d1_eligible: number;
  d1_retained: number;
  d7_eligible: number;
  d7_retained: number;
  d30_eligible: number;
  d30_retained: number;
}

export interface ConversationDepth {
  bucket_1: number;
  bucket_2_4: number;
  bucket_5_9: number;
  bucket_10: number;
  avg: number;
  median: number;
  p90: number;
}

export interface HeatCell {
  dow: number;
  hour: number;
  count: number;
}

export interface WeekPoint {
  week: string;
  messages: number;
  active_users: number;
}

// --- Качество ответов (answer quality) ---

export interface CTRSlug {
  slug: string;
  impressions: number;
  taps: number;
}

export interface CTROverview {
  cards: number;
  impressions: number;
  taps: number;
  per_slug: CTRSlug[];
}

export interface LengthVsVerdict {
  avg_up: number;
  avg_down: number;
  avg_none: number;
  n_up: number;
  n_down: number;
  n_none: number;
}

export interface NegativeConversation {
  conversation: string;
  downs: number;
  last_down: string;
}

export interface Followup {
  answers: number;
  followups: number;
}

export interface DownvotedMessage {
  conversation: string;
  question?: string;
  answer?: string;
  had_card: boolean;
  created_at: string;
}

// --- Надёжность (reliability & ops) ---

export interface WorkerHealth {
  configured: boolean;
  online: boolean;
  latency_ms: number;
  model: string;
  checked_at?: string;
}

export interface DepStatus {
  name: string;
  configured: boolean;
  up: boolean;
  latency_ms: number;
  detail?: string;
}

export interface FailCodeStat {
  code: string;
  count: number;
}

export interface AlertItem {
  level: string;
  metric: string;
  message: string;
}

// --- Стоимость / FinOps ---

export interface UnitEconomics {
  cost_usd: number;
  messages: number;
  users: number;
  tokens_in: number;
  tokens_out: number;
  cost_per_message: number;
  cost_per_user: number;
}

export interface CacheStats {
  input_uncached: number;
  cache_write: number;
  cache_read: number;
  hit_rate: number;
  savings_usd: number;
  has_data: boolean;
}

export interface ModelCost {
  model: string;
  cost_usd: number;
  tokens_in: number;
  tokens_out: number;
  requests: number;
}

export interface CostByKind {
  claude_cost: number;
  claude_calls: number;
  taps: number;
  transcriptions: number;
  transcribe_sec: number;
}

// --- Безопасность (security & abuse) ---

export interface SecurityEvent {
  user?: string;
  action: string;
  ip?: string;
  created_at: string;
}

export interface OtpRequester {
  email: string;
  codes: number;
}

export interface OtpStats {
  issued: number;
  used: number;
  exhausted: number;
  expired_unused: number;
  delivery_rate: number;
  top_requesters: OtpRequester[];
}

export interface SuspiciousIP {
  source: string;
  ip: string;
  count: number;
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
