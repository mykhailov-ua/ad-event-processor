import type { Config } from 'tailwindcss';
import animate from 'tailwindcss-animate';

const config: Config = {
  darkMode: 'class',
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Inter Variable"', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'monospace'],
        numeric: ['"Inter Variable"', 'Inter', 'system-ui', 'sans-serif'],
      },
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        chart: {
          1: 'hsl(var(--chart-1))',
          2: 'hsl(var(--chart-2))',
          3: 'hsl(var(--chart-3))',
          4: 'hsl(var(--chart-4))',
          5: 'hsl(var(--chart-5))',
        },
        admin: {
          brand: {
            DEFAULT: 'hsl(var(--admin-brand))',
            hover: 'hsl(var(--admin-brand-hover))',
            foreground: 'hsl(var(--admin-brand-fg))',
            soft: 'hsl(var(--admin-brand) / 0.12)',
          },
          positive: 'hsl(var(--admin-positive-fg))',
          negative: 'hsl(var(--admin-negative-fg))',
          warn: {
            DEFAULT: 'hsl(var(--admin-warn-fg))',
            bg: 'hsl(var(--admin-warn-bg))',
            border: 'hsl(var(--admin-warn-border))',
          },
          status: {
            active: 'hsl(var(--admin-status-active))',
            paused: 'hsl(var(--admin-status-paused))',
            draft: 'hsl(var(--admin-status-draft))',
            scheduled: 'hsl(var(--admin-status-scheduled))',
          },
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontSize: {
        'ui-dense': ['0.8125rem', { lineHeight: '1.125rem' }],
        'ui-caption': ['0.6875rem', { lineHeight: '1rem' }],
        'ui-mini': ['0.625rem', { lineHeight: '0.875rem' }],
        'ui-micro': ['0.5rem', { lineHeight: '0.625rem' }],
      },
    },
  },
  plugins: [animate],
};

export default config;
