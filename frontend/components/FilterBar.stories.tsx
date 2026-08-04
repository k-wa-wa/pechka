import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import FilterBar, { FilterOption } from './FilterBar'

const TYPES: FilterOption[] = [
  { value: '', label: 'All' },
  { value: 'video', label: 'Video' },
  { value: 'image_gallery', label: 'Gallery' },
  { value: 'vr360', label: 'VR360' },
  { value: 'document', label: 'Document' },
]

const meta: Meta<typeof FilterBar> = {
  title: 'Components/FilterBar',
  component: FilterBar,
  parameters: { layout: 'padded' },
  args: { types: TYPES, currentType: '' },
}

export default meta
type Story = StoryObj<typeof FilterBar>

export const AllSelected: Story = {
  args: { currentType: '' },
}

export const VideoSelected: Story = {
  args: { currentType: 'video' },
}
