import type { Metadata } from "next";
import AdminView from "@/components/AdminView";
import { getAdminContents, getChannels } from "@/lib/api";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Admin — pechka",
};

const PAGE_SIZE = 20;

export default async function AdminPage() {
  const [data, channels] = await Promise.all([
    getAdminContents({ limit: PAGE_SIZE, sort: "updated_desc" }).catch(() => ({
      contents: [],
      total: 0,
      limit: PAGE_SIZE,
      next_cursor: null,
      prev_cursor: null,
      has_next: false,
      has_prev: false,
    })),
    getChannels().catch(() => []),
  ]);

  return (
    <AdminView
      contents={data.contents}
      total={data.total}
      limit={data.limit}
      nextCursor={data.next_cursor}
      prevCursor={data.prev_cursor}
      hasNext={data.has_next}
      hasPrev={data.has_prev}
      channels={channels}
    />
  );
}
