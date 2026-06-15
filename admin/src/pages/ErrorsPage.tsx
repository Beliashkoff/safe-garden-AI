import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Grid,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip as ReTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import RefreshIcon from '@mui/icons-material/Refresh';
import { api, humanError } from '../api/client';
import type { ErrorBreakdown, ServerError } from '../api/types';

const PAGE = 50;
const nf = new Intl.NumberFormat('ru-RU');

function statusColor(status: number): 'error' | 'warning' | 'default' {
  if (status >= 500) return 'error';
  if (status >= 400) return 'warning';
  return 'default';
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

export function ErrorsPage() {
  const [events, setEvents] = useState<ServerError[]>([]);
  const [breakdown, setBreakdown] = useState<ErrorBreakdown | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = async (nextOffset: number, append: boolean) => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get<{ errors: ServerError[] }>(
        `/errors?limit=${PAGE}&offset=${nextOffset}`,
      );
      const list = res.errors ?? [];
      setHasMore(list.length === PAGE);
      setOffset(nextOffset);
      setEvents((prev) => (append ? [...prev, ...list] : list));
    } catch (err) {
      setError(humanError(err));
    } finally {
      setLoading(false);
    }
  };

  const loadSummary = async () => {
    try {
      const res = await api.get<ErrorBreakdown>('/stats/errors-breakdown?days=7');
      setBreakdown(res);
    } catch {
      // Сводка вторична: ошибку показывает основная загрузка ленты.
    }
  };

  const refresh = () => {
    void load(0, false);
    void loadSummary();
  };

  useEffect(() => {
    void load(0, false);
    void loadSummary();
  }, []);

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          Серверные ошибки (5xx) бэкенда. Содержимое сообщений пользователей сюда не попадает.
        </Typography>
        <Button startIcon={<RefreshIcon />} onClick={refresh} disabled={loading}>
          Обновить
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {breakdown && (breakdown.by_route.length > 0 || breakdown.by_day.length > 0) && (
        <Grid container spacing={3} sx={{ mb: 3 }}>
          <Grid size={{ xs: 12, md: 6 }}>
            <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
              <Typography variant="subtitle1" sx={{ mb: 1 }}>
                Сводка по маршрутам (7 дней)
              </Typography>
              {breakdown.by_route.length === 0 ? (
                <Typography color="text.secondary">Ошибок за период не было.</Typography>
              ) : (
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Маршрут</TableCell>
                      <TableCell>Код</TableCell>
                      <TableCell align="right">Кол-во</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {breakdown.by_route.map((r) => (
                      <TableRow key={`${r.route}-${r.status}`} hover>
                        <TableCell>
                          <code>{r.route}</code>
                        </TableCell>
                        <TableCell>
                          <Chip label={r.status} size="small" color={statusColor(r.status)} />
                        </TableCell>
                        <TableCell align="right">{nf.format(r.count)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </Paper>
          </Grid>
          <Grid size={{ xs: 12, md: 6 }}>
            <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
              <Typography variant="subtitle1" sx={{ mb: 1 }}>
                Ошибки по дням (7 дней)
              </Typography>
              {breakdown.by_day.length === 0 ? (
                <Typography color="text.secondary">Ошибок за период не было.</Typography>
              ) : (
                <ResponsiveContainer width="100%" height={220}>
                  <LineChart data={breakdown.by_day} margin={{ top: 4, right: 16, bottom: 4, left: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E0E8E0" />
                    <XAxis dataKey="day" tick={{ fontSize: 12 }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                    <ReTooltip />
                    <Line
                      type="monotone"
                      dataKey="count"
                      name="Ошибки"
                      stroke="#C62828"
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </Paper>
          </Grid>
        </Grid>
      )}

      {events.length === 0 && !loading ? (
        <Paper variant="outlined" sx={{ p: 5, textAlign: 'center' }}>
          <Typography variant="h6">Ошибок нет 🎉</Typography>
          <Typography color="text.secondary">За хранимый период серверных ошибок не было.</Typography>
        </Paper>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Время</TableCell>
                <TableCell>Код</TableCell>
                <TableCell>Метод</TableCell>
                <TableCell>Маршрут</TableCell>
                <TableCell>Сообщение</TableCell>
                <TableCell>Источник</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {events.map((e) => (
                <TableRow key={e.id} hover>
                  <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(e.created_at)}</TableCell>
                  <TableCell>
                    <Chip label={e.status} size="small" color={statusColor(e.status)} />
                  </TableCell>
                  <TableCell>{e.method}</TableCell>
                  <TableCell>
                    <code>{e.route}</code>
                  </TableCell>
                  <TableCell sx={{ maxWidth: 360 }}>
                    <Tooltip title={e.request_id ? `request_id: ${e.request_id}` : ''}>
                      <Typography variant="body2" noWrap>
                        {e.message || '—'}
                      </Typography>
                    </Tooltip>
                  </TableCell>
                  <TableCell>{e.source}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
        {loading ? (
          <CircularProgress size={28} />
        ) : (
          hasMore && (
            <Button variant="outlined" onClick={() => void load(offset + PAGE, true)}>
              Показать ещё
            </Button>
          )
        )}
      </Box>
    </Box>
  );
}
