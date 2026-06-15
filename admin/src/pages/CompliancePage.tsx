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
import DownloadIcon from '@mui/icons-material/Download';
import { api, humanError } from '../api/client';
import { StatCard } from '../components/StatCard';
import type { ComplianceOverview, DeletionEvent, StalePurge } from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');
const mb = (b: number) => `${(b / 1048576).toFixed(1)} МБ`;

const DEL_LABELS: Record<string, string> = {
  account_deleted: 'Аккаунт удалён',
  account_media_purged: 'Медиа удалены',
};

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

function age(hours: number): string {
  return hours >= 48 ? `${Math.round(hours / 24)} дн` : `${Math.round(hours)} ч`;
}

// CSV-выгрузка журнала удалений (доказательство erasure). BOM для Excel.
function exportCsv(events: DeletionEvent[]) {
  const header = 'Событие;Пользователь;Время\n';
  const rows = events
    .map((e) => `${DEL_LABELS[e.action] ?? e.action};${e.user ?? ''};${e.created_at}`)
    .join('\n');
  const blob = new Blob(['﻿' + header + rows], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'deletions.csv';
  a.click();
  URL.revokeObjectURL(url);
}

export function CompliancePage() {
  const [ov, setOv] = useState<ComplianceOverview | null>(null);
  const [deletions, setDeletions] = useState<DeletionEvent[]>([]);
  const [stale, setStale] = useState<StalePurge[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    void (async () => {
      try {
        const [o, d, s] = await Promise.all([
          api.get<ComplianceOverview>('/compliance/overview'),
          api.get<{ events: DeletionEvent[] }>('/compliance/deletions?limit=200'),
          api.get<{ items: StalePurge[] }>('/compliance/stale-purges'),
        ]);
        setOv(o);
        setDeletions(d.events ?? []);
        setStale(s.items ?? []);
      } catch (err) {
        setError(humanError(err));
      }
    })();
  }, []);

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (!ov) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const cu = ov.cleanup;
  const cleanupValue = !cu.recorded ? '—' : cu.stale ? 'Задержка' : 'OK';
  const cleanupHint = !cu.recorded
    ? 'ещё не было прогона (появится после следующего часа)'
    : `${formatTime(cu.last_run_at ?? '')} · удалено ${nf.format(cu.users_purged)}/GC ${nf.format(cu.uploads_gc)}`;

  return (
    <Box>
      <Grid container spacing={2}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Очистка (cron)"
            value={cleanupValue}
            hint={cleanupHint}
            valueColor={cu.recorded && cu.stale ? 'error.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Зависший purge"
            value={nf.format(stale.length)}
            hint="удалены, но медиа не стёрты >72ч"
            valueColor={stale.length > 0 ? 'error.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Осиротевшие загрузки"
            value={nf.format(ov.uploads.stale_total)}
            hint={`${mb(ov.uploads.stale_bytes)} · всего непривязанных ${nf.format(ov.uploads.unused_total)}`}
            valueColor={ov.uploads.stale_total > 0 ? 'warning.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Остаток удалённых"
            value={nf.format(ov.usage_residue.rows)}
            hint={
              ov.usage_residue.has_residue
                ? `строк usage_log · ${nf.format(ov.usage_residue.users)} юзеров · старейшая ${age(ov.usage_residue.oldest_hours)}`
                : 'нет данных удалённых юзеров'
            }
          />
        </Grid>
      </Grid>

      {cu.recorded && cu.stale && (
        <Alert severity="error" sx={{ mt: 2 }}>
          Cron очистки не отрабатывал больше 2 часов (последний прогон {formatTime(cu.last_run_at ?? '')}).
          Проверьте systemd-таймер на VM — иначе backlog удаления растёт молча.
        </Alert>
      )}

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Typography variant="h6" sx={{ mb: 1 }}>
          Окна хранения служебных таблиц
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Строки, просрочившие срок хранения (152-ФЗ — минимизация). GC служебных таблиц — отдельная
          задача; здесь только замер.
        </Typography>
        <Table size="small" sx={{ mt: 1, maxWidth: 480 }}>
          <TableBody>
            <TableRow>
              <TableCell>Просроченные OTP-коды</TableCell>
              <TableCell align="right">{nf.format(ov.retention.expired_otp)}</TableCell>
            </TableRow>
            <TableRow>
              <TableCell>Просроченные OAuth-state</TableCell>
              <TableCell align="right">{nf.format(ov.retention.expired_oauth)}</TableCell>
            </TableRow>
            <TableRow>
              <TableCell>Отозванные refresh-токены &gt;30д</TableCell>
              <TableCell align="right">{nf.format(ov.retention.old_revoked)}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Paper>

      {stale.length > 0 && (
        <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Зависшая очистка медиа
          </Typography>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Пользователь</TableCell>
                <TableCell>Удалён</TableCell>
                <TableCell align="right">Висит</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {stale.map((p) => (
                <TableRow key={p.user} hover>
                  <TableCell>
                    <code>{p.user}</code>
                  </TableCell>
                  <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(p.deleted_at)}</TableCell>
                  <TableCell align="right">{age(p.pending_hours)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
          <Typography variant="h6">Журнал удалений (доказательство erasure)</Typography>
          <Button
            startIcon={<DownloadIcon />}
            onClick={() => exportCsv(deletions)}
            disabled={deletions.length === 0}
          >
            Экспорт CSV
          </Button>
        </Stack>
        {deletions.length === 0 ? (
          <Typography color="text.secondary">Событий удаления за хранимый период нет.</Typography>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Время</TableCell>
                <TableCell>Событие</TableCell>
                <TableCell>Пользователь</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {deletions.map((e, i) => (
                <TableRow key={`${e.created_at}-${i}`} hover>
                  <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(e.created_at)}</TableCell>
                  <TableCell>
                    <Chip
                      label={DEL_LABELS[e.action] ?? e.action}
                      size="small"
                      color={e.action === 'account_deleted' ? 'default' : 'success'}
                    />
                  </TableCell>
                  <TableCell>{e.user ? <code>{e.user}</code> : '—'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>
    </Box>
  );
}
