import { Video, VideoTimestamp } from "@/src/types"
import { Text } from "@mantine/core"
import EditableTitle from "@/src/components/EditableTitle"
import HlsPlayer from "@/src/components/HlsPlayer"
import TimestampView from "./TimestampView"

type Props = {
  video: Video
  timestamps: VideoTimestamp[]
}
export default function VideoView({ video, timestamps }: Props) {
  async function onUpdateTitle(title: string) {
    const res = await fetch(`/api/videos/${video.id}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ title, description: video.description }),
    })
    console.log(res)
  }

  return (
    <>
      <HlsPlayer id={video.id} src={video.url} />

      <EditableTitle title={video.title} onUpdateTitle={onUpdateTitle} />

      <Text>{video.description}</Text>

      <TimestampView videoId={video.id} timestamps={timestamps} />
    </>
  )
}
