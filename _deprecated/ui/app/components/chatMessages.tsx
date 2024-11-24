import {
  Flex,
  Avatar,
  Paper,
  Space,
  TypographyStylesProvider,
} from "@mantine/core"
import { Message } from "~/types"
import { parse } from "markdown-wasm"
import { memo } from "react"

type Props = {
  messages: Message[]
}

const RenderMessage = memo(function RenderMessage({
  message: { _id, role, content },
}: {
  message: Message
}) {
  function parseMarkdown(markdownContent: string) {
    try {
      return parse(markdownContent)
    } catch (e) {
      return markdownContent
    }
  }

  return (
    <Flex
      key={_id}
      align="flex-end"
      gap="xs"
      direction={role === "user" ? "row-reverse" : "row"}
    >
      <Avatar />
      <Paper shadow="md" px="xs">
        <TypographyStylesProvider>
          <div dangerouslySetInnerHTML={{ __html: parseMarkdown(content) }} />
        </TypographyStylesProvider>
      </Paper>
      <Space w="xl" />
    </Flex>
  )
})

export default function ChatMessages({ messages }: Props) {
  return (
    <Flex h="100%" direction="column" gap="sm">
      {messages.map((message) => (
        <RenderMessage key={message._id} message={message} />
      ))}
    </Flex>
  )
}
