"use client"

import { Video } from "@/app/types"
import { Text } from "@mantine/core"
import Player from "next-video/player"
import EditableTitle from "@/components/EditableTitle"

type Props = {
  video: Video
}
export default function VideoView({ video }: Props) {
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
      <Player src={video.url} />

      <EditableTitle title={video.title} onUpdateTitle={onUpdateTitle} />

      <Text>{video.description}</Text>
    </>
  )
}
