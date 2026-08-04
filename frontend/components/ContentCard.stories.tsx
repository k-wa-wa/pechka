import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import ContentCard from './ContentCard'
import { CONTENTS } from '@/mocks/fixtures'

const meta: Meta<typeof ContentCard> = {
  title: 'Components/ContentCard',
  component: ContentCard,
  parameters: { layout: 'padded' },
  decorators: [
    (Story) => (
      <div style={{ maxWidth: 320 }}>
        <Story />
      </div>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof ContentCard>

export const Video: Story = {
  args: { content: CONTENTS[0] },
}

export const ImageGallery: Story = {
  args: { content: CONTENTS.find((c) => c.content_type === 'image_gallery')! },
}

export const Vr360: Story = {
  args: { content: CONTENTS.find((c) => c.content_type === 'vr360')! },
}

export const Processing: Story = {
  args: { content: CONTENTS.find((c) => c.status === 'processing')! },
}

export const ManyTags: Story = {
  args: {
    content: {
      ...CONTENTS[0],
      tags: ['夏', '海', '思い出', '家族', '旅行', '花火'],
    },
  },
}
