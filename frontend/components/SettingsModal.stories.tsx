import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import SettingsModal from './SettingsModal'

const meta: Meta<typeof SettingsModal> = {
  title: 'Components/SettingsModal',
  component: SettingsModal,
  parameters: { layout: 'fullscreen' },
  args: {
    isOpen: true,
    onClose: () => {},
  },
}

export default meta
type Story = StoryObj<typeof SettingsModal>

export const Default: Story = {}
