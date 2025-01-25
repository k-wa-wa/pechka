"use client"

import { timeToHHMMSS } from "@/utils/time"
import { ActionIcon, Button, Group, TextInput } from "@mantine/core"
import { useForm } from "@mantine/form"
import { IconCheck } from "@tabler/icons-react"

type Props = {
  onAddTimestamp: (timestamp: string, description: string) => void
  getCurrentTimestamp: () => number | undefined
}

export default function NewTimestampInput({
  onAddTimestamp,
  getCurrentTimestamp,
}: Props) {
  const form = useForm({
    mode: "uncontrolled",
    initialValues: {
      newTimestamp: "00:00:00",
      newDescription: "",
    },
    validate: {
      newTimestamp: (value) =>
        /^([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$/.test(value)
          ? null
          : "Invalid Timestamp",
      newDescription: (value) => (value.length ? null : "required"),
    },
  })

  return (
    <form
      onSubmit={form.onSubmit((values) => {
        onAddTimestamp(values.newTimestamp, values.newDescription)
        form.setFieldValue("newDescription", "")
      })}
    >
      <Group align="flex-start">
        <TextInput
          key={form.key("newTimestamp")}
          {...form.getInputProps("newTimestamp")}
          w="100px"
          placeholder="タイムスタンプ"
        />
        <TextInput
          key={form.key("newDescription")}
          {...form.getInputProps("newDescription")}
          placeholder="説明"
        />
        <Group mt="4">
          <ActionIcon color="green" type="submit">
            <IconCheck />
          </ActionIcon>

          <Button
            size="xs"
            variant="outline"
            onClick={() => {
              const currentTimestamp = getCurrentTimestamp()
              if (!currentTimestamp) return
              form.setFieldValue("newTimestamp", timeToHHMMSS(currentTimestamp))
            }}
          >
            再生位置を取得
          </Button>
        </Group>
      </Group>
    </form>
  )
}
