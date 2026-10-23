import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { delay, http } from "msw";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import SubtitleEditorModal from "./SubtitleEditorModal";

const meta: Meta<typeof SubtitleEditorModal> = {
  title: "Components/SubtitleEditorModal",
  component: SubtitleEditorModal,
  parameters: { layout: "fullscreen" },
  args: {
    onClose: () => {},
  },
};

export default meta;
type Story = StoryObj<typeof SubtitleEditorModal>;

// ADMIN_CONTENTS[0] (vid001) has subtitle fixtures wired up in mocks/handlers.ts
export const WithCues: Story = {
  args: { content: ADMIN_CONTENTS[0] },
};

export const NoSubtitles: Story = {
  args: { content: ADMIN_CONTENTS[1] },
};

export const Loading: Story = {
  args: { content: ADMIN_CONTENTS[0] },
  parameters: {
    msw: {
      handlers: [
        http.get("/api/v1/admin/contents/:contentId/subtitles", async () => {
          await delay("infinite");
        }),
      ],
    },
  },
};
