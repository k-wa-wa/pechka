import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import { http, HttpResponse, delay } from 'msw'
import UploadModal from './UploadModal'

const meta: Meta<typeof UploadModal> = {
  title: 'Components/UploadModal',
  component: UploadModal,
  parameters: { layout: 'fullscreen' },
  args: {
    onClose: () => {},
    onUploaded: () => {},
  },
}

export default meta
type Story = StoryObj<typeof UploadModal>

export const Default: Story = {}

export const UploadFails: Story = {
  parameters: {
    msw: {
      handlers: [
        http.post('/api/v1/admin/contents/upload', async () => {
          await delay(300)
          return HttpResponse.json({ error: 'upload failed' }, { status: 500 })
        }),
      ],
    },
  },
}
