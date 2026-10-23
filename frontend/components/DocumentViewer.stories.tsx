import type { Meta, StoryObj } from '@storybook/nextjs-vite'
import DocumentViewer from './DocumentViewer'
import { SAMPLE_MARKDOWN } from '@/mocks/fixtures'

const meta: Meta<typeof DocumentViewer> = {
  title: 'Components/DocumentViewer',
  component: DocumentViewer,
  parameters: { layout: 'padded' },
}

export default meta
type Story = StoryObj<typeof DocumentViewer>

export const Markdown: Story = {
  args: {
    title: '2024年度 活動報告書',
    documentKey: 'resources/documents/doc001/document.md',
    documentMimeType: 'text/markdown',
    initialMarkdown: SAMPLE_MARKDOWN,
  },
}

export const PDF: Story = {
  args: {
    title: 'システム構成ガイド (PDF)',
    documentKey: 'resources/documents/doc002/document.pdf',
    documentMimeType: 'application/pdf',
  },
}

export const MissingDocument: Story = {
  args: {
    title: '未登録ドキュメント',
    documentKey: null,
    documentMimeType: null,
  },
}
