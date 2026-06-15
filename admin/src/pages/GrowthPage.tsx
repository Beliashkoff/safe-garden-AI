import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  CircularProgress,
  Grid,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
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
  Tooltip as ReTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { api, humanError } from '../api/client';
import { StatCard } from '../components/StatCard';
import type { Activation, ConversationDepth, HeatCell, RetentionCohort, WeekPoint } from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');

// Дни недели Postgres: 0=Вс..6=Сб. Показываем с понедельника.
const DOW_ORDER = [1, 2, 3, 4, 5, 6, 0];
const DOW_LABELS: Record<number, string> = {
  1: 'Пн',
  2: 'Вт',
  3: 'Ср',
  4: 'Чт',
  5: 'Пт',
  6: 'Сб',
  0: 'Вс',
};

// Зелёный фон с прозрачностью по интенсивности 0..1 (для heatmap и retention).
function greenBg(intensity: number): string {
  if (intensity <= 0) return 'transparent';
  const a = 0.1 + 0.85 * Math.min(1, intensity);
  return `rgba(46, 125, 50, ${a.toFixed(3)})`;
}

function retCell(retained: number, eligible: number) {
  if (eligible <= 0) {
    return { label: '—', bg: 'transparent' as string };
  }
  const p = Math.round((retained / eligible) * 100);
  return { label: `${p}%`, bg: greenBg(p / 100) };
}

