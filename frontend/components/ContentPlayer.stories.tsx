import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { VARIANTS } from "@/mocks/fixtures";
import ContentPlayer from "./ContentPlayer";

const meta: Meta<typeof ContentPlayer> = {
  title: "Components/ContentPlayer",
  component: ContentPlayer,
  parameters: { layout: "padded" },
  args: {
    shortId: "vid001",
    hasSubtitles: false,
  },
};

export default meta;
type Story = StoryObj<typeof ContentPlayer>;

export const Video: Story = {
  args: { variants: VARIANTS.vid001, isVr: false },
};

export const NoVariants: Story = {
  args: { variants: [], isVr: false },
};
