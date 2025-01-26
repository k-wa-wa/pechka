"use client"

import { VideoTimestamp } from "@/app/types"
import {
  Badge,
  Card,
  CardSection,
  Group,
  Menu,
  MenuDropdown,
  MenuItem,
  MenuTarget,
  ScrollAreaAutosize,
  Stack,
  Text,
} from "@mantine/core"
import { useState } from "react"
import NewTimestampInput from "./NewTimestampInput"
import { HHMMSStoTime } from "@/utils/time"
import {
  IconClockRecord,
  IconDotsVertical,
  IconTrash,
} from "@tabler/icons-react"

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

  async function onDeleteTimestamp(timestampId: string) {
    await fetch(`/api/video-timestamps/${timestampId}`, {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
      },
    }).then((res) => res.json())

    setTimestamps([
      ...timestamps.filter(({ timestampId: id }) => id != timestampId),
    ])
  }

  function onTimestampSelected(target: string) {
    const video = window?.document?.getElementById(videoId)
    if (!video) return
    ;(video as HTMLVideoElement).currentTime = HHMMSStoTime(target)
  }

  function getCurrentTimestamp() {
    return (
      window?.document?.getElementById(videoId) as HTMLVideoElement | null
    )?.currentTime
  }

  return (
    <Card shadow="sm" padding="md" radius="md" withBorder>
      <CardSection withBorder inheritPadding py="xs" bg="gray.1">
        <Group gap="xs">
          <Text fw={500}>タイムスタンプ</Text>
          <IconClockRecord size="20" />
        </Group>
      </CardSection>

      <Stack>
        <ScrollAreaAutosize mah={250}>
          <Stack gap="2" mt="4">
            {timestamps
              .sort(
                (a, b) => HHMMSStoTime(a.timestamp) - HHMMSStoTime(b.timestamp)
              )
              .map(({ timestampId, timestamp, description }) => (
                <Group key={timestampId} gap="8">
                  <Group gap="4">
                    <Menu trigger="click-hover" position="left-start">
                      <MenuTarget>
                        <IconDotsVertical size="16px" />
                      </MenuTarget>
                      <MenuDropdown>
                        <MenuItem
                          color="red"
                          leftSection={<IconTrash size="16px" />}
                          onClick={() => onDeleteTimestamp(timestampId)}
                        >
                          削除
                        </MenuItem>
                      </MenuDropdown>
                    </Menu>
                    <Badge
                      variant="outline"
                      color="gray"
                      size="sm"
                      style={{ cursor: "pointer" }}
                      onClick={() => onTimestampSelected(timestamp)}
                    >
                      {timestamp}
                    </Badge>
                  </Group>

                  <Text>{description}</Text>
                </Group>
              ))}
          </Stack>
        </ScrollAreaAutosize>

        <NewTimestampInput
          onAddTimestamp={onAddTimestamp}
          getCurrentTimestamp={getCurrentTimestamp}
        />
      </Stack>
    </Card>
  )
}
