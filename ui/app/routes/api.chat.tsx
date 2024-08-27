import { ActionFunction } from "@remix-run/node"

async function wait(second: number) {
  return new Promise((resolve) => setTimeout(resolve, 1000 * second))
}

type ReqBody = {
  messages: { content: string }[]
}
type ResBody = {
  message: { role: string; content: string }
}

export const action: ActionFunction = async ({ request }) => {
  return request
    .json()
    .then((body: ReqBody) => {
      const message = body.messages.map((message) => message.content).join("\n")

      const stream = new ReadableStream({
        start(controller) {
          const encoder = new TextEncoder()

          ;(async () => {
            for (const m of message) {
              const res: ResBody = {
                message: {
                  role: "model",
                  content: m,
                },
              }
              controller.enqueue(encoder.encode(`${JSON.stringify(res)}\n\n`))
              await wait(0.1)
            }

            controller.close()
          })()
        },
      })

      return new Response(stream, {
        headers: {
          "Content-Type": "application/x-ndjson",
        },
      })
    })
    .catch((e) => console.log(e))
}
