import type { Config } from 'tailwindcss';
import animate from 'tailwindcss-animate';

const config: Config = {
  darkMode: 'class',
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Inter Variable"', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono Variable"', 'JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      colors: {
        muted: {
          DEFAULT: '#f4f4f5',
          foreground: '#71717a',
        },
        card: {
          DEFAULT: '#ffffff',
          foreground: '#09090b',
        },
        popover: {
          DEFAULT: '#ffffff',
          foreground: '#09090b',
        },
        primary: {
          DEFAULT: '#18181b',
          foreground: '#fafafa',
        },
        secondary: {
          DEFAULT: '#f4f4f5',
          foreground: '#18181b',
        },
        accent: {
          DEFAULT: '#f4f4f5',
          foreground: '#18181b',
        },
        destructive: {
          DEFAULT: '#ef4444',
          foreground: '#fafafa',
        },
        border: '#e4e4e7',
        input: '#e4e4e7',
        ring: '#3b82f6',
        background: '#ffffff',
        foreground: '#09090b',
      },
      borderRadius: {
        lg: '0.5rem',
        md: '0.375rem',
        sm: '0.25rem',
      },
      fontSize: {
        'admin-dense': ['0.8125rem', { lineHeight: '1.125rem' }],
        'admin-caption': ['0.6875rem', { lineHeight: '1rem' }],
        'admin-mini': ['0.625rem', { lineHeight: '0.875rem' }],
        'admin-micro': ['0.5rem', { lineHeight: '0.625rem' }],
      },
    },
  },
  plugins: [animate],
};

export default config;
