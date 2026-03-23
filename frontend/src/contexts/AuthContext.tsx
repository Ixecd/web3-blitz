import { createContext, useContext, useState, useCallback } from 'react'
import { api } from '@/api/client'

interface AuthState {
  token:  string | null
  email:  string | null
  userID: number | null
}

interface AuthContextValue extends AuthState {
  login:  (email: string, password: string, rememberMe?: boolean) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

// 同时检查两个 storage，login 时根据 rememberMe 决定用哪个
function getItem(key: string): string | null {
  return localStorage.getItem(key) ?? sessionStorage.getItem(key)
}

function setItem(key: string, value: string, persist: boolean) {
  if (persist) {
    localStorage.setItem(key, value)
    sessionStorage.removeItem(key)
  } else {
    sessionStorage.setItem(key, value)
    localStorage.removeItem(key)
  }
}

function clearAll() {
  ;['access_token', 'refresh_token', 'email', 'user_id'].forEach(k => {
    localStorage.removeItem(k)
    sessionStorage.removeItem(k)
  })
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    token:  getItem('access_token'),
    email:  getItem('email'),
    userID: Number(getItem('user_id')) || null,
  })

  const login = useCallback(async (email: string, password: string, rememberMe = false) => {
    const data = await api.post<{
      access_token:  string
      refresh_token: string
      email:         string
      user_id:       number
    }>('/api/v1/login', { email, password })

    setItem('access_token',  data.access_token,  rememberMe)
    setItem('refresh_token', data.refresh_token, rememberMe)
    setItem('email',         data.email,          rememberMe)
    setItem('user_id',       String(data.user_id), rememberMe)

    setState({ token: data.access_token, email: data.email, userID: data.user_id })
  }, [])

  const logout = useCallback(async () => {
    const rt = getItem('refresh_token')
    if (rt) {
      await api.post('/api/v1/logout', { refresh_token: rt }).catch(() => {})
    }
    clearAll()
    setState({ token: null, email: null, userID: null })
  }, [])

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be inside AuthProvider')
  return ctx
}
