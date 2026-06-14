import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  InputAdornment,
  Paper,
  Snackbar,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import SearchIcon from '@mui/icons-material/Search';
import EditIcon from '@mui/icons-material/Edit';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import { api, humanError } from '../api/client';
import type { Product, ProductInput } from '../api/types';
import { problemLabel } from '../labels';
import { ProductDialog } from '../components/ProductDialog';

export function CatalogPage() {
  const [products, setProducts] = useState<Product[] | null>(null);
  const [problemKeys, setProblemKeys] = useState<string[]>([]);
  const [search, setSearch] = useState('');
  const [error, setError] = useState('');
  const [toast, setToast] = useState('');
  const [editing, setEditing] = useState<Product | 'new' | null>(null);
  const [deleting, setDeleting] = useState<Product | null>(null);
  const [busy, setBusy] = useState(false);

  const load = async () => {
    try {
      const [list, problems] = await Promise.all([
        api.get<{ products: Product[] }>('/fertilizers'),
        api.get<{ problems: string[] }>('/fertilizers/problems'),
      ]);
      setProducts(list.products);
      setProblemKeys(problems.problems);
    } catch (err) {
      setError(humanError(err));
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const filtered = useMemo(() => {
    if (!products) return [];
    const q = search.trim().toLowerCase();
    if (!q) return products;
    return products.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.slug.includes(q) ||
        p.category.toLowerCase().includes(q),
    );
  }, [products, search]);

  const save = async (input: ProductInput, id?: string) => {
    setBusy(true);
    try {
      if (id) {
        await api.put(`/fertilizers/${id}`, input);
        setToast('Товар обновлён. ИИ уже использует новые данные.');
      } else {
        await api.post('/fertilizers', input);
        setToast('Товар добавлен. Теперь ИИ может его рекомендовать.');
      }
      setEditing(null);
      await load();
    } catch (err) {
      throw new Error(humanError(err));
    } finally {
      setBusy(false);
    }
  };

  const toggleActive = async (p: Product) => {
    try {
      await api.put(`/fertilizers/${p.id}`, {
        slug: p.slug,
        name: p.name,
        short_desc: p.short_desc,
        long_desc: p.long_desc ?? '',
        image_url: p.image_url ?? '',
        deeplink_url: p.deeplink_url ?? '',
        category: p.category,
        problems: p.problems,
        plants: p.plants,
        priority: p.priority,
        active: !p.active,
        price_rub: p.price_rub ?? null,
      });
      setToast(p.active ? 'Товар скрыт из рекомендаций.' : 'Товар снова в рекомендациях.');
      await load();
    } catch (err) {
      setError(humanError(err));
    }
  };

  const remove = async () => {
    if (!deleting) return;
    setBusy(true);
    try {
      await api.del(`/fertilizers/${deleting.id}`);
      setToast('Товар удалён.');
      setDeleting(null);
      await load();
    } catch (err) {
      setError(humanError(err));
    } finally {
      setBusy(false);
    }
  };

  if (error && !products) {
    return <Alert severity="error">{error}</Alert>;
  }
  if (!products) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 2 }}>
        <TextField
          placeholder="Поиск по названию, slug или категории"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          size="small"
          sx={{ flexGrow: 1, maxWidth: 480 }}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
            },
          }}
        />
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setEditing('new')}>
          Добавить товар
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>
          {error}
        </Alert>
      )}

      {products.length === 0 ? (
        <Paper variant="outlined" sx={{ p: 5, textAlign: 'center' }}>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Каталог пуст
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 2 }}>
            Пока каталог пуст, ИИ не рекомендует никакие товары — отвечает только советами.
            Добавьте первое удобрение, и оно сразу станет доступно для рекомендаций.
          </Typography>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => setEditing('new')}>
            Добавить первый товар
          </Button>
        </Paper>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Название</TableCell>
                <TableCell>Категория</TableCell>
                <TableCell>Помогает при</TableCell>
                <TableCell align="right">Цена, ₽</TableCell>
                <TableCell align="center">Активен</TableCell>
                <TableCell align="right">Действия</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filtered.map((p) => (
                <TableRow key={p.id} hover sx={{ opacity: p.active ? 1 : 0.55 }}>
                  <TableCell>
                    <Typography variant="body2" fontWeight={600}>
                      {p.name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {p.slug}
                    </Typography>
                  </TableCell>
                  <TableCell>{p.category}</TableCell>
                  <TableCell sx={{ maxWidth: 280 }}>
                    <Stack direction="row" spacing={0.5} useFlexGap flexWrap="wrap">
                      {p.problems.slice(0, 3).map((key) => (
                        <Chip key={key} label={problemLabel(key)} size="small" />
                      ))}
                      {p.problems.length > 3 && (
                        <Chip label={`+${p.problems.length - 3}`} size="small" variant="outlined" />
                      )}
                    </Stack>
                  </TableCell>
                  <TableCell align="right">
                    {p.price_rub != null ? p.price_rub.toLocaleString('ru-RU') : '—'}
                  </TableCell>
                  <TableCell align="center">
                    <Switch
                      checked={p.active}
                      onChange={() => void toggleActive(p)}
                      size="small"
                      slotProps={{ input: { 'aria-label': 'Активен' } }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <Button size="small" startIcon={<EditIcon />} onClick={() => setEditing(p)}>
                      Изменить
                    </Button>
                    <Button
                      size="small"
                      color="error"
                      startIcon={<DeleteOutlineIcon />}
                      onClick={() => setDeleting(p)}
                    >
                      Удалить
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {editing !== null && (
        <ProductDialog
          product={editing === 'new' ? null : editing}
          problemKeys={problemKeys}
          busy={busy}
          onClose={() => setEditing(null)}
          onSave={save}
        />
      )}

      <Dialog open={deleting !== null} onClose={() => setDeleting(null)}>
        <DialogTitle>Удалить товар?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            «{deleting?.name}» будет удалён навсегда и сразу исчезнет из рекомендаций ИИ. Если
            нужно убрать его временно — выключите переключатель «Активен».
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleting(null)}>Отмена</Button>
          <Button color="error" variant="contained" disabled={busy} onClick={() => void remove()}>
            Удалить
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={toast !== ''}
        autoHideDuration={4000}
        onClose={() => setToast('')}
        message={toast}
      />
    </Box>
  );
}
