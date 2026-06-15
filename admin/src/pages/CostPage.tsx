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
  Typography,
} from '@mui/material';
import { api, humanError } from '../api/client';
import { StatCard } from '../components/StatCard';
import type { CacheStats, CostByKind, ModelCost, UnitEconomics } from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');
const money = (n: number) => `$${n.toFixed(2)}`;
const money4 = (n: number) => `$${n.toFixed(4)}`;

// Веса стоимости токена у Opus: вход $5 / выход $25 за 1M → выход в 5× дороже.
// Соотношение одинаково для 4.7 и 4.8, поэтому доля output в косте устойчива к
// точной цене и считается через коэффициент 5, а не абсолютные тарифы.
const OUTPUT_COST_WEIGHT = 5;

export function CostPage() {
  const [unit, setUnit] = useState<UnitEconomics | null>(null);
  const [cache, setCache] = useState<CacheStats | null>(null);
  const [models, setModels] = useState<ModelCost[]>([]);
  const [kind, setKind] = useState<CostByKind | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    void (async () => {
      try {
        const [u, c, m, k] = await Promise.all([
          api.get<UnitEconomics>('/stats/unit-economics?days=30'),
          api.get<CacheStats>('/stats/cache?days=30'),
          api.get<{ models: ModelCost[] }>('/stats/cost-by-model?days=30'),
          api.get<CostByKind>('/stats/cost-by-kind?days=30'),
        ]);
        setUnit(u);
        setCache(c);
        setModels(m.models);
        setKind(k);
      } catch (err) {
        setError(humanError(err));
      }
    })();
  }, []);

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (!unit || !cache || !kind) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const totalTokens = unit.tokens_in + unit.tokens_out;
  const ratio = unit.tokens_out > 0 ? unit.tokens_in / unit.tokens_out : 0;
  const outShare =
    totalTokens > 0
      ? Math.round(
          ((unit.tokens_out * OUTPUT_COST_WEIGHT) /
            (unit.tokens_in + unit.tokens_out * OUTPUT_COST_WEIGHT)) *
            100,
        )
      : 0;

  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Юнит-экономика и FinOps за 30 дней. Стоимость — оценочная, из счётчиков токенов.
      </Typography>

      <Grid container spacing={2}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Стоимость сообщения"
            value={money4(unit.cost_per_message)}
            hint={`${nf.format(unit.messages)} сообщений · всего ${money(unit.cost_usd)}`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Стоимость на юзера"
            value={money(unit.cost_per_user)}
            hint={`${nf.format(unit.users)} активных за 30 дней`}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Кэш hit-rate"
            value={cache.has_data ? `${Math.round(cache.hit_rate * 100)}%` : '—'}
            hint={
              cache.has_data
                ? `экономия ${money(cache.savings_usd)} за 30 дней`
                : 'данные появятся после деплоя 0017'
            }
            valueColor={cache.has_data && cache.hit_rate > 0 ? 'success.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Токены вход/выход"
            value={ratio > 0 ? `${ratio.toFixed(1)}:1` : '—'}
            hint={`выход ≈ ${outShare}% косты (выход в 5× дороже)`}
            valueColor={outShare > 60 ? 'warning.main' : undefined}
          />
        </Grid>
      </Grid>

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 7 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Стоимость по версиям модели (30 дней)
            </Typography>
            {models.length === 0 ? (
              <Typography color="text.secondary">Пока нет данных за период.</Typography>
            ) : (
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Модель</TableCell>
                    <TableCell align="right">Расход</TableCell>
                    <TableCell align="right">Запросы</TableCell>
                    <TableCell align="right">Токены (вх/вых)</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {models.map((m) => (
                    <TableRow key={m.model} hover>
                      <TableCell>
                        <code>{m.model}</code>
                      </TableCell>
                      <TableCell align="right">{money(m.cost_usd)}</TableCell>
                      <TableCell align="right">{nf.format(m.requests)}</TableCell>
                      <TableCell align="right">
                        {nf.format(m.tokens_in)} / {nf.format(m.tokens_out)}
                      </TableCell>
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
              Косты по типу операции (30 дней)
            </Typography>
            <Table size="small">
              <TableBody>
                <TableRow>
                  <TableCell>Claude (диагностика)</TableCell>
                  <TableCell align="right">
                    {money(kind.claude_cost)} · {nf.format(kind.claude_calls)} запр.
                  </TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>Клики по карточкам</TableCell>
                  <TableCell align="right">{nf.format(kind.taps)} · $0</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>Транскрипция (SpeechKit, ₽)</TableCell>
                  <TableCell align="right">
                    {nf.format(kind.transcriptions)} · {nf.format(kind.transcribe_sec)} сек
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
              SpeechKit оплачивается в рублях по длительности — здесь объём, не доллары.
            </Typography>
            {cache.has_data && (
              <Box sx={{ mt: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                  Prompt-кэш (токены)
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Без кэша: {nf.format(cache.input_uncached)} · запись:{' '}
                  {nf.format(cache.cache_write)} · чтение из кэша: {nf.format(cache.cache_read)}
                </Typography>
              </Box>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}
