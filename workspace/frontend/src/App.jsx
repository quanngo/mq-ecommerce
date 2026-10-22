import { Navigate, Route, Routes } from 'react-router-dom';
import { ProtectedRoute } from './auth/ProtectedRoute';
import { AppLayout } from './pages/AppLayout';
import { ChangePasswordPage } from './pages/ChangePasswordPage';
import { LoginPage } from './pages/LoginPage';
import { OrderDetailPage } from './pages/OrderDetailPage';
import { OrdersPage } from './pages/OrdersPage';
import { ProductImagesPage } from './pages/ProductImagesPage';
import { ProfilePage } from './pages/ProfilePage';
import { UsersPage } from './pages/UsersPage';

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="/account/password" element={<ChangePasswordPage />} />
          <Route element={<ProtectedRoute permission="orders.read" />}>
            <Route path="/orders" element={<OrdersPage />} />
          </Route>
          <Route element={<ProtectedRoute permission="orders.detail" />}>
            <Route path="/orders/:id" element={<OrderDetailPage />} />
          </Route>
          <Route element={<ProtectedRoute permission="orders.write" />}>
            <Route path="/product-images" element={<ProductImagesPage />} />
          </Route>
          <Route element={<ProtectedRoute permission="users.read" />}>
            <Route path="/users" element={<UsersPage />} />
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/profile" replace />} />
    </Routes>
  );
}
