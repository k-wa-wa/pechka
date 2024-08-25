import { useState } from "react"
import {
  Page,
  Navbar,
  Messagebar,
  Messages,
  Message,
  Link,
  Icon,
  App,
} from "konsta/react"
import { LuArrowUpCircle } from "react-icons/lu"
import { VITE_API_BASE_URL } from "./constants"

import "./index.css"

type Message = {
  id: string
  role: "user" | "model"
  content: string
}

export default function App_() {
  const [inputMessage, setInputMessage] = useState("")
  const [messages, setMessages] = useState<Message[]>([])

  async function handleSendClick() {
    if (!inputMessage) return

    const message = inputMessage
    setInputMessage("")
    const res = await fetch(`${VITE_API_BASE_URL}/api/chat`, {
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
    const { body } = res
    if (!res.ok || !body) return

    let resMessage = ""
    const _messages = messages
    const reader = res.body?.getReader()
    const decoder = new TextDecoder("utf-8")
    if (!res.ok || !reader) return
    const read = async function () {
      const { done, value } = await reader.read()
      if (done) return reader.releaseLock()

      const chunk = decoder.decode(value, { stream: true })
      const res: {
        model: string
        message: {
          role: string
          content: string
        }
        done: boolean
      } = JSON.parse(chunk)
      resMessage += res.message.content
      setMessages([
        ..._messages,
        {
          id: "aaa",
          role: "user",
          content: message,
        },
        {
          id: "",
          role: "model",
          content: resMessage,
        },
      ])
      return read()
    }
    await read()
    reader.releaseLock()
  }

  return (
    <App theme="ios">
      <Page>
        <Navbar title="Pechka" />
        <Messages>
          {messages.map((message, index) => (
            <Message
              key={index}
              type={message.role === "user" ? "sent" : "received"}
              name={""}
              text={message.content}
              avatar={<></>}
            />
          ))}
        </Messages>
        <Messagebar
          placeholder="Message"
          value={inputMessage}
          onInput={(e) => setInputMessage(e.target.value)}
          right={
            <Link
              onClick={inputMessage ? handleSendClick : undefined}
              toolbar
              /* style={{
                opacity: inputOpacity,
                cursor: isClickable ? "pointer" : "default",
              }} */
            >
              <Icon ios={<LuArrowUpCircle className="w-7 h-7" />} />
            </Link>
          }
        />
      </Page>
    </App>
  )
}
