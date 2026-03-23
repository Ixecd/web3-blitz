import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/contexts/AuthContext'
import { ThemeProvider } from '@/contexts/ThemeContext'
import Layout        from '@/components/Layout'
import Login         from '@/pages/Login'
import Dashboard     from '@/pages/Dashboard'
import Deposit       from '@/pages/Deposit'
import Withdraw      from '@/pages/Withdraw'
import Admin         from '@/pages/Admin'
import ForgotPassword from '@/pages/ForgotPassword'
import ResetPassword  from '@/pages/ResetPassword'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuth()
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <Routes>
          {/* 公开路由 */}
          <Route path="/login"           element={<Login />} />
          <Route path="/forgot-password" element={<ForgotPassword />} />
          <Route path="/reset-password"  element={<ResetPassword />} />

          {/* 私有路由 */}
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Layout />
              </PrivateRoute>
            }
          >
            <Route index             element={<Dashboard />} />
            <Route path="deposit"    element={<Deposit />} />
            <Route path="withdraw"   element={<Withdraw />} />
            <Route path="admin"      element={<Admin />} />
          </Route>
        </Routes>
      </AuthProvider>
    </ThemeProvider>
  )
}
