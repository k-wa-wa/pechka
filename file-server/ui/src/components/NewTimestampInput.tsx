import { timeToHHMMSS } from "@/src/utils/time"
import { ActionIcon, Group, Stack, TextInput, Text } from "@mantine/core"
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
        <Stack gap="4px">
          <TextInput
            key={form.key("newTimestamp")}
            {...form.getInputProps("newTimestamp")}
            description="タイムスタンプ"
            w="150px"
          />
          <Group gap="0">
            <Text ml="xs" size="xs" c="dimmed">
              (
            </Text>
            <Text
              size="xs"
              c="dimmed"
              td="underline"
              onClick={() => {
                const currentTimestamp = getCurrentTimestamp()
                if (!currentTimestamp) return
                form.setFieldValue(
                  "newTimestamp",
                  timeToHHMMSS(currentTimestamp)
                )
              }}
              style={{ cursor: "pointer" }}
            >
              再生位置を取得
            </Text>
            <Text size="xs" c="dimmed">
              )
            </Text>
          </Group>
        </Stack>
        <TextInput
          key={form.key("newDescription")}
          {...form.getInputProps("newDescription")}
          description="説明"
        />

        <Group mt="22px">
          <ActionIcon color="green" type="submit">
            <IconCheck />
          </ActionIcon>
        </Group>
      </Group>
    </form>
  )
}
