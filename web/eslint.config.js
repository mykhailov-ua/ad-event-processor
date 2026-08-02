import globals from 'globals';
import jsdoc from 'eslint-plugin-jsdoc';

export default [
  { ignores: ['dist', 'node_modules', 'test-results', 'playwright-report', 'admin/**'] },
  {
    files: ['**/*.js'],
    languageOptions: {
      ecmaVersion: 2024,
      globals: {
        ...globals.browser,
      },
    },
    plugins: {
      jsdoc,
    },
    rules: {
      'no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^[A-Z]' }],
      'jsdoc/require-jsdoc': 'off',
      'jsdoc/check-param-names': 'warn',
      'jsdoc/check-tag-names': 'warn',
    },
  },
];
