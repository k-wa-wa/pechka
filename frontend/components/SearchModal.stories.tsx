import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import { http, HttpResponse, delay } from 'msw'
import { expect, userEvent, within } from 'storybook/test'
import SearchModal from './SearchModal'

const meta: Meta<typeof SearchModal> = {
  title: 'Components/SearchModal',
  component: SearchModal,
  parameters: { layout: 'fullscreen' },
  args: {
    isOpen: true,
    onClose: () => {},
  },
}

export default meta
type Story = StoryObj<typeof SearchModal>

export const Default: Story = {}

export const Loading: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get('/api/v1/search', async () => {
          await delay('infinite')
        }),
      ],
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByPlaceholderText('コンテンツを検索...'), '夏')
    await expect(await canvas.findByText('検索中...')).toBeInTheDocument()
  },
}

export const NoResults: Story = {
  parameters: {
    msw: {
      handlers: [http.get('/api/v1/search', () => HttpResponse.json([]))],
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByPlaceholderText('コンテンツを検索...'), '存在しない検索語')
    await expect(
      await canvas.findByText('」に一致するコンテンツが見つかりません', { exact: false })
    ).toBeInTheDocument()
  },
}
