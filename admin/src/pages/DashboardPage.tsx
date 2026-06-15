import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { api, humanError } from '../api/client';
import type {
  CostUser,
  DayPoint,
  FeedbackPoint,
  MessageStatusPoint,
  Overview,
  ProviderStat,
  TapStat,
} from '../api/types';

function StatCard({
  title,
  value,
  hint,
  valueColor,
}: {
  title: string;
  value: string;
  hint?: string;
  valueColor?: string;
}) {
  return (
    <Card variant="outlined" sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="body2" color="text.secondary">
          {title}
        </Typography>
        <Typography variant="h5" sx={{ my: 0.5 }} color={valueColor}>
          {value}
        </Typography>
        {hint && (
          <Typography variant="caption" color="text.secondary">
            {hint}
          </Typography>
        )}
      </CardContent>
    </Card>
  );
}

const nf = new Intl.NumberFormat('ru-RU');
const money = (n: number) => `$${n.toFixed(2)}`;

const PROVIDER_LABELS: Record<string, string> = {
  yandex: 'Яндекс ID',
  vk: 'VK ID',
  email: 'Email-код',
};

// Доля в процентах или прочерк, когда знаменатель нулевой.
function pct(part: number, whole: number): string {
  if (whole <= 0) return '—';
  return `${Math.round((part / whole) * 100)}%`;
}

// Цвет значения success-rate по порогам здоровья.
function rateColor(part: number, whole: number): string | undefined {
  if (whole <= 0) return undefined;
  const r = (part / whole) * 100;
  if (r >= 95) return 'success.main';
  if (r >= 80) return 'warning.main';
  return 'error.main';
}

// Скользящее 7-дневное среднее стоимости поверх дневного ряда.
function costSeries(points: DayPoint[]) {
  return points.map((p, i, arr) => {
    const window = arr.slice(Math.max(0, i - 6), i + 1);
    const ma = window.reduce((s, x) => s + x.cost_usd, 0) / window.length;
    return { day: p.day, cost: Number(p.cost_usd.toFixed(4)), ma: Number(ma.toFixed(4)) };
  });
}

