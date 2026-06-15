import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
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
import { problemLabel } from '../labels';
import type { CatalogProduct, ProblemStat } from '../api/types';

const nf = new Intl.NumberFormat('ru-RU');

function pct(part: number, whole: number): string {
  if (whole <= 0) return '—';
  return `${Math.round((part / whole) * 100)}%`;
}

export function AssortmentPage() {
  const [problems, setProblems] = useState<ProblemStat[]>([]);
  const [products, setProducts] = useState<CatalogProduct[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const [p, c] = await Promise.all([
          api.get<{ problems: ProblemStat[] }>('/stats/problems?days=30'),
          api.get<{ products: CatalogProduct[] }>('/stats/catalog-performance?days=30'),
        ]);
        setProblems(p.problems ?? []);
        setProducts(c.products ?? []);
      } catch (err) {
        setError(humanError(err));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  const totalDiag = problems.reduce((s, p) => s + p.total, 0);
  const totalMiss = problems.reduce((s, p) => s + p.misses, 0);
  const dead = products.filter((p) => p.impressions === 0 && p.taps === 0).length;

  return (
    <Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Что диагностирует ИИ и насколько каталог это закрывает. Данные по проблемам копятся с момента
        деплоя (раньше problem-ключ нигде не сохранялся).
      </Typography>

      <Grid container spacing={2}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Диагнозов (30д)"
            value={nf.format(totalDiag)}
            hint="вызовов recommend_fertilizer"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Каталог не закрыл"
            value={pct(totalMiss, totalDiag)}
            hint={`${nf.format(totalMiss)} запросов без товара`}
            valueColor={totalDiag > 0 && totalMiss / Math.max(totalDiag, 1) > 0.3 ? 'warning.main' : undefined}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Активных товаров"
            value={nf.format(products.length)}
            hint="в каталоге (active)"
          />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            title="Мёртвые товары"
            value={nf.format(dead)}
            hint="без показов и кликов за 30д"
            valueColor={dead > 0 ? 'warning.main' : undefined}
          />
        </Grid>
      </Grid>

      <Grid container spacing={3} sx={{ mt: 0 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Проблемы и пробелы каталога (30д)
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Высокий miss-rate = частая болезнь без удобрения в каталоге → дыра ассортимента.
            </Typography>
            {problems.length === 0 ? (
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                Пока нет диагнозов за период. Появятся после первых рекомендаций.
              </Typography>
            ) : (
              <Table size="small" sx={{ mt: 1 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Проблема</TableCell>
                    <TableCell align="right">Запросов</TableCell>
                    <TableCell align="right">Miss-rate</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {problems.map((p) => {
                    const missRate = p.total > 0 ? p.misses / p.total : 0;
                    return (
                      <TableRow key={p.problem} hover>
                        <TableCell>{problemLabel(p.problem)}</TableCell>
                        <TableCell align="right">{nf.format(p.total)}</TableCell>
                        <TableCell
                          align="right"
                          sx={{ color: missRate > 0.5 ? 'error.main' : missRate > 0.2 ? 'warning.main' : undefined }}
                        >
                          {pct(p.misses, p.total)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Эффективность товаров (30д)
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Все активные товары. «Мёртвый» = ИИ ни разу не показал или не было кликов.
            </Typography>
            {products.length === 0 ? (
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                В каталоге нет активных товаров.
              </Typography>
            ) : (
              <Table size="small" sx={{ mt: 1 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Товар</TableCell>
                    <TableCell align="right">Показы</TableCell>
                    <TableCell align="right">Клики</TableCell>
                    <TableCell align="right">CTR</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {products.map((p) => {
                    const isDead = p.impressions === 0 && p.taps === 0;
                    return (
                      <TableRow key={p.slug} hover>
                        <TableCell>
                          {p.name}
                          {isDead && <Chip label="мёртвый" size="small" color="warning" sx={{ ml: 1 }} />}
                        </TableCell>
                        <TableCell align="right">{nf.format(p.impressions)}</TableCell>
                        <TableCell align="right">{nf.format(p.taps)}</TableCell>
                        <TableCell align="right">{pct(p.taps, p.impressions)}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}
