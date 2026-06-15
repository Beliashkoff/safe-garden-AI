import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
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
import { api, humanError } from '../api/client';
import { StatCard } from '../components/StatCard';
import type { OtpStats, SecurityEvent, Session, SuspiciousIP } from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');
const EVENTS_PAGE = 50;

const SEC_LABELS: Record<string, string> = {
  refresh_reuse_detected: 'Переиспользование refresh-токена (угон сессии)',
  account_deleted: 'Аккаунт удалён',
  account_media_purged: 'Медиа удалены (очистка)',
};

const IP_SOURCE_LABELS: Record<string, string> = {
  admin_login_failed: 'Брутфорс админки',
  refresh_reuse: 'Угон refresh-токена',
};

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

export function SecurityPage() {
  const [otp, setOtp] = useState<OtpStats | null>(null);
  const [ips, setIps] = useState<SuspiciousIP[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [events, setEvents] = useState<SecurityEvent[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [eventsLoading, setEventsLoading] = useState(false);

  const loadEvents = async (nextOffset: number, append: boolean) => {
    setEventsLoading(true);
    try {
      const res = await api.get<{ events: SecurityEvent[] }>(
        `/security/events?limit=${EVENTS_PAGE}&offset=${nextOffset}`,
      );
      const list = res.events ?? [];
      setHasMore(list.length === EVENTS_PAGE);
      setOffset(nextOffset);
      setEvents((prev) => (append ? [...prev, ...list] : list));
    } catch (err) {
      setError(humanError(err));
    } finally {
      setEventsLoading(false);
    }
  };

  useEffect(() => {
    void (async () => {
      try {
        const [o, ip, ss] = await Promise.all([
          api.get<OtpStats>('/security/otp?days=7'),
          api.get<{ ips: SuspiciousIP[] }>('/security/suspicious-ips?days=7'),
          api.get<{ sessions: Session[] }>('/sessions'),
        ]);
        setOtp(o);
        setIps(ip.ips ?? []);
        setSessions(ss.sessions ?? []);
        await loadEvents(0, false);
      } catch (err) {
        setError(humanError(err));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (error && !otp) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (loading || !otp) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const distinctIPs = new Set(sessions.map((s) => s.ip).filter(Boolean)).size;
  const sessionAnomaly = distinctIPs > 1;
  const deliveryPct = otp.issued > 0 ? Math.round(otp.delivery_rate * 100) : null;

  return (
    <Box>
      <Grid container spacing={2}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Доставка OTP (7д)"
            value={deliveryPct === null ? '—' : `${deliveryPct}%`}
            hint={`${nf.format(otp.used)} из ${nf.format(otp.issued)} использовано`}
            valueColor={deliveryPct !== null && deliveryPct < 80 ? 'warning.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Отбитые OTP (7д)"
            value={nf.format(otp.exhausted)}
            hint={`исчерпан лимит попыток · просрочено ${nf.format(otp.expired_unused)}`}
            valueColor={otp.exhausted > 0 ? 'warning.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Активных админ-сессий"
            value={nf.format(sessions.length)}
            hint={sessionAnomaly ? `с ${distinctIPs} разных IP — проверьте!` : `${distinctIPs} IP`}
            valueColor={sessionAnomaly ? 'error.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Подозрительные IP (7д)"
            value={nf.format(ips.length)}
            hint="по неуспешным входам"
            valueColor={ips.length > 0 ? 'warning.main' : undefined}
          />
        </Grid>
      </Grid>

      {sessionAnomaly && (
        <Alert severity="error" sx={{ mt: 2 }}>
          Активны сессии админки с {distinctIPs} разных IP. Оператор один — это возможная
          компрометация. Проверьте и завершите лишние сессии в «Настройках».
        </Alert>
      )}

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Подозрительные IP (7 дней)
            </Typography>
            {ips.length === 0 ? (
              <Typography color="text.secondary">Неуспешных входов за период нет.</Typography>
            ) : (
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>IP</TableCell>
                    <TableCell>Источник</TableCell>
                    <TableCell align="right">Попыток</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {ips.map((i) => (
                    <TableRow key={`${i.source}-${i.ip}`} hover>
                      <TableCell>
                        <code>{i.ip}</code>
                      </TableCell>
                      <TableCell>{IP_SOURCE_LABELS[i.source] ?? i.source}</TableCell>
                      <TableCell align="right">{nf.format(i.count)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Запросчики OTP (≥3 кода, 7 дней)
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Email замаскирован. Всплеск на один адрес/домен — флуд почты или перебор.
            </Typography>
            {otp.top_requesters.length === 0 ? (
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                Аномальных запросчиков нет.
              </Typography>
            ) : (
              <Table size="small" sx={{ mt: 1 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Email</TableCell>
                    <TableCell align="right">Кодов</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {otp.top_requesters.map((rq) => (
                    <TableRow key={rq.email} hover>
                      <TableCell>{rq.email}</TableCell>
                      <TableCell align="right">{nf.format(rq.codes)}</TableCell>
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
          Лента событий безопасности
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Угон сессий, удаления аккаунтов и очистка медиа. Идентификатор пользователя замаскирован.
        </Typography>
        {events.length === 0 && !eventsLoading ? (
          <Typography color="text.secondary" sx={{ mt: 2 }}>
            Событий безопасности за хранимый период нет.
          </Typography>
        ) : (
          <Table size="small" sx={{ mt: 1 }}>
            <TableHead>
              <TableRow>
                <TableCell>Время</TableCell>
                <TableCell>Событие</TableCell>
                <TableCell>Пользователь</TableCell>
                <TableCell>IP</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {events.map((e, i) => (
                <TableRow key={`${e.created_at}-${i}`} hover>
                  <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(e.created_at)}</TableCell>
                  <TableCell>
                    {e.action === 'refresh_reuse_detected' ? (
                      <Chip label={SEC_LABELS[e.action]} size="small" color="error" />
                    ) : (
                      (SEC_LABELS[e.action] ?? e.action)
                    )}
                  </TableCell>
                  <TableCell>{e.user ? <code>{e.user}</code> : '—'}</TableCell>
                  <TableCell>{e.ip || '—'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
          {eventsLoading ? (
            <CircularProgress size={28} />
          ) : (
            hasMore && (
              <Button variant="outlined" onClick={() => void loadEvents(offset + EVENTS_PAGE, true)}>
                Показать ещё
              </Button>
            )
          )}
        </Box>
      </Paper>
    </Box>
  );
}
