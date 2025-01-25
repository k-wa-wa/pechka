"use client"

import { VideoTimestamp } from "@/app/types"
import { Badge, Group, Space, Stack, Text } from "@mantine/core"
import { useState } from "react"
import NewTimestampInput from "./NewTimestampInput"

type Props = {
  videoId: string
  timestamps: VideoTimestamp[]
}
export default function TimestampView({
  videoId,
  timestamps: defaultTimestamps,
}: Props) {
  const [timestamps, setTimestamps] =
    useState<VideoTimestamp[]>(defaultTimestamps)

  async function onAddTimestamp(timestamp: string, description: string) {
    const tempId = `temp-${new Date().toISOString()}`
    setTimestamps([
      ...timestamps,
      {
        timestampId: tempId,
        videoId,
        timestamp,
        description,
      },
    ])
    const res = await fetch(`/api/video-timestamps/${videoId}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        timestamp,
        description,
      }),
    })

    const assignedId = ((await res.json()) as VideoTimestamp).timestampId
    setTimestamps([
      ...timestamps.filter(({ timestampId }) => timestampId != tempId),
      {
        timestampId: assignedId,
        videoId,
        timestamp,
        description,
      },
    ])
  }

  function onTimestampSelected(target: string) {
    const video = window?.document?.getElementById(videoId)
    if (!video) return
    const [h, m, s] = target.split(":")
    ;(video as HTMLVideoElement).currentTime =
      (Number(h) || 0) * 3600 + (Number(m) || 0) * 60 + (Number(s) || 0)
  }

  function getCurrentTimestamp() {
    return (
      window?.document?.getElementById(videoId) as HTMLVideoElement | null
    )?.currentTime
  }

  return (
    <Stack gap="2">
      {timestamps.map(({ timestampId, timestamp, description }) => (
        <Group key={timestampId} gap="8">
          <Badge
            variant="outline"
            color="gray"
            size="sm"
            style={{ cursor: "pointer" }}
            onClick={() => onTimestampSelected(timestamp)}
          >
            {timestamp}
          </Badge>
          <Text>{description}</Text>
        </Group>
      ))}

      <Space h="xs" />
      <NewTimestampInput
        onAddTimestamp={onAddTimestamp}
        getCurrentTimestamp={getCurrentTimestamp}
      />
    </Stack>
  )
}
