import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import ContentPlayer from './ContentPlayer'
import { VARIANTS } from '@/mocks/fixtures'

const meta: Meta<typeof ContentPlayer> = {
  title: 'Components/ContentPlayer',
  component: ContentPlayer,
  parameters: { layout: 'padded' },
  args: {
    shortId: 'vid001',
    hasSubtitles: false,
  },
}

export default meta
type Story = StoryObj<typeof ContentPlayer>

export const Video: Story = {
  args: { variants: VARIANTS.vid001, isVr: false },
}

export const NoVariants: Story = {
  args: { variants: [], isVr: false },
}
