import { useState } from 'react';
import type { FormEvent } from 'react';
import {
  Alert,
  Box,
  Button,
  Link,
  Paper,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { api, humanError } from '../api/client';
import type { Admin } from '../api/types';
import { useAuth } from '../auth/AuthContext';

type Mode = 'login' | 'setup-request' | 'setup-complete' | 'reset-request' | 'reset-complete';

// Один экран на все пред-авторизационные сценарии: вход, первичная настройка
// (когда аккаунта ещё нет) и восстановление пароля. Код всегда уходит только
// на привязанную почту — сервер другие адреса отклоняет.
export function AuthPage() {
  const { bootstrapped, setAdmin, refresh } = useAuth();
  const [mode, setMode] = useState<Mode>(bootstrapped === false ? 'setup-request' : 'login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [password2, setPassword2] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [busy, setBusy] = useState(false);

  const run = async (fn: () => Promise<void>) => {
    setError('');
    setInfo('');
    setBusy(true);
    try {
      await fn();
    } catch (err) {
      setError(humanError(err));
    } finally {
      setBusy(false);
    }
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    switch (mode) {
      case 'login':
        void run(async () => {
          const res = await api.post<{ admin: Admin }>('/auth/login', { email, password });
          setAdmin(res.admin);
        });
        break;
      case 'setup-request':
        void run(async () => {
          await api.post('/auth/setup/request', { email });
          setInfo('Код отправлен на почту. Проверьте входящие.');
          setMode('setup-complete');
        });
        break;
      case 'setup-complete':
        if (password !== password2) {
          setError('Пароли не совпадают.');
          return;
        }
        void run(async () => {
          const res = await api.post<{ admin: Admin }>('/auth/setup/complete', {
            email,
            code,
            password,
          });
          setAdmin(res.admin);
          await refresh();
        });
        break;
      case 'reset-request':
        void run(async () => {
          await api.post('/auth/reset/request', { email });
          setInfo('Код восстановления отправлен на почту.');
          setMode('reset-complete');
        });
        break;
      case 'reset-complete':
        if (password !== password2) {
          setError('Пароли не совпадают.');
          return;
        }
        void run(async () => {
          await api.post('/auth/reset/complete', { email, code, new_password: password });
          setInfo('Пароль обновлён. Войдите с новым паролем.');
          setPassword('');
          setPassword2('');
          setCode('');
          setMode('login');
        });
        break;
    }
  };

  const titles: Record<Mode, string> = {
    login: 'Вход в админ-панель',
    'setup-request': 'Первичная настройка',
    'setup-complete': 'Создание пароля',
    'reset-request': 'Восстановление пароля',
    'reset-complete': 'Новый пароль',
  };

  const needsCodeAndPassword = mode === 'setup-complete' || mode === 'reset-complete';

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        p: 2,
        background: 'linear-gradient(160deg, #E8F5E9 0%, #F4F6F4 60%)',
      }}
    >
      <Paper elevation={3} sx={{ p: 4, width: '100%', maxWidth: 420 }}>
        <Stack spacing={2.5} component="form" onSubmit={submit}>
          <Box textAlign="center">
            <Typography variant="h4" sx={{ mb: 0.5 }}>
              🌱
            </Typography>
            <Typography variant="h5" color="primary.main">
              Safe Garden AI
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {titles[mode]}
            </Typography>
          </Box>

          {mode === 'setup-request' && (
            <Alert severity="info">
              Аккаунт администратора ещё не создан. Укажите привязанную почту — мы отправим на неё
              код подтверждения.
            </Alert>
          )}
          {error && <Alert severity="error">{error}</Alert>}
          {info && <Alert severity="success">{info}</Alert>}

          <TextField
            label="Почта"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            fullWidth
            autoComplete="username"
          />

          {mode === 'login' && (
            <TextField
              label="Пароль"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              fullWidth
              autoComplete="current-password"
            />
          )}

          {needsCodeAndPassword && (
            <>
              <TextField
                label="Код из письма"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
                fullWidth
                slotProps={{ htmlInput: { inputMode: 'numeric', maxLength: 6 } }}
              />
              <TextField
                label="Новый пароль"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                fullWidth
                helperText="Минимум 12 символов, обязательно буквы и цифры"
                autoComplete="new-password"
              />
              <TextField
                label="Повторите пароль"
                type="password"
                value={password2}
                onChange={(e) => setPassword2(e.target.value)}
                required
                fullWidth
                autoComplete="new-password"
              />
            </>
          )}

          <Button type="submit" variant="contained" size="large" disabled={busy} fullWidth>
            {mode === 'login' && 'Войти'}
            {(mode === 'setup-request' || mode === 'reset-request') && 'Отправить код'}
            {needsCodeAndPassword && 'Сохранить'}
          </Button>

          <Box textAlign="center">
            {mode === 'login' && (
              <Link component="button" type="button" onClick={() => setMode('reset-request')}>
                Забыли пароль?
              </Link>
            )}
            {mode !== 'login' && bootstrapped !== false && (
              <Link component="button" type="button" onClick={() => setMode('login')}>
                Вернуться ко входу
              </Link>
            )}
          </Box>
        </Stack>
      </Paper>
    </Box>
  );
}
