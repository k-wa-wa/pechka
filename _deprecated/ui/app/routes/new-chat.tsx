import { Container, Flex, Text, Center, Burger } from "@mantine/core"
import ChatInput from "../components/chatInput"
import { useState } from "react"
import { v4 as uuidv4 } from "uuid"
import { Message } from "~/types"
import ChatMessages from "~/components/chatMessages"
import { IoChevronBack } from "react-icons/io5"
import { useNavigate } from "@remix-run/react"

function generateId() {
  return uuidv4()
}

export default function ChatPage() {
  const [inputMessage, setInputMessage] = useState("")
  const [messages, setMessages] = useState<Message[]>([])

  const navigate = useNavigate()

  async function onSendMessage() {
    const message = inputMessage
    if (!message) return
    setInputMessage(() => "")
    setMessages((prev) => [
      ...prev,
      {
        _id: generateId(),
        role: "user",
        content: message,
      },
      {
        _id: generateId(),
        role: "model",
        content: "",
      },
    ])

    const res = await fetch(`/api/chat`, {
      method: "POST",
      body: JSON.stringify({
        model: "",
        messages: [
          {
            role: "user",
            content: message,
          },
        ],
      }),
    })
    const reader = res.body?.getReader()
    if (!res.ok || !reader) return

    const decoder = new TextDecoder("utf-8")
    const read = async function () {
      const { done, value } = await reader.read()
      if (done) return reader.releaseLock()

      const chunk = decoder.decode(value, { stream: true })
      for (const c of chunk.split("\n\n")) {
        if (!c.startsWith("{")) continue
        const res: {
          model: string
          message: {
            role: string
            content: string
          }
          done: boolean
        } = JSON.parse(c)
        setMessages((prev) => [
          ...prev.slice(0, -1),
          {
            ...prev[prev.length - 1],
            content:
              prev[prev.length - 1].content + (res?.message?.content || ""),
          },
        ])
      }

      return read()
    }
    await read()
    reader.releaseLock()
  }

  return (
    <Flex direction="column" px="xs" gap="sm" h="100%">
      <Container
        w="100%"
        h="40px"
        style={{ position: "sticky", top: 0, zIndex: 5 }}
        bg="white"
      >
        <IoChevronBack
          size="20px"
          style={{
            position: "absolute",
            left: 0,
            top: 0,
            bottom: 0,
            margin: "auto",
          }}
          cursor="pointer"
          onClick={() => navigate("/chat")}
        />

        <Center h="100%">
          <Text fw={700} size="lg">
            Pechka
          </Text>
        </Center>
        <Burger
          style={{
            position: "absolute",
            right: 0,
            top: 0,
            bottom: 0,
            margin: "auto",
          }}
        />
      </Container>

      <Container w="100%" p="0" style={{ flexGrow: 1 }}>
        <ChatMessages messages={messages} />
      </Container>

      <Container
        w="100%"
        p="0"
        pb="xs"
        style={{ position: "sticky", bottom: 0 }}
        bg="white"
      >
        <ChatInput
          inputMessage={inputMessage}
          setInputMessage={setInputMessage}
          onSendMessage={onSendMessage}
        />
      </Container>
    </Flex>
  )
}
