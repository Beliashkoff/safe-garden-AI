import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
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
import RefreshIcon from '@mui/icons-material/Refresh';
import { api, humanError } from '../api/client';
import type { ServerError } from '../api/types';

const PAGE = 50;

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

  useEffect(() => {
    void load(0, false);
  }, []);

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          Серверные ошибки (5xx) бэкенда. Содержимое сообщений пользователей сюда не попадает.
        </Typography>
        <Button startIcon={<RefreshIcon />} onClick={() => void load(0, false)} disabled={loading}>
          Обновить
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
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
