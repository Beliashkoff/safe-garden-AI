import { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import { api, humanError } from '../api/client';
import type { AuditEntry } from '../api/types';
import { actionLabel } from '../labels';

const PAGE = 50;

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

export function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = async (nextOffset: number, append: boolean) => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get<{ audit: AuditEntry[] }>(
        `/audit?limit=${PAGE}&offset=${nextOffset}`,
      );
      const list = res.audit ?? [];
      setHasMore(list.length === PAGE);
      setOffset(nextOffset);
      setEntries((prev) => (append ? [...prev, ...list] : list));
    } catch (err) {
      setError(humanError(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load(0, false);
  }, []);

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          Действия в админ-панели: входы, изменения каталога, смена пароля.
        </Typography>
        <Button startIcon={<RefreshIcon />} onClick={() => void load(0, false)} disabled={loading}>
          Обновить
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Время</TableCell>
              <TableCell>Действие</TableCell>
              <TableCell>Объект</TableCell>
              <TableCell>IP</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {entries.map((e) => (
              <TableRow key={e.id} hover>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(e.created_at)}</TableCell>
                <TableCell>{actionLabel(e.action)}</TableCell>
                <TableCell>
                  {e.entity ? `${e.entity}: ${e.entity_id ?? ''}` : '—'}
                </TableCell>
                <TableCell>{e.ip || '—'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
        {loading ? (
          <CircularProgress size={28} />
        ) : (
          hasMore && (
            <Button variant="outlined" onClick={() => void load(offset + PAGE, true)}>
              Показать ещё
            </Button>
          )
        )}
      </Box>
    </Box>
  );
}
