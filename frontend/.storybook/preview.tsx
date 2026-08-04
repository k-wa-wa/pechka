import type { Preview } from '@storybook/nextjs-vite'
import { mswLoader } from 'msw-storybook-addon/csf3'
import { LanguageProvider } from '../lib/i18n/LanguageContext'
import { handlers } from '../mocks/handlers'
import '../app/globals.css'

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
       color: /(background|color)$/i,
       date: /Date$/i,
      },
    },
    msw: { handlers },
    backgrounds: {
      default: 'pechka-dark',
      values: [{ name: 'pechka-dark', value: '#0d1117' }],
    },
  },
  loaders: [mswLoader()],
  decorators: [
    (Story) => (
      <LanguageProvider>
        <Story />
      </LanguageProvider>
    ),
  ],
};

export default preview;