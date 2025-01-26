"use client"

import { Title, Group, Input, ActionIcon } from "@mantine/core"
import { IconCheck, IconPencilMinus, IconX } from "@tabler/icons-react"
import { useState } from "react"

type Props = {
  title: string
  onUpdateTitle?: (title: string) => void
}

export default function EditableTitle(props: Props) {
  const [isEditable, setIsEditable] = useState(false)
  const [title, setTitle] = useState(props.title)

  function onUpdateTitle() {
    setIsEditable(false)
    props.onUpdateTitle?.(title)
  }

  function onCancel() {
    setIsEditable(false)
    setTitle(props.title)
  }

  return (
    <>
      {isEditable ? (
        <Group w="640px">
          {/* レスポンシブ */}
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            w="480px"
          />
          <ActionIcon color="green" variant="outline">
            <IconCheck onClick={() => onUpdateTitle()} />
          </ActionIcon>
          <ActionIcon variant="outline" color="red">
            <IconX onClick={() => onCancel()} />
          </ActionIcon>
        </Group>
      ) : (
        <Group>
          <Title order={2}>{title}</Title>
          <IconPencilMinus
            cursor="pointer"
            onClick={() => setIsEditable(true)}
          />
        </Group>
      )}
    </>
  )
}
