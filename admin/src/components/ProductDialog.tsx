import { useState } from 'react';
import {
  Alert,
  Autocomplete,
  Avatar,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Grid,
  InputAdornment,
  MenuItem,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material';
import UploadIcon from '@mui/icons-material/Upload';
import { api, humanError } from '../api/client';
import type { Product, ProductInput } from '../api/types';
import { problemLabel } from '../labels';

interface Props {
  product: Product | null;
  problemKeys: string[];
  busy: boolean;
  onClose: () => void;
  onSave: (input: ProductInput, id?: string) => Promise<void>;
}

// Форма товара. Главное правило продукта: ИИ нативно отдаёт пользователю только
// название и «для чего» (short_desc); цену и ссылку — лишь если пользователь
// спросит. Поэтому подсказки в форме объясняют редактору, что и как покажется.
export function ProductDialog({ product, problemKeys, busy, onClose, onSave }: Props) {
  const [name, setName] = useState(product?.name ?? '');
  const [slug, setSlug] = useState(product?.slug ?? '');
  const [slugTouched, setSlugTouched] = useState(Boolean(product));
  const [category, setCategory] = useState(product?.category ?? '');
  const [shortDesc, setShortDesc] = useState(product?.short_desc ?? '');
  const [longDesc, setLongDesc] = useState(product?.long_desc ?? '');
  const [problems, setProblems] = useState<string[]>(product?.problems ?? []);
  const [plants, setPlants] = useState<string[]>(product?.plants ?? []);
  const [priority, setPriority] = useState(String(product?.priority ?? 0));
  const [hasPrice, setHasPrice] = useState(product?.price_rub != null);
  const [price, setPrice] = useState(product?.price_rub != null ? String(product.price_rub) : '');
  const [imageURL, setImageURL] = useState(product?.image_url ?? '');
  const [deeplinkURL, setDeeplinkURL] = useState(product?.deeplink_url ?? '');
  const [active, setActive] = useState(product?.active ?? true);
  const [error, setError] = useState('');
  const [uploading, setUploading] = useState(false);

  const onNameChange = (value: string) => {
    setName(value);
    if (!slugTouched) {
      // Лёгкая клиентская транслитерация — сервер всё равно нормализует slug.
      setSlug(
        value
          .toLowerCase()
          .replace(/[^a-z0-9Ѐ-ӿ]+/g, '-')
          .replace(/^-+|-+$/g, ''),
      );
    }
  };

  const onProblemsChange = (e: SelectChangeEvent<string[]>) => {
    const v = e.target.value;
    setProblems(typeof v === 'string' ? v.split(',') : v);
  };

  const uploadImage = async (file: File) => {
    setUploading(true);
    setError('');
    try {
      const form = new FormData();
      form.append('file', file);
      const res = await api.upload<{ url: string }>('/fertilizers/image', form);
      setImageURL(res.url);
    } catch (err) {
      setError(humanError(err));
    } finally {
      setUploading(false);
    }
  };

  const submit = async () => {
    setError('');
    if (!name.trim() || !shortDesc.trim() || !category.trim()) {
      setError('Заполните название, краткое описание и категорию.');
      return;
    }
    if (problems.length === 0) {
      setError('Выберите хотя бы одну проблему, при которой помогает товар.');
      return;
    }
    const input: ProductInput = {
      slug: slug.trim(),
      name: name.trim(),
      short_desc: shortDesc.trim(),
      long_desc: longDesc.trim(),
      image_url: imageURL.trim(),
      deeplink_url: deeplinkURL.trim(),
      category: category.trim(),
      problems,
      plants,
      priority: Number.parseInt(priority, 10) || 0,
      active,
      price_rub: hasPrice && price.trim() !== '' ? Number.parseInt(price, 10) : null,
    };
    try {
      await onSave(input, product?.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось сохранить.');
    }
  };

  return (
    <Dialog open onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{product ? 'Редактировать товар' : 'Новый товар'}</DialogTitle>
      <DialogContent dividers>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, sm: 8 }}>
            <TextField
              label="Название"
              value={name}
              onChange={(e) => onNameChange(e.target.value)}
              fullWidth
              required
              helperText="Это название ИИ называет пользователю"
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 4 }}>
            <TextField
              label="Категория"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              fullWidth
              required
              placeholder="например, минеральные"
            />
          </Grid>

          <Grid size={{ xs: 12 }}>
            <TextField
              label="Slug (идентификатор)"
              value={slug}
              onChange={(e) => {
                setSlugTouched(true);
                setSlug(e.target.value);
              }}
              fullWidth
              helperText="Латиницей, для ссылок и аналитики. Заполняется автоматически из названия — можно изменить."
            />
          </Grid>

          <Grid size={{ xs: 12 }}>
            <TextField
              label="Краткое описание — «для чего»"
              value={shortDesc}
              onChange={(e) => setShortDesc(e.target.value)}
              fullWidth
              required
              multiline
              minRows={2}
              helperText="Короткое объяснение пользы. Показывается на карточке товара пользователю."
            />
          </Grid>

          <Grid size={{ xs: 12 }}>
            <TextField
              label="Подробное описание (необязательно)"
              value={longDesc}
              onChange={(e) => setLongDesc(e.target.value)}
              fullWidth
              multiline
              minRows={3}
              helperText="ИИ может использовать, если пользователь попросит подробности."
            />
          </Grid>

          <Grid size={{ xs: 12 }}>
            <Typography variant="body2" sx={{ mb: 0.5 }}>
              При каких проблемах помогает *
            </Typography>
            <Select
              multiple
              value={problems}
              onChange={onProblemsChange}
              fullWidth
              displayEmpty
              renderValue={(selected) =>
                selected.length === 0 ? (
                  <Typography color="text.secondary">Выберите проблемы…</Typography>
                ) : (
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                    {selected.map((key) => (
                      <Chip key={key} label={problemLabel(key)} size="small" />
                    ))}
                  </Box>
                )
              }
            >
              {problemKeys.map((key) => (
                <MenuItem key={key} value={key}>
                  {problemLabel(key)}
                </MenuItem>
              ))}
            </Select>
            <Typography variant="caption" color="text.secondary">
              ИИ подбирает товар по этим проблемам. Это связь между диагнозом и рекомендацией.
            </Typography>
          </Grid>

          <Grid size={{ xs: 12, sm: 6 }}>
            <Autocomplete
              multiple
              freeSolo
              options={[]}
              value={plants}
              onChange={(_, v) => setPlants(v.map((s) => s.toString().toLowerCase().trim()))}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => {
                  const { key, ...rest } = getTagProps({ index });
                  return <Chip key={key} label={option} size="small" {...rest} />;
                })
              }
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Растения (необязательно)"
                  placeholder="томат, огурец…"
                  helperText="Пусто = подходит любому растению"
                />
              )}
            />
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <TextField
              label="Приоритет"
              type="number"
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              fullWidth
              helperText="Выше = раньше в выдаче"
            />
          </Grid>
          <Grid size={{ xs: 6, sm: 3 }}>
            <Stack>
              <FormControlLabel
                control={<Switch checked={hasPrice} onChange={(e) => setHasPrice(e.target.checked)} />}
                label="Указать цену"
              />
              <TextField
                label="Цена"
                type="number"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
                disabled={!hasPrice}
                slotProps={{
                  input: { endAdornment: <InputAdornment position="end">₽</InputAdornment> },
                  htmlInput: { min: 0, step: 1, inputMode: 'numeric' },
                }}
                helperText="Целое число, рубли"
              />
            </Stack>
          </Grid>

          <Grid size={{ xs: 12, sm: 6 }}>
            <Stack direction="row" spacing={2} alignItems="center">
              <Avatar
                variant="rounded"
                src={imageURL || undefined}
                sx={{ width: 64, height: 64, bgcolor: 'action.hover' }}
              />
              <Box sx={{ flexGrow: 1 }}>
                <Button
                  component="label"
                  variant="outlined"
                  startIcon={<UploadIcon />}
                  disabled={uploading}
                  fullWidth
                >
                  {uploading ? 'Загрузка…' : 'Загрузить фото'}
                  <input
                    type="file"
                    hidden
                    accept="image/jpeg,image/png,image/webp"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) void uploadImage(file);
                    }}
                  />
                </Button>
                <Typography variant="caption" color="text.secondary">
                  Необязательно. JPEG/PNG/WebP до 5 МБ.
                </Typography>
              </Box>
            </Stack>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <TextField
              label="Ссылка на товар (необязательно)"
              value={deeplinkURL}
              onChange={(e) => setDeeplinkURL(e.target.value)}
              fullWidth
              placeholder="https://…"
              helperText="Куда ведёт кнопка «Подробнее»"
            />
          </Grid>

          <Grid size={{ xs: 12 }}>
            <FormControlLabel
              control={<Switch checked={active} onChange={(e) => setActive(e.target.checked)} />}
              label="Активен (участвует в рекомендациях ИИ)"
            />
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Отмена</Button>
        <Button variant="contained" onClick={() => void submit()} disabled={busy || uploading}>
          Сохранить
        </Button>
      </DialogActions>
    </Dialog>
  );
}
