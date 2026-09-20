import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import AdminView from './AdminView'
import { ADMIN_CONTENTS } from '@/mocks/fixtures'

const meta: Meta<typeof AdminView> = {
  title: 'Components/AdminView',
  component: AdminView,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj<typeof AdminView>

export const Default: Story = {
  args: { contents: ADMIN_CONTENTS, total: ADMIN_CONTENTS.length, limit: 20, offset: 0 },
}

export const Empty: Story = {
  args: { contents: [], total: 0, limit: 20, offset: 0 },
}

export const MultiplePages: Story = {
  args: { contents: ADMIN_CONTENTS.slice(0, 2), total: 45, limit: 2, offset: 0 },
}
