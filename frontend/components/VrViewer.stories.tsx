import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import VrViewer from './VrViewer'
import { VARIANTS } from '@/mocks/fixtures'

const meta: Meta<typeof VrViewer> = {
  title: 'Components/VrViewer',
  component: VrViewer,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj<typeof VrViewer>

// Renders the real A-Frame scene shell; the video sphere has no actual
// stream to play without a media backend running.
export const Default: Story = {
  args: { variants: VARIANTS.vr001 },
}

export const NoVariants: Story = {
  args: { variants: [] },
}
