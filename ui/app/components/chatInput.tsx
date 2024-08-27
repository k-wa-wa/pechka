import { Flex, Textarea, ActionIcon } from "@mantine/core"
import { IoSend } from "react-icons/io5"

type Props = {
  inputMessage: string
  setInputMessage: (inputMessage: string) => void
  onSendMessage: () => void
}
export default function ChatInput({
  inputMessage,
  setInputMessage,
  onSendMessage,
}: Props) {
  const onKeyDown: React.KeyboardEventHandler<HTMLTextAreaElement> = (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      onSendMessage()
    }
  }

  return (
    <Flex align="center" gap="sm">
      <Textarea
        style={{ flexGrow: 1 }}
        autosize
        value={inputMessage}
        onChange={(e) => setInputMessage(e.target.value)}
        onKeyDown={onKeyDown}
      />
      <ActionIcon onClick={onSendMessage}>
        <IoSend />
      </ActionIcon>
    </Flex>
  )
}
