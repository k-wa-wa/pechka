import { ScrollArea, Flex, Avatar, Paper, Text, Space } from "@mantine/core"
import { Message } from "~/types"

type Props = {
  messages: Message[]
}
export default function ChatMessages({ messages }: Props) {
  return (
    <ScrollArea h="100%">
      {messages.map(({ _id, role, content }) => (
        <Flex
          key={_id}
          align="flex-end"
          gap="xs"
          direction={role === "user" ? "row-reverse" : "row"}
        >
          <Avatar />
          <Paper shadow="md" px="xs">
            <Text style={{ overflowWrap: "break-word" }}>{content}</Text>
          </Paper>
          <Space w="xl" />
        </Flex>
      ))}
    </ScrollArea>
  )
}
