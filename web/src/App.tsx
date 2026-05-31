import { BrowserRouter, Routes, Route } from 'react-router-dom';
import ProtectedRoute from './components/ProtectedRoute';
import DashboardLayout from './components/DashboardLayout';
import HomePage from './pages/HomePage';
import Login from './pages/Login';
import Register from './pages/Register';
import ModelMarket from './pages/ModelMarket';
import Dashboard from './pages/Dashboard';
import Account from './pages/dashboard/Account';
import TeamManagement from './pages/dashboard/TeamManagement';
import TeamDetail from './pages/dashboard/TeamDetail';
import ApiKeysManagement from './pages/dashboard/ApiKeysManagement';
import AdminLayout from './pages/dashboard/AdminLayout';
import AdminTeams from './pages/dashboard/AdminTeams';
import AdminSettings from './pages/dashboard/AdminSettings';
import AdminLoginLogs from './pages/dashboard/AdminLoginLogs';
import AdminRechargeLogs from './pages/dashboard/AdminRechargeLogs';
import AdminUsers from './pages/dashboard/AdminUsers';
import NotFound from './pages/NotFound';
import { SiteConfigProvider } from './hooks/useSiteConfig';

export default function App() {
  return (
    <SiteConfigProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/models" element={<ModelMarket />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />

          {/* Dashboard - protected */}
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <DashboardLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="account" element={<Account />} />
            <Route path="api-keys" element={<ApiKeysManagement />} />
            <Route path="teams" element={<TeamManagement />} />
            <Route path="teams/:slug" element={<TeamDetail />} />
            <Route path="admin" element={<AdminLayout />}>
              <Route index element={<AdminTeams />} />
              <Route path="teams" element={<AdminTeams />} />
              <Route path="users" element={<AdminUsers />} />
              <Route path="recharge-logs" element={<AdminRechargeLogs />} />
              <Route path="login-logs" element={<AdminLoginLogs />} />
              <Route path="settings" element={<AdminSettings />} />
            </Route>
          </Route>

          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </SiteConfigProvider>
  );
}
