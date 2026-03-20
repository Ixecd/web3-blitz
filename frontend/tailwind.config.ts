import type { Config } from 'tailwindcss'

export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        display: ['Syne', 'sans-serif'],
        body:    ['DM Sans', 'sans-serif'],
        mono:    ['IBM Plex Mono', 'monospace'],
      },
      colors: {
        // 所有颜色指向 CSS 变量，主题切换只需改变量值
        base:    'var(--color-base)',
        surface: 'var(--color-surface)',
        raised:  'var(--color-raised)',
        overlay: 'var(--color-overlay)',
        border: {
          DEFAULT: 'var(--color-border)',
          strong:  'var(--color-border-strong)',
        },
        accent: {
          DEFAULT: 'var(--color-accent)',
          dim:     'var(--color-accent-dim)',
          muted:   'var(--color-accent-muted)',
          glow:    'var(--color-accent-glow)',
        },
        text: {
          primary: 'var(--color-text-primary)',
          muted:   'var(--color-text-muted)',
          faint:   'var(--color-text-faint)',
        },
        success: 'var(--color-success)',
        danger:  'var(--color-danger)',
        warning: 'var(--color-warning)',
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      animation: {
        'fade-up':    'fadeUp 0.35s cubic-bezier(0.16,1,0.3,1) both',
        'fade-in':    'fadeIn 0.2s ease both',
        'pulse-accent': 'pulseAccent 2s ease-in-out infinite',
      },
      keyframes: {
        fadeUp: {
          '0%':   { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        fadeIn: {
          '0%':   { opacity: '0' },
          '100%': { opacity: '1' },
        },
        pulseAccent: {
          '0%, 100%': { opacity: '1' },
          '50%':      { opacity: '0.4' },
        },
      },
    },
  },
  plugins: [],
} satisfies Config
