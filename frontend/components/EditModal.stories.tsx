import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import { http, HttpResponse, delay } from 'msw'
import EditModal from './EditModal'
import { ADMIN_CONTENTS } from '@/mocks/fixtures'

const meta: Meta<typeof EditModal> = {
  title: 'Components/EditModal',
  component: EditModal,
  parameters: { layout: 'fullscreen' },
  args: {
    content: ADMIN_CONTENTS[0],
    onClose: () => {},
    onSave: () => {},
  },
}

export default meta
type Story = StoryObj<typeof EditModal>

export const Default: Story = {}

export const SaveFails: Story = {
  parameters: {
    msw: {
      handlers: [
        http.put('/api/v1/admin/contents/:id', async () => {
          await delay(300)
          return HttpResponse.json({ error: 'save failed' }, { status: 500 })
        }),
      ],
    },
  },
}
