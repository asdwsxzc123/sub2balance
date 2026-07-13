import { Navigate, Route, Routes } from 'react-router-dom';
import { AppLayout } from '@/components/layout/AppLayout';
import { ProtectedRoute } from '@/components/guards/ProtectedRoute';
import { useAuth } from '@/hooks/useAuth';
import LoginPage from '@/pages/LoginPage';
import CreateRequestPage from '@/pages/CreateRequestPage';
import MyRequestsPage from '@/pages/MyRequestsPage';
import PasswordResetPage from '@/pages/PasswordResetPage';
import AdminDashboardPage from '@/pages/AdminDashboardPage';
import PendingApprovalsPage from '@/pages/PendingApprovalsPage';
import AllRequestsPage from '@/pages/AllRequestsPage';
import UsersPage from '@/pages/UsersPage';
import GroupPricesPage from '@/pages/GroupPricesPage';
import AuditLogsPage from '@/pages/AuditLogsPage';
import SettingsPage from '@/pages/SettingsPage';

function RootRedirect() {
  const { user, token, loading } = useAuth();
  if (loading) {
    return <div className="flex min-h-screen items-center justify-center text-muted-foreground">加载中…</div>;
  }
  if (!token || !user) return <Navigate to="/login" replace />;
  return <Navigate to={user.role === 'admin' ? '/admin' : '/requests/new'} replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<RootRedirect />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/requests/new" element={<CreateRequestPage />} />
          <Route path="/requests/my" element={<MyRequestsPage />} />
          <Route path="/password-reset" element={<PasswordResetPage />} />
        </Route>
      </Route>

      <Route element={<ProtectedRoute requireAdmin />}>
        <Route element={<AppLayout />}>
          <Route path="/admin" element={<AdminDashboardPage />} />
          <Route path="/admin/pending" element={<PendingApprovalsPage />} />
          <Route path="/admin/requests" element={<AllRequestsPage />} />
          <Route path="/admin/users" element={<UsersPage />} />
          <Route path="/admin/group-prices" element={<GroupPricesPage />} />
          <Route path="/admin/logs" element={<AuditLogsPage />} />
          <Route path="/admin/settings" element={<SettingsPage />} />
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
