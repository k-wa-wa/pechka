import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import HomeView from './HomeView'
import { CONTENTS } from '@/mocks/fixtures'

const READY = CONTENTS.filter((c) => c.status === 'ready')

const meta: Meta<typeof HomeView> = {
  title: 'Components/HomeView',
  component: HomeView,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj<typeof HomeView>

export const Default: Story = {
  args: {
    carouselReady: READY.slice(0, 6),
    gridItems: READY,
    currentType: '',
  },
}

export const FilteredByVideo: Story = {
  args: {
    carouselReady: READY.slice(0, 6),
    gridItems: READY.filter((c) => c.content_type === 'video'),
    currentType: 'video',
  },
}

export const Empty: Story = {
  args: {
    carouselReady: [],
    gridItems: [],
    currentType: '',
  },
}