export function DashboardPage() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [points, setPoints] = useState<DayPoint[]>([]);
  const [taps, setTaps] = useState<TapStat[]>([]);
  const [feedback, setFeedback] = useState<FeedbackPoint[]>([]);
  const [statuses, setStatuses] = useState<MessageStatusPoint[]>([]);
  const [providers, setProviders] = useState<ProviderStat[]>([]);
  const [topUsers, setTopUsers] = useState<CostUser[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    void (async () => {
      try {
        const [o, ts, top, fb, ms, lb, tu] = await Promise.all([
          api.get<Overview>('/stats/overview'),
          api.get<{ points: DayPoint[] }>('/stats/timeseries?days=30'),
          api.get<{ products: TapStat[] }>('/stats/top-products?days=30'),
          api.get<{ points: FeedbackPoint[] }>('/stats/feedback?days=30'),
          api.get<{ points: MessageStatusPoint[] }>('/stats/message-status?days=30'),
          api.get<{ providers: ProviderStat[] }>('/stats/login-breakdown?days=30'),
          api.get<{ users: CostUser[] }>('/stats/top-users?days=7'),
        ]);
        setOverview(o);
        setPoints(ts.points);
        setTaps(top.products);
        setFeedback(fb.points);
        setStatuses(ms.points);
        setProviders(lb.providers);
        setTopUsers(tu.users);
      } catch (err) {
        setError(humanError(err));
      }
    })();
  }, []);

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (!overview) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const answers7d =
    overview.answers_complete_7d + overview.answers_failed_7d + overview.answers_cancelled_7d;
  const votes = overview.feedback_up_30d + overview.feedback_down_30d;
  const stickiness = overview.mau > 0 ? Math.round((overview.dau / overview.mau) * 100) : 0;
  const purgeStale = overview.media_purge_pending > 0 && overview.media_purge_oldest_hours > 48;

  return (
    <Box>
      <Grid container spacing={2}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Пользователи"
            value={nf.format(overview.users_total)}
            hint={`+${nf.format(overview.users_new_7d)} за 7 дней`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Активные (DAU)"
            value={nf.format(overview.dau)}
            hint={`WAU ${nf.format(overview.wau)} · MAU ${nf.format(overview.mau)} · липкость ${stickiness}%`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Вопросы к ИИ"
            value={nf.format(overview.messages_total)}
            hint={`${nf.format(overview.messages_7d)} за 7 дней`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Успешные ответы ИИ"
            value={pct(overview.answers_complete_7d, answers7d)}
            hint={`за 7 дней · сбоев ${nf.format(overview.answers_failed_7d)}`}
            valueColor={rateColor(overview.answers_complete_7d, answers7d)}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Оценки ответов"
            value={votes > 0 ? `${pct(overview.feedback_up_30d, votes)} 👍` : '—'}
            hint={`${nf.format(overview.feedback_up_30d)}↑ / ${nf.format(overview.feedback_down_30d)}↓ · оценено ${Math.round(overview.feedback_coverage_30d * 100)}%`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Прогноз расхода / мес"
            value={money(overview.cost_forecast_month)}
            hint={`MTD ${money(overview.cost_mtd)} · пред. мес ${money(overview.cost_prev_month)}`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Расход ИИ"
            value={money(overview.cost_usd_30d)}
            hint="за 30 дней"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Токены (вход/выход)"
            value={`${nf.format(overview.tokens_in_30d)} / ${nf.format(overview.tokens_out_30d)}`}
            hint="за 30 дней"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Клики по товарам"
            value={nf.format(overview.taps_30d)}
            hint="за 30 дней"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Товары в каталоге"
            value={`${nf.format(overview.catalog_active)} / ${nf.format(overview.catalog_total)}`}
            hint="активные / всего"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Удаление аккаунтов"
            value={nf.format(overview.media_purge_pending)}
            hint={
              purgeStale
                ? `в очереди · застряло ${Math.round(overview.media_purge_oldest_hours)} ч · всего ${nf.format(overview.accounts_deleted_total)}`
                : `в очереди на очистку · всего ${nf.format(overview.accounts_deleted_total)}`
            }
            valueColor={purgeStale ? 'error.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Ошибки сервера"
            value={nf.format(overview.errors_24h)}
            hint="за 24 часа"
            valueColor={overview.errors_24h > 0 ? 'warning.main' : undefined}
          />
        </Grid>
      </Grid>

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Активность за 30 дней
        </Typography>
        {points.length === 0 ? (
          <Typography color="text.secondary">Пока нет данных за период.</Typography>
        ) : (
          <ResponsiveContainer width="100%" height={280}>
            <LineChart data={points} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
              <XAxis dataKey="day" tick={{ fontSize: 12 }} />
              <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
              <Tooltip />
              <Line
                type="monotone"
                dataKey="messages"
                name="Вопросы"
                stroke="#2E7D32"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="users"
                name="Новые пользователи"
                stroke="#1976D2"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Paper>

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Расход ИИ по дням ($)
        </Typography>
        {points.length === 0 ? (
          <Typography color="text.secondary">Пока нет данных за период.</Typography>
        ) : (
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={costSeries(points)} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
              <XAxis dataKey="day" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip formatter={(value) => money(Number(value))} />
              <Legend />
              <Line
                type="monotone"
                dataKey="cost"
                name="Расход за день"
                stroke="#2E7D32"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="ma"
                name="Среднее за 7 дней"
                stroke="#EF6C00"
                strokeWidth={2}
                strokeDasharray="5 4"
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Paper>

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Оценки ответов по дням
            </Typography>
            {feedback.length === 0 ? (
              <Typography color="text.secondary">
                Пока нет оценок ответов. Они появятся, когда пользователи начнут оценивать ответы ИИ.
              </Typography>
            ) : (
              <ResponsiveContainer width="100%" height={240}>
                <BarChart data={feedback} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
                  <XAxis dataKey="day" tick={{ fontSize: 12 }} />
                  <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="up" name="👍" stackId="fb" fill="#2E7D32" />
                  <Bar dataKey="down" name="👎" stackId="fb" fill="#C62828" />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Статусы ответов ИИ по дням
            </Typography>
            {statuses.length === 0 ? (
              <Typography color="text.secondary">Пока нет ответов за период.</Typography>
            ) : (
              <ResponsiveContainer width="100%" height={240}>
                <BarChart data={statuses} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
                  <XAxis dataKey="day" tick={{ fontSize: 12 }} />
                  <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="complete" name="Успешно" stackId="st" fill="#2E7D32" />
                  <Bar dataKey="failed" name="Сбой" stackId="st" fill="#C62828" />
                  <Bar dataKey="cancelled" name="Отменено" stackId="st" fill="#9E9E9E" />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 5 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Способы входа (30 дней)
            </Typography>
            {providers.length === 0 ? (
              <Typography color="text.secondary">Пока нет входов за период.</Typography>
            ) : (
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Способ</TableCell>
                    <TableCell align="right">Входы</TableCell>
                    <TableCell align="right">Пользователи</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {providers.map((p) => (
                    <TableRow key={p.provider}>
                      <TableCell>{PROVIDER_LABELS[p.provider] ?? p.provider}</TableCell>
                      <TableCell align="right">{nf.format(p.logins)}</TableCell>
                      <TableCell align="right">{nf.format(p.users)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 7 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Топ по расходу (7 дней)
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Идентификатор пользователя замаскирован. Резкий перевес одного — повод проверить
              аномальную переписку.
            </Typography>
            {topUsers.length === 0 ? (
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                Пока нет расхода за период.
              </Typography>
            ) : (
              <Table size="small" sx={{ mt: 1 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Пользователь</TableCell>
                    <TableCell align="right">Запросы</TableCell>
                    <TableCell align="right">Токены (вх/вых)</TableCell>
                    <TableCell align="right">Расход</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {topUsers.map((u) => (
                    <TableRow key={u.user}>
                      <TableCell>
                        <code>{u.user}</code>
                      </TableCell>
                      <TableCell align="right">{nf.format(u.requests)}</TableCell>
                      <TableCell align="right">
                        {nf.format(u.tokens_in)} / {nf.format(u.tokens_out)}
                      </TableCell>
                      <TableCell align="right">{money(u.cost_usd)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 1 }}>
          Популярные товары (клики за 30 дней)
        </Typography>
        {taps.length === 0 ? (
          <Typography color="text.secondary">
            Пока нет кликов по карточкам товаров. Они появятся, когда ИИ начнёт рекомендовать
            удобрения из каталога.
          </Typography>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Товар (slug)</TableCell>
                <TableCell align="right">Клики</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {taps.map((t) => (
                <TableRow key={t.slug}>
                  <TableCell>{t.slug}</TableCell>
                  <TableCell align="right">{nf.format(t.taps)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>
    </Box>
  );
}
