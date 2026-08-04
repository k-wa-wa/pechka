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
  args: { contents: ADMIN_CONTENTS },
}

export const Empty: Story = {
  args: { contents: [] },
}
