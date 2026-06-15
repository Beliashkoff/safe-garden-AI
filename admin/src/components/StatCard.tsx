import { Card, CardContent, Typography } from '@mui/material';

// Маленькая KPI-карточка, общая для страниц аналитики.
export function StatCard({
  title,
  value,
  hint,
  valueColor,
}: {
  title: string;
  value: string;
  hint?: string;
  valueColor?: string;
}) {
  return (
    <Card variant="outlined" sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="body2" color="text.secondary">
          {title}
        </Typography>
        <Typography variant="h5" sx={{ my: 0.5 }} color={valueColor}>
          {value}
        </Typography>
        {hint && (
          <Typography variant="caption" color="text.secondary">
            {hint}
          </Typography>
        )}
      </CardContent>
    </Card>
  );
}
