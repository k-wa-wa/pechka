import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import ContentDetailView from './ContentDetailView'
import { CONTENTS, VARIANTS } from '@/mocks/fixtures'

const meta: Meta<typeof ContentDetailView> = {
  title: 'Components/ContentDetailView',
  component: ContentDetailView,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj<typeof ContentDetailView>

export const Video: Story = {
  args: {
    content: CONTENTS.find((c) => c.short_id === 'vid001')!,
    variants: VARIANTS.vid001,
  },
}

export const Vr360: Story = {
  args: {
    content: CONTENTS.find((c) => c.short_id === 'vr001')!,
    variants: VARIANTS.vr001,
  },
}

export const NoDescription: Story = {
  args: {
    content: { ...CONTENTS.find((c) => c.short_id === 'vid002')!, description: '' },
    variants: VARIANTS.vid002,
  },
}
