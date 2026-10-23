import type { Metadata } from "next";
import { notFound } from "next/navigation";
import ChannelView from "@/components/ChannelView";
import { getChannel, getContents } from "@/lib/api";
import type { Channel, ContentType, MongoContent } from "@/lib/types";

export const dynamic = "force-dynamic";

interface Props {
  params: Promise<{ channelId: string }>;
  searchParams: Promise<{ type?: string }>;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { channelId } = await params;
  try {
    const channel = await getChannel(channelId);
    return { title: `${channel.name} — pechka` };
  } catch {
    return { title: "pechka" };
  }
}

export default async function ChannelPage({ params, searchParams }: Props) {
  const { channelId } = await params;
  const sp = await searchParams;
  const contentType = (sp.type as ContentType) || undefined;

  let channel: Channel;
  let contents: MongoContent[];
  try {
    [channel, contents] = await Promise.all([
      getChannel(channelId),
      getContents({ limit: 100, channel_id: channelId, content_type: contentType }),
    ]);
  } catch {
    notFound();
  }

  const readyContents = contents.filter((c) => c.status === "ready");

  return <ChannelView channel={channel} contents={readyContents} currentType={sp.type ?? ""} />;
}
