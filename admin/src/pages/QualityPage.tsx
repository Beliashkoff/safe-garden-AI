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
import { api, humanError } from '../api/client';
import { StatCard } from '../components/StatCard';
import type {
  CTROverview,
  DownvotedMessage,
  Followup,
  LengthVsVerdict,
  NegativeConversation,
} from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');
const DOWN_PAGE = 20;

function pct(part: number, whole: number): string {
  if (whole <= 0) return '—';
  return `${Math.round((part / whole) * 100)}%`;
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

// Многострочный обрезанный текст (вопрос/ответ в ленте дизлайков).
function clampSx(lines: number) {
  return {
    display: '-webkit-box',
    WebkitLineClamp: lines,
    WebkitBoxOrient: 'vertical' as const,
    overflow: 'hidden',
  };
}

export function QualityPage() {
  const [ctr, setCtr] = useState<CTROverview | null>(null);
  const [length, setLength] = useState<LengthVsVerdict | null>(null);
  const [followup, setFollowup] = useState<Followup | null>(null);
  const [negative, setNegative] = useState<NegativeConversation[]>([]);
  const [downvoted, setDownvoted] = useState<DownvotedMessage[]>([]);
  const [downOffset, setDownOffset] = useState(0);
  const [downMore, setDownMore] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [downLoading, setDownLoading] = useState(false);

  const loadDownvoted = async (offset: number, append: boolean) => {
    setDownLoading(true);
    try {
      const res = await api.get<{ items: DownvotedMessage[] }>(
        `/stats/downvoted?limit=${DOWN_PAGE}&offset=${offset}`,
      );
      const list = res.items ?? [];
      setDownMore(list.length === DOWN_PAGE);
      setDownOffset(offset);
      setDownvoted((prev) => (append ? [...prev, ...list] : list));
    } catch (err) {
      setError(humanError(err));
    } finally {
      setDownLoading(false);
    }
  };

  useEffect(() => {
    void (async () => {
      try {
        const [c, l, f, neg] = await Promise.all([
          api.get<CTROverview>('/stats/ctr?days=30'),
          api.get<LengthVsVerdict>('/stats/length-vs-verdict?days=30'),
          api.get<Followup>('/stats/followup?days=30'),
          api.get<{ items: NegativeConversation[] }>('/stats/negative-conversations?days=30&min=2'),
        ]);
        setCtr(c);
        setLength(l);
        setFollowup(f);
        setNegative(neg.items);
        await loadDownvoted(0, false);
      } catch (err) {
        setError(humanError(err));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (error && !ctr) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (loading || !ctr || !length || !followup) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const lengthRows = [
    { label: '👍 Лайк', avg: length.avg_up, n: length.n_up },
    { label: '👎 Дизлайк', avg: length.avg_down, n: length.n_down },
    { label: 'Без оценки', avg: length.avg_none, n: length.n_none },
  ];

  return (
    <Box>
      <Grid container spacing={2}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="CTR рекомендаций"
            value={pct(ctr.taps, ctr.impressions)}
            hint={`${nf.format(ctr.taps)} кликов из ${nf.format(ctr.impressions)} показов`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Карточек показано"
            value={nf.format(ctr.cards)}
            hint="за 30 дней"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Переспросы"
            value={pct(followup.followups, followup.answers)}
            hint={`${nf.format(followup.followups)} из ${nf.format(followup.answers)} ответов · ≤5 мин`}
            valueColor={
              followup.answers > 0 && followup.followups / followup.answers > 0.3
                ? 'warning.main'
                : undefined
            }
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Чатов с негативом"
            value={nf.format(negative.length)}
            hint="≥2 дизлайков за 30 дней"
            valueColor={negative.length > 0 ? 'warning.main' : undefined}
          />
        </Grid>
      </Grid>

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 7 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              CTR карточек по товарам (30 дней)
            </Typography>
            {ctr.per_slug.length === 0 ? (
              <Typography color="text.secondary">
                Пока нет показов карточек. Они появятся, когда ИИ начнёт рекомендовать удобрения.
              </Typography>
            ) : (
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Товар (slug)</TableCell>
                    <TableCell align="right">Показы</TableCell>
                    <TableCell align="right">Клики</TableCell>
                    <TableCell align="right">CTR</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {ctr.per_slug.map((s) => (
                    <TableRow key={s.slug} hover>
                      <TableCell>{s.slug}</TableCell>
                      <TableCell align="right">{nf.format(s.impressions)}</TableCell>
                      <TableCell align="right">{nf.format(s.taps)}</TableCell>
                      <TableCell align="right">{pct(s.taps, s.impressions)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 5 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Длина ответа vs оценка
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Средняя длина ответа (токенов на выход) по вердикту. Систематический перекос — сигнал к
              правке промпта.
            </Typography>
            <Table size="small" sx={{ mt: 1 }}>
              <TableHead>
                <TableRow>
                  <TableCell>Оценка</TableCell>
                  <TableCell align="right">Ср. токенов</TableCell>
                  <TableCell align="right">Ответов</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {lengthRows.map((row) => (
                  <TableRow key={row.label}>
                    <TableCell>{row.label}</TableCell>
                    <TableCell align="right">{Math.round(row.avg)}</TableCell>
                    <TableCell align="right">{nf.format(row.n)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Paper>
        </Grid>
      </Grid>

      {negative.length > 0 && (
        <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Чаты с накопленным негативом (30 дней)
          </Typography>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Чат</TableCell>
                <TableCell align="right">Дизлайков</TableCell>
                <TableCell>Последний</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {negative.map((c) => (
                <TableRow key={c.conversation} hover>
                  <TableCell>
                    <code>{c.conversation}</code>
                  </TableCell>
                  <TableCell align="right">{nf.format(c.downs)}</TableCell>
                  <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(c.last_down)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}

      <Paper variant="outlined" sx={{ mt: 3, p: 2 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
          <Typography variant="h6">Ответы с дизлайком (на разбор)</Typography>
        </Stack>
        <Typography variant="caption" color="text.secondary">
          Содержимое показывается только для ручного разбора качества и нигде не сохраняется в логи.
          Идентификатор чата замаскирован.
        </Typography>

        {downvoted.length === 0 && !downLoading ? (
          <Typography color="text.secondary" sx={{ mt: 2 }}>
            Дизлайков пока нет. Здесь появятся ответы, которые пользователи оценили на 👎.
          </Typography>
        ) : (
          <Stack spacing={1.5} sx={{ mt: 2 }}>
            {downvoted.map((m, i) => (
              <Paper key={`${m.conversation}-${i}`} variant="outlined" sx={{ p: 1.5, bgcolor: 'action.hover' }}>
                <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
                  <code>{m.conversation}</code>
                  {m.had_card && <Chip label="была карточка" size="small" color="success" />}
                  <Typography variant="caption" color="text.secondary" sx={{ ml: 'auto' }}>
                    {formatTime(m.created_at)}
                  </Typography>
                </Stack>
                {m.question && (
                  <Typography variant="body2" sx={{ ...clampSx(3) }}>
                    <b>Вопрос:</b> {m.question}
                  </Typography>
                )}
                {m.answer && (
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, ...clampSx(6) }}>
                    <b>Ответ:</b> {m.answer}
                  </Typography>
                )}
              </Paper>
            ))}
          </Stack>
        )}

        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
          {downLoading ? (
            <CircularProgress size={28} />
          ) : (
            downMore && (
              <Button variant="outlined" onClick={() => void loadDownvoted(downOffset + DOWN_PAGE, true)}>
                Показать ещё
              </Button>
            )
          )}
        </Box>
      </Paper>
    </Box>
  );
}
