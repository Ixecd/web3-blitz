import { THEMES, useTheme } from '@/contexts/ThemeContext'

export default function ThemeSwitcher() {
  const { theme, setTheme } = useTheme()

  return (
    <div className="flex items-center gap-1.5 px-3 py-2">
      {THEMES.map(t => (
        <button
          key={t.id}
          onClick={() => setTheme(t.id)}
          title={t.name}
          className="relative w-5 h-5 rounded-full transition-all duration-200 hover:scale-110"
          style={{ backgroundColor: t.color }}
        >
          {/* 选中状态：白色外圈 */}
          {theme.id === t.id && (
            <span
              className="absolute inset-0 rounded-full ring-2 ring-offset-1 ring-offset-[var(--color-surface)]"
              style={{ '--tw-ring-color': t.color } as React.CSSProperties}
            />
          )}
        </button>
      ))}
    </div>
  )
}