export function GrowthPage() {
  const [activation, setActivation] = useState<Activation | null>(null);
  const [cohorts, setCohorts] = useState<RetentionCohort[]>([]);
  const [depth, setDepth] = useState<ConversationDepth | null>(null);
  const [heat, setHeat] = useState<HeatCell[]>([]);
  const [weekly, setWeekly] = useState<WeekPoint[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const [a, ret, d, hm, wk] = await Promise.all([
          api.get<Activation>('/stats/activation?days=30'),
          api.get<{ cohorts: RetentionCohort[] }>('/stats/retention?days=84'),
          api.get<ConversationDepth>('/stats/conversation-depth?days=30'),
          api.get<{ cells: HeatCell[] }>('/stats/heatmap?days=30'),
          api.get<{ points: WeekPoint[] }>('/stats/activity-weekly?days=180'),
        ]);
        setActivation(a);
        setCohorts(ret.cohorts);
        setDepth(d);
        setHeat(hm.cells);
        setWeekly(wk.points);
      } catch (err) {
        setError(humanError(err));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (loading || !activation || !depth) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const actRate = activation.signups > 0 ? Math.round((activation.activated / activation.signups) * 100) : 0;

  const depthData = [
    { bucket: '1', users: depth.bucket_1 },
    { bucket: '2–4', users: depth.bucket_2_4 },
    { bucket: '5–9', users: depth.bucket_5_9 },
    { bucket: '10+', users: depth.bucket_10 },
  ];

  const heatMap: Record<string, number> = {};
  let heatMax = 0;
  for (const c of heat) {
    heatMap[`${c.dow}-${c.hour}`] = c.count;
    if (c.count > heatMax) heatMax = c.count;
  }

  return (
    <Box>
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 4 }}>
          <StatCard
            title="Активация новых (30 дней)"
            value={`${actRate}%`}
            hint={`${nf.format(activation.activated)} из ${nf.format(activation.signups)} задали вопрос`}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <StatCard
            title="До первого вопроса"
            value={activation.activated > 0 ? `${activation.median_hours.toFixed(1)} ч` : '—'}
            hint="медиана: регистрация → первый вопрос"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <StatCard
            title="Глубина диалога"
            value={`${depth.median.toFixed(1)}`}
            hint={`медиана вопросов · среднее ${depth.avg.toFixed(1)} · p90 ${depth.p90.toFixed(0)}`}
          />
        </Grid>
      </Grid>

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 0.5 }}>
          Когортное удержание по неделям
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Доля вернувшихся (вернулся / созревших). «—» — когорта ещё не дожила до этого дня. Активация —
          прокси: первый вопрос, не «фото в первой сессии» (его в данных нет).
        </Typography>
        {cohorts.length === 0 ? (
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            Пока нет когорт за период.
          </Typography>
        ) : (
          <Table size="small" sx={{ mt: 1 }}>
            <TableHead>
              <TableRow>
                <TableCell>Неделя</TableCell>
                <TableCell align="right">Размер</TableCell>
                <TableCell align="center">D1</TableCell>
                <TableCell align="center">D7</TableCell>
                <TableCell align="center">D30</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {cohorts.map((c) => {
                const d1 = retCell(c.d1_retained, c.d1_eligible);
                const d7 = retCell(c.d7_retained, c.d7_eligible);
                const d30 = retCell(c.d30_retained, c.d30_eligible);
                return (
                  <TableRow key={c.week}>
                    <TableCell sx={{ whiteSpace: 'nowrap' }}>{c.week}</TableCell>
                    <TableCell align="right">{nf.format(c.size)}</TableCell>
                    <TableCell align="center" sx={{ bgcolor: d1.bg }}>
                      {d1.label}
                    </TableCell>
                    <TableCell align="center" sx={{ bgcolor: d7.bg }}>
                      {d7.label}
                    </TableCell>
                    <TableCell align="center" sx={{ bgcolor: d30.bg }}>
                      {d30.label}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Paper>

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Глубина диалога (вопросов на пользователя, 30 дней)
            </Typography>
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={depthData} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
                <XAxis dataKey="bucket" tick={{ fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                <ReTooltip />
                <Bar dataKey="users" name="Пользователи" fill="#2E7D32" />
              </BarChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Сезонная активность по неделям
            </Typography>
            {weekly.length === 0 ? (
              <Typography color="text.secondary">Пока нет данных за период.</Typography>
            ) : (
              <ResponsiveContainer width="100%" height={240}>
                <LineChart data={weekly} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
                  <XAxis dataKey="week" tick={{ fontSize: 11 }} />
                  <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                  <ReTooltip />
                  <Legend />
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
                    dataKey="active_users"
                    name="Активные"
                    stroke="#1976D2"
                    strokeWidth={2}
                    dot={false}
                  />
                </LineChart>
              </ResponsiveContainer>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 0.5 }}>
          Активность по часам (МСК, 30 дней)
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Чем темнее, тем больше вопросов в этот час. Пиковые окна — для рассылок; провалы — для техработ.
        </Typography>
        {heatMax === 0 ? (
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            Пока нет данных за период.
          </Typography>
        ) : (
          <Box sx={{ mt: 1.5, overflowX: 'auto' }}>
            <Box sx={{ minWidth: 560 }}>
              <Box
                sx={{
                  display: 'grid',
                  gridTemplateColumns: '32px repeat(24, 1fr)',
                  gap: '2px',
                  alignItems: 'center',
                }}
              >
                <Box />
                {Array.from({ length: 24 }, (_, h) => (
                  <Typography
                    key={`h-${h}`}
                    variant="caption"
                    sx={{ textAlign: 'center', fontSize: 9, color: 'text.secondary' }}
                  >
                    {h % 3 === 0 ? h : ''}
                  </Typography>
                ))}
                {DOW_ORDER.map((dow) => (
                  <Box key={`row-${dow}`} sx={{ display: 'contents' }}>
                    <Typography variant="caption" sx={{ fontSize: 11, color: 'text.secondary' }}>
                      {DOW_LABELS[dow]}
                    </Typography>
                    {Array.from({ length: 24 }, (_, h) => {
                      const count = heatMap[`${dow}-${h}`] ?? 0;
                      return (
                        <Tooltip key={`${dow}-${h}`} title={`${DOW_LABELS[dow]} ${h}:00 — ${count}`}>
                          <Box
                            sx={{
                              aspectRatio: '1 / 1',
                              minHeight: 14,
                              borderRadius: 0.5,
                              bgcolor: greenBg(count / heatMax),
                              border: '1px solid #EEF2EE',
                            }}
                          />
                        </Tooltip>
                      );
                    })}
                  </Box>
                ))}
              </Box>
            </Box>
          </Box>
        )}
      </Paper>
    </Box>
  );
}
