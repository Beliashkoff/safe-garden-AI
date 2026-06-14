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
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { api, humanError } from '../api/client';
import type { DayPoint, Overview, TapStat } from '../api/types';

function StatCard({ title, value, hint }: { title: string; value: string; hint?: string }) {
  return (
    <Card variant="outlined" sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="body2" color="text.secondary">
          {title}
        </Typography>
        <Typography variant="h5" sx={{ my: 0.5 }}>
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

export function DashboardPage() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [points, setPoints] = useState<DayPoint[]>([]);
  const [taps, setTaps] = useState<TapStat[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    void (async () => {
      try {
        const [o, ts, top] = await Promise.all([
          api.get<Overview>('/stats/overview'),
          api.get<{ points: DayPoint[] }>('/stats/timeseries?days=30'),
          api.get<{ products: TapStat[] }>('/stats/top-products?days=30'),
        ]);
        setOverview(o);
        setPoints(ts.points);
        setTaps(top.products);
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
            title="Вопросы к ИИ"
            value={nf.format(overview.messages_total)}
            hint={`${nf.format(overview.messages_7d)} за 7 дней`}
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
            title="Клики по товарам"
            value={nf.format(overview.taps_30d)}
            hint="за 30 дней"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Расход ИИ"
            value={`$${overview.cost_usd_30d.toFixed(2)}`}
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
            title="Ошибки сервера"
            value={nf.format(overview.errors_24h)}
            hint="за 24 часа"
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
