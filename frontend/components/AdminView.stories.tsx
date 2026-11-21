import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import type { Content } from "@/lib/types";
import { ADMIN_CONTENTS } from "@/mocks/fixtures";
import AdminView from "./AdminView";

function encodeStoryUpdatedDescCursor(item: Content): string {
  const payload = JSON.stringify({
    sort: "updated_desc",
    v: Date.parse(item.updated_at),
    id: item.short_id,
  });
  const bytes = new TextEncoder().encode(payload);
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

const SORTED_ADMIN_CONTENTS = [...ADMIN_CONTENTS].sort((a, b) => {
  const diff = Date.parse(b.updated_at) - Date.parse(a.updated_at);
  if (diff !== 0) return diff;
  return a.short_id < b.short_id ? -1 : a.short_id > b.short_id ? 1 : 0;
});

const meta: Meta<typeof AdminView> = {
  title: "Components/AdminView",
  component: AdminView,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof AdminView>;

export const Default: Story = {
  args: {
    contents: SORTED_ADMIN_CONTENTS,
    total: SORTED_ADMIN_CONTENTS.length,
    limit: 20,
    nextCursor: null,
    prevCursor: null,
    hasNext: false,
    hasPrev: false,
  },
};

export const Empty: Story = {
  args: {
    contents: [],
    total: 0,
    limit: 20,
    nextCursor: null,
    prevCursor: null,
    hasNext: false,
    hasPrev: false,
  },
};

export const MultiplePages: Story = {
  args: {
    contents: SORTED_ADMIN_CONTENTS.slice(0, 2),
    total: SORTED_ADMIN_CONTENTS.length,
    limit: 2,
    nextCursor: encodeStoryUpdatedDescCursor(SORTED_ADMIN_CONTENTS[1]),
    prevCursor: null,
    hasNext: SORTED_ADMIN_CONTENTS.length > 2,
    hasPrev: false,
  },
};
