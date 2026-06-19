/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        // Palette Daydream
        surface: {
          DEFAULT: 'hsl(var(--surface))',
          subtle: 'hsl(var(--surface-subtle))',
          raised: 'hsl(var(--surface-raised))',
          100: 'hsl(var(--surface-100))',
          200: 'hsl(var(--surface-200))',
          300: 'hsl(var(--surface-300))',
          400: 'hsl(var(--surface-400))',
          500: 'hsl(var(--surface-500))',
          600: 'hsl(var(--surface-600))',
          700: 'hsl(var(--surface-700))',
          800: 'hsl(var(--surface-800))',
        },
        border: 'hsl(var(--border))',
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          hover: 'hsl(var(--accent-hover))',
          muted: 'hsl(var(--accent-muted))',
        },
        text: {
          DEFAULT: 'hsl(var(--text))',
          muted: 'hsl(var(--text-muted))',
          subtle: 'hsl(var(--text-subtle))',
        },
        hp: 'hsl(var(--hp))',
        mp: 'hsl(var(--mp))',
        stm: 'hsl(var(--stm))',
        danger: 'hsl(var(--danger))',
        success: 'hsl(var(--success))',
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      animation: {
        'screen-shake': 'shake 0.4s ease-in-out',
        'red-flash': 'redFlash 0.3s ease-in-out',
        'golden-glow': 'goldenGlow 0.6s ease-in-out',
        'pulse-slow': 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        shake: {
          '0%, 100%': { transform: 'translateX(0)' },
          '20%': { transform: 'translateX(-6px)' },
          '40%': { transform: 'translateX(6px)' },
          '60%': { transform: 'translateX(-4px)' },
          '80%': { transform: 'translateX(4px)' },
        },
        redFlash: {
          '0%, 100%': { backgroundColor: 'transparent' },
          '50%': { backgroundColor: 'rgba(239, 68, 68, 0.15)' },
        },
        goldenGlow: {
          '0%, 100%': { boxShadow: 'none' },
          '50%': { boxShadow: '0 0 20px rgba(234, 179, 8, 0.4)' },
        },
      },
    },
  },
  plugins: [],
}
