import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import Carousel from './Carousel'
import { CONTENTS } from '@/mocks/fixtures'

const meta: Meta<typeof Carousel> = {
  title: 'Components/Carousel',
  component: Carousel,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj<typeof Carousel>

export const Default: Story = {
  args: { items: CONTENTS.filter((c) => c.status === 'ready').slice(0, 6) },
}

export const SingleItem: Story = {
  args: { items: [CONTENTS[0]] },
}

export const Empty: Story = {
  args: { items: [] },
}
