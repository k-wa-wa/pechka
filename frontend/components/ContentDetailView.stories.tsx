import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { expect, within } from "storybook/test";
import { CONTENTS, VARIANTS } from "@/mocks/fixtures";
import ContentDetailView from "./ContentDetailView";

const meta: Meta<typeof ContentDetailView> = {
  title: "Components/ContentDetailView",
  component: ContentDetailView,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof ContentDetailView>;

export const Video: Story = {
  args: {
    content: CONTENTS.find((c) => c.short_id === "vid001")!,
    variants: VARIANTS.vid001,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const channel = canvas.getByRole("link", { name: "Blu-ray" });
    await expect(channel).toHaveAttribute("href", "/channels/bluray");
    await expect(channel).toHaveTextContent("›");
    await expect(canvas.getByText("#夏")).toBeInTheDocument();
  },
};

export const Vr360: Story = {
  args: {
    content: CONTENTS.find((c) => c.short_id === "vr001")!,
    variants: VARIANTS.vr001,
  },
};

export const NoDescription: Story = {
  args: {
    content: {
      ...CONTENTS.find((c) => c.short_id === "vid002")!,
      description: "",
    },
    variants: VARIANTS.vid002,
  },
};

export const NoTags: Story = {
  args: {
    content: {
      ...CONTENTS.find((c) => c.short_id === "vid001")!,
      tags: [],
    },
    variants: VARIANTS.vid001,
  },
};

export const LongChannelNameManyTags: Story = {
  args: {
    content: {
      ...CONTENTS.find((c) => c.short_id === "vid001")!,
      channel_name: "とても長いチャンネル名のサンプル Tech Feed Weekly Digest",
      tags: ["tech-feed", "kubernetes", "go", "frontend", "storybook", "夏", "海", "思い出"],
    },
    variants: VARIANTS.vid001,
  },
};

export const DocumentMarkdown: Story = {
  args: {
    content: CONTENTS.find((c) => c.short_id === "doc001")!,
    variants: [],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const channel = canvas.getByRole("link", { name: "Tech Feed" });
    await expect(channel).toHaveAttribute("href", "/channels/tech-feed");
    await expect(canvas.getByText("#報告")).toBeInTheDocument();
  },
};

export const DocumentPDF: Story = {
  args: {
    content: {
      ...CONTENTS.find((c) => c.short_id === "doc001")!,
      short_id: "doc002",
      title: "システム構成ガイド (PDF)",
      document_key: "resources/documents/doc002/document.pdf",
      document_mime_type: "application/pdf",
    },
    variants: [],
  },
};
