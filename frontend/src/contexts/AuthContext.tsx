import { createContext, useContext, useState, useCallback } from 'react'
import { api } from '@/api/client'

interface AuthState {
  token:    string | null
  username: string | null
  userID:   number | null
}

interface AuthContextValue extends AuthState {
  login:  (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    token:    localStorage.getItem('access_token'),
    username: localStorage.getItem('username'),
    userID:   Number(localStorage.getItem('user_id')) || null,
  })

  const login = useCallback(async (username: string, password: string) => {
    const data = await api.post<{
      access_token:  string
      refresh_token: string
      username:      string
      user_id:       number
    }>('/api/v1/login', { username, password })

    localStorage.setItem('access_token',  data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
    localStorage.setItem('username',      data.username)
    localStorage.setItem('user_id',       String(data.user_id))

    setState({ token: data.access_token, username: data.username, userID: data.user_id })
  }, [])

  const logout = useCallback(async () => {
    const rt = localStorage.getItem('refresh_token')
    if (rt) {
      await api.post('/api/v1/logout', { refresh_token: rt }).catch(() => {})
    }
    localStorage.clear()
    setState({ token: null, username: null, userID: null })
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
