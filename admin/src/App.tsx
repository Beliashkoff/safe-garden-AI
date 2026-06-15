import { lazy, Suspense } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { Box, CircularProgress } from '@mui/material';
import { useAuth } from './auth/AuthContext';
import { Layout } from './components/Layout';
import { AuthPage } from './pages/AuthPage';

// Защищённые страницы грузятся по требованию: экран входа (самый частый
// холодный старт) не тянет recharts и таблицы.
const DashboardPage = lazy(() =>
  import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })),
);
const GrowthPage = lazy(() => import('./pages/GrowthPage').then((m) => ({ default: m.GrowthPage })));
const QualityPage = lazy(() =>
  import('./pages/QualityPage').then((m) => ({ default: m.QualityPage })),
);
const ReliabilityPage = lazy(() =>
  import('./pages/ReliabilityPage').then((m) => ({ default: m.ReliabilityPage })),
);
const CostPage = lazy(() => import('./pages/CostPage').then((m) => ({ default: m.CostPage })));
const SecurityPage = lazy(() =>
  import('./pages/SecurityPage').then((m) => ({ default: m.SecurityPage })),
);
const CompliancePage = lazy(() =>
  import('./pages/CompliancePage').then((m) => ({ default: m.CompliancePage })),
);
const AssortmentPage = lazy(() =>
  import('./pages/AssortmentPage').then((m) => ({ default: m.AssortmentPage })),
);
const CatalogPage = lazy(() =>
  import('./pages/CatalogPage').then((m) => ({ default: m.CatalogPage })),
);
const ErrorsPage = lazy(() => import('./pages/ErrorsPage').then((m) => ({ default: m.ErrorsPage })));
const AuditPage = lazy(() => import('./pages/AuditPage').then((m) => ({ default: m.AuditPage })));
const SettingsPage = lazy(() =>
  import('./pages/SettingsPage').then((m) => ({ default: m.SettingsPage })),
);

function Spinner() {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}>
      <CircularProgress />
    </Box>
  );
}

export function App() {
  const { loading, admin } = useAuth();

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!admin) {
    return (
      <Routes>
        <Route path="*" element={<AuthPage />} />
      </Routes>
    );
  }

  return (
    <Layout>
      <Suspense fallback={<Spinner />}>
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/growth" element={<GrowthPage />} />
          <Route path="/quality" element={<QualityPage />} />
          <Route path="/cost" element={<CostPage />} />
          <Route path="/reliability" element={<ReliabilityPage />} />
          <Route path="/security" element={<SecurityPage />} />
          <Route path="/compliance" element={<CompliancePage />} />
          <Route path="/catalog" element={<CatalogPage />} />
          <Route path="/assortment" element={<AssortmentPage />} />
          <Route path="/errors" element={<ErrorsPage />} />
          <Route path="/audit" element={<AuditPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </Layout>
  );
}
