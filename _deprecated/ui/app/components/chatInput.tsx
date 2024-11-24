import { Flex, Textarea, ActionIcon } from "@mantine/core"
import { IoMic, IoSend } from "react-icons/io5"

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

/*   function useSpeechRecognition() {
    const recognition = new (window.SpeechRecognition || window.webkitSpeechRecognition)()
    recognition.onresult = async function (e) {
      const transcript = e.results[0][0].transcript

      setInputMessage(transcript)
    }

    recognition.start()
  } */

  return (
    <Flex align="center" gap="sm">
      <Textarea
        style={{ flexGrow: 1 }}
        autosize
        value={inputMessage}
        onChange={(e) => setInputMessage(e.target.value)}
        onKeyDown={onKeyDown}
      />
      <ActionIcon disabled>
        <IoMic />
      </ActionIcon>
      <ActionIcon onClick={onSendMessage}>
        <IoSend />
      </ActionIcon>
    </Flex>
  )
}
