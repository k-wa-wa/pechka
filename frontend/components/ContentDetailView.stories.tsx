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

export const DocumentMarkdown: Story = {
  args: {
    content: CONTENTS.find((c) => c.short_id === 'doc001')!,
    variants: [],
  },
}

export const DocumentPDF: Story = {
  args: {
    content: {
      ...CONTENTS.find((c) => c.short_id === 'doc001')!,
      short_id: 'doc002',
      title: 'システム構成ガイド (PDF)',
      document_key: 'resources/documents/doc002/document.pdf',
      document_mime_type: 'application/pdf',
    },
    variants: [],
  },
}
