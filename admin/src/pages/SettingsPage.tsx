import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  Grid,
  Paper,
  Snackbar,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import { api, humanError } from '../api/client';
import type { Session } from '../api/types';
import { useAuth } from '../auth/AuthContext';

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

export function SettingsPage() {
  const { admin } = useAuth();
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [next2, setNext2] = useState('');
  const [pwError, setPwError] = useState('');
  const [toast, setToast] = useState('');
  const [busy, setBusy] = useState(false);

  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [sessError, setSessError] = useState('');

  const loadSessions = async () => {
    try {
      const res = await api.get<{ sessions: Session[] }>('/sessions');
      setSessions(res.sessions);
    } catch (err) {
      setSessError(humanError(err));
    }
  };

  useEffect(() => {
    void loadSessions();
  }, []);

  const changePassword = async () => {
    setPwError('');
    if (next !== next2) {
      setPwError('Новые пароли не совпадают.');
      return;
    }
    if (next.length < 12) {
      setPwError('Пароль должен быть не короче 12 символов.');
      return;
    }
    setBusy(true);
    try {
      await api.post('/me/password', { current_password: current, new_password: next });
      setToast('Пароль изменён. Остальные сессии завершены.');
      setCurrent('');
      setNext('');
      setNext2('');
      await loadSessions();
    } catch (err) {
      setPwError(humanError(err));
    } finally {
      setBusy(false);
    }
  };

  const revokeSession = async (id: string) => {
    try {
      await api.del(`/sessions/${id}`);
      setToast('Сессия завершена.');
      await loadSessions();
    } catch (err) {
      setSessError(humanError(err));
    }
  };

  return (
    <Box>
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 5 }}>
          <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Аккаунт
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Почта: <strong>{admin?.email}</strong>
            </Typography>
            {admin?.last_login_at && (
              <Typography variant="body2" color="text.secondary">
                Последний вход: {formatTime(admin.last_login_at)}
              </Typography>
            )}

            <Divider sx={{ my: 2 }} />

            <Typography variant="h6" sx={{ mb: 2 }}>
              Смена пароля
            </Typography>
            {pwError && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {pwError}
              </Alert>
            )}
            <Stack spacing={2}>
              <TextField
                label="Текущий пароль"
                type="password"
                value={current}
                onChange={(e) => setCurrent(e.target.value)}
                fullWidth
                autoComplete="current-password"
              />
              <TextField
                label="Новый пароль"
                type="password"
                value={next}
                onChange={(e) => setNext(e.target.value)}
                fullWidth
                helperText="Минимум 12 символов, буквы и цифры"
                autoComplete="new-password"
              />
              <TextField
                label="Повторите новый пароль"
                type="password"
                value={next2}
                onChange={(e) => setNext2(e.target.value)}
                fullWidth
                autoComplete="new-password"
              />
              <Button
                variant="contained"
                onClick={() => void changePassword()}
                disabled={busy || !current || !next}
              >
                Изменить пароль
              </Button>
            </Stack>
          </Paper>
        </Grid>

        <Grid size={{ xs: 12, md: 7 }}>
          <Paper variant="outlined" sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Активные сессии
            </Typography>
            {sessError && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {sessError}
              </Alert>
            )}
            {sessions === null ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
                <CircularProgress size={28} />
              </Box>
            ) : (
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Устройство / IP</TableCell>
                      <TableCell>Активность</TableCell>
                      <TableCell align="right" />
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {sessions.map((s) => (
                      <TableRow key={s.id}>
                        <TableCell>
                          <Typography variant="body2" noWrap sx={{ maxWidth: 280 }}>
                            {s.user_agent || 'Неизвестное устройство'}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {s.ip || '—'}
                          </Typography>
                        </TableCell>
                        <TableCell>{formatTime(s.last_seen_at)}</TableCell>
                        <TableCell align="right">
                          {s.current ? (
                            <Chip label="текущая" size="small" color="primary" />
                          ) : (
                            <Button size="small" color="error" onClick={() => void revokeSession(s.id)}>
                              Завершить
                            </Button>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Snackbar
        open={toast !== ''}
        autoHideDuration={4000}
        onClose={() => setToast('')}
        message={toast}
      />
    </Box>
  );
}
