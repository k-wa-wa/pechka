import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import VideoPlayer from './VideoPlayer'
import { VARIANTS } from '@/mocks/fixtures'

const meta: Meta<typeof VideoPlayer> = {
  title: 'Components/VideoPlayer',
  component: VideoPlayer,
  parameters: { layout: 'padded' },
  args: {
    shortId: 'vid001',
    hasSubtitles: false,
  },
}

export default meta
type Story = StoryObj<typeof VideoPlayer>

// hls_key points at a non-existent stream in Storybook — hls.js reports a
// real HLS load error, which is the expected fallback UI without a media
// backend running.
export const MultipleQualities: Story = {
  args: { variants: VARIANTS.vid001 },
}

export const NotFound: Story = {
  args: { variants: [] },
}
