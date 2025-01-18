"use client"

import { ActionIcon, Group, TextInput } from "@mantine/core"
import { TimeInput } from "@mantine/dates"
import { IconCheck } from "@tabler/icons-react"
import { useState } from "react"

type Props = {
  onAddTimestamp: (timestamp: string, description: string) => void
}

export default function NewTimestampInput({ onAddTimestamp }: Props) {
  const [newTimestamp, setNewTimestamp] = useState<string>("")
  const [newDescription, setNewDescription] = useState<string>("")

  return (
    <Group>
      <TimeInput
        value={newTimestamp}
        onChange={(e) => setNewTimestamp(e.currentTarget.value)}
        withSeconds
      />
      <TextInput
        value={newDescription}
        onChange={(e) => setNewDescription(e.currentTarget.value)}
      />
      <ActionIcon
        color="green"
        disabled={!newTimestamp || !newDescription}
        onClick={() => {
          onAddTimestamp(newTimestamp, newDescription)
          setNewTimestamp("")
          setNewDescription("")
        }}
      >
        <IconCheck />
      </ActionIcon>
    </Group>
  )
}
