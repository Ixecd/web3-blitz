import { createContext, useContext, useState, useEffect, useCallback } from 'react'

export type ThemeId = 'void' | 'glacier' | 'grove' | 'ember' | 'aurum'

export interface ThemeMeta {
  id:    ThemeId
  name:  string
  dark:  boolean   // 深色主题 or 亮色主题
  color: string    // accent 预览色（hex）
}

export const THEMES: ThemeMeta[] = [
  { id: 'void',    name: 'Void',    dark: true,  color: '#5b21b6' },
  { id: 'glacier', name: 'Glacier', dark: false, color: '#0ea5e9' },
  { id: 'grove',   name: 'Grove',   dark: true,  color: '#16a34a' },
  { id: 'ember',   name: 'Ember',   dark: false, color: '#ea580c' },
  { id: 'aurum',   name: 'Aurum',   dark: true,  color: '#c9962a' },
]

interface ThemeContextValue {
  theme:     ThemeMeta
  setTheme:  (id: ThemeId) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

const STORAGE_KEY = 'blitz-theme'
const DEFAULT_THEME = 'void'

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [themeId, setThemeId] = useState<ThemeId>(() => {
    const saved = localStorage.getItem(STORAGE_KEY)
    return (saved as ThemeId) ?? DEFAULT_THEME
  })

  // 同步 data-theme 到 html 根节点
  useEffect(() => {
    const root = document.documentElement
    root.setAttribute('data-theme', themeId)

    // 深色主题加 dark class 供 Tailwind dark: 使用
    const meta = THEMES.find(t => t.id === themeId)
    if (meta?.dark) {
      root.classList.add('dark')
    } else {
      root.classList.remove('dark')
    }
  }, [themeId])

  const setTheme = useCallback((id: ThemeId) => {
    setThemeId(id)
    localStorage.setItem(STORAGE_KEY, id)
  }, [])

  const theme = THEMES.find(t => t.id === themeId) ?? THEMES[0]

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be inside ThemeProvider')
  return ctx
}
