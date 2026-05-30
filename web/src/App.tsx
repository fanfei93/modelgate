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
import AdminDashboard from './pages/dashboard/AdminDashboard';
import NotFound from './pages/NotFound';

export default function App() {
  return (
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
          <Route path="admin" element={<AdminDashboard />} />
        </Route>

        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  );
}
