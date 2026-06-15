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
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import FiberManualRecordIcon from '@mui/icons-material/FiberManualRecord';
import { api, humanError } from '../api/client';
import type { DepStatus, FailCodeStat, WorkerHealth } from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');

// Русские подписи кодов отказа turn'а.
const FAIL_LABELS: Record<string, string> = {
  upstream_error: 'Ошибка апстрима (worker / Claude)',
  tool_loop_exhausted: 'Зацикливание tool-use',
  timeout: 'Таймаут',
  unknown: 'Без кода (старые записи)',
};

type Light = 'up' | 'down' | 'off';

function StatusDot({ state }: { state: Light }) {
  const color = state === 'up' ? 'success.main' : state === 'down' ? 'error.main' : 'text.disabled';
  return <FiberManualRecordIcon sx={{ color, fontSize: 16, verticalAlign: 'middle', mr: 1 }} />;
}

function statusText(state: Light): string {
  return state === 'up' ? 'в норме' : state === 'down' ? 'недоступен' : 'не настроено';
}

export function ReliabilityPage() {
  const [worker, setWorker] = useState<WorkerHealth | null>(null);
  const [deps, setDeps] = useState<DepStatus[]>([]);
  const [failCodes, setFailCodes] = useState<FailCodeStat[]>([]);
  const [failError, setFailError] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    setError('');
    setFailError('');
    try {
      const [wh, dp] = await Promise.all([
        api.get<WorkerHealth>('/health/worker'),
        api.get<{ deps: DepStatus[] }>('/health/deps'),
      ]);
      setWorker(wh);
      setDeps(dp.deps ?? []);
    } catch (err) {
      setError(humanError(err));
    } finally {
      setLoading(false);
    }
    // Загружается отдельно: если миграция 0016 (fail_code) ещё не применена,
    // этот виджет не должен ломать страницу со светофорами.
    try {
      const fc = await api.get<{ codes: FailCodeStat[] }>('/stats/fail-codes?days=14');
      setFailCodes(fc.codes ?? []);
    } catch (err) {
      setFailError(humanError(err));
    }
  };

  useEffect(() => {
    void load();
  }, []);

  if (loading && !worker) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const workerLight: Light = !worker?.configured ? 'off' : worker.online ? 'up' : 'down';
  const failTotal = failCodes.reduce((s, c) => s + c.count, 0);

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          Активные проверки доступности. Результаты кэшируются ~15 секунд.
        </Typography>
        <Button startIcon={<RefreshIcon />} onClick={() => void load()} disabled={loading}>
          Обновить
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 5 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              LLM-worker (Frankfurt)
            </Typography>
            {worker && (
              <Stack spacing={0.5}>
                <Typography variant="body1">
                  <StatusDot state={workerLight} />
                  {statusText(workerLight)}
                  {worker.configured && worker.online ? ` · ${nf.format(worker.latency_ms)} мс` : ''}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Модель: <code>{worker.model || '—'}</code>
                </Typography>
                {!worker.configured && (
                  <Typography variant="caption" color="text.secondary">
                    Проверка доступна только с реальным worker-клиентом (mTLS). В dev/mock — выключена.
                  </Typography>
                )}
              </Stack>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 7 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Зависимости
            </Typography>
            {deps.length === 0 ? (
              <Typography color="text.secondary">Проверки зависимостей не настроены.</Typography>
            ) : (
              <Table size="small">
                <TableBody>
                  {deps.map((d) => {
                    const light: Light = !d.configured ? 'off' : d.up ? 'up' : 'down';
                    return (
                      <TableRow key={d.name}>
                        <TableCell sx={{ width: 24 }}>
                          <StatusDot state={light} />
                        </TableCell>
                        <TableCell>{d.name}</TableCell>
                        <TableCell>{statusText(light)}</TableCell>
                        <TableCell align="right" sx={{ color: 'text.secondary' }}>
                          {d.configured && d.up ? `${nf.format(d.latency_ms)} мс` : (d.detail ?? '')}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 1 }}>
          Сбои ответов по коду (14 дней)
        </Typography>
        {failError ? (
          <Alert severity="info">
            Разбивка недоступна. Вероятно, не применена миграция 0016 (messages.fail_code) на этой
            среде.
          </Alert>
        ) : failCodes.length === 0 ? (
          <Typography color="text.secondary">Сбоев ответов за период не было 🎉</Typography>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Причина</TableCell>
                <TableCell align="right">Кол-во</TableCell>
                <TableCell align="right">Доля</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {failCodes.map((c) => (
                <TableRow key={c.code} hover>
                  <TableCell>
                    <Chip label={c.code} size="small" sx={{ mr: 1 }} />
                    {FAIL_LABELS[c.code] ?? ''}
                  </TableCell>
                  <TableCell align="right">{nf.format(c.count)}</TableCell>
                  <TableCell align="right">
                    {failTotal > 0 ? `${Math.round((c.count / failTotal) * 100)}%` : '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>
    </Box>
  );
}
