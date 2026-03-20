import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import {
  LayoutDashboard,
  ArrowDownToLine,
  ArrowUpFromLine,
  ShieldCheck,
  LogOut,
  Zap,
} from 'lucide-react'
import ThemeSwitcher from '@/components/ThemeSwitcher'

const nav = [
  { to: '/',         label: '总览',  icon: LayoutDashboard, end: true  },
  { to: '/deposit',  label: '充值',  icon: ArrowDownToLine, end: false },
  { to: '/withdraw', label: '提币',  icon: ArrowUpFromLine, end: false },
  { to: '/admin',    label: '管理',  icon: ShieldCheck,     end: false },
]

export default function Layout() {
  const { username, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div
      className="flex h-screen overflow-hidden"
      style={{ backgroundColor: 'var(--color-base)', color: 'var(--color-text-primary)' }}
    >
      {/* ── Sidebar ─────────────────────────────────── */}
      <aside
        className="w-52 shrink-0 flex flex-col border-r"
        style={{
          backgroundColor: 'var(--color-surface)',
          borderColor: 'var(--color-border)',
        }}
      >
        {/* Logo */}
        <div
          className="px-5 py-6 flex items-center gap-2.5 border-b"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div
            className="w-7 h-7 rounded-lg flex items-center justify-center"
            style={{ backgroundColor: 'var(--color-accent)' }}
          >
            <Zap size={14} style={{ color: 'var(--color-base)' }} strokeWidth={2.5} />
          </div>
          <span className="font-display font-bold text-base tracking-tight">
            BLITZ
          </span>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-3 py-4 space-y-0.5">
          {nav.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-all duration-150"
              style={({ isActive }) => ({
                backgroundColor: isActive ? 'var(--color-accent-muted)' : 'transparent',
                color: isActive ? 'var(--color-accent)' : 'var(--color-text-muted)',
                fontWeight: isActive ? 500 : 400,
              })}
            >
              {({ isActive }) => (
                <>
                  <Icon size={15} strokeWidth={isActive ? 2.5 : 2} />
                  {label}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* Theme switcher */}
        <div
          className="border-t"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <ThemeSwitcher />
        </div>

        {/* User */}
        <div
          className="px-3 py-3 border-t space-y-0.5"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div className="px-3 py-1.5">
            <p className="text-xs font-mono" style={{ color: 'var(--color-text-faint)' }}>
              已登录
            </p>
            <p className="text-sm font-mono truncate" style={{ color: 'var(--color-text-primary)' }}>
              {username}
            </p>
          </div>
          <button
            onClick={handleLogout}
            className="btn-ghost w-full flex items-center gap-3"
          >
            <LogOut size={15} />
            退出
          </button>
        </div>
      </aside>

      {/* ── Main ─────────────────────────────────────── */}
      <main className="flex-1 overflow-auto">
        <div className="mx-auto max-w-3xl px-8 py-8 animate-fade-up">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
