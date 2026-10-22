import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import ChannelView from './ChannelView'
import { CHANNELS, CONTENTS } from '@/mocks/fixtures'

const READY = CONTENTS.filter((c) => c.status === 'ready')

const meta: Meta<typeof ChannelView> = {
  title: 'Components/ChannelView',
  component: ChannelView,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj<typeof ChannelView>

export const Default: Story = {
  args: {
    channel: CHANNELS[1],
    contents: READY.filter((c) => c.channel_id === CHANNELS[1].id),
    currentType: '',
  },
}

export const Empty: Story = {
  args: {
    channel: CHANNELS[0],
    contents: [],
    currentType: '',
  },
}
