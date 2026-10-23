import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { delay, HttpResponse, http } from "msw";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import AdminTable from "./AdminTable";

const meta: Meta<typeof AdminTable> = {
  title: "Components/AdminTable",
  component: AdminTable,
  parameters: { layout: "padded" },
};

export default meta;
type Story = StoryObj<typeof AdminTable>;

export const Default: Story = {
  args: { initialContents: ADMIN_CONTENTS },
};

export const Empty: Story = {
  args: { initialContents: [] },
};

export const ArchiveFails: Story = {
  args: { initialContents: ADMIN_CONTENTS },
  parameters: {
    msw: {
      handlers: [
        http.post("/api/v1/admin/contents/:id/archive", async () => {
          await delay(300);
          return HttpResponse.json({ error: "archive failed" }, { status: 500 });
        }),
      ],
    },
  },
};
