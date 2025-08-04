import { createRequestHandler } from "@remix-run/express"
import morgan from "morgan"
import express from "express"
const app = express()

const combinedFormat = `:remote-addr - :remote-user [:date[clf]] ":method :url HTTP/:http-version" :status :res[content-length] ":referrer" ":user-agent"`
const logFormat = `${combinedFormat} ":req[x-request-id]"`
app.use(morgan(logFormat))

app.use(
  "/assets",
  express.static("build/client/assets")
)

app.all(
  "*",
  createRequestHandler({
    build: await import("./build/server/index.js"),
    mode: process.env.NODE_ENV,
  })
)

const port = process.env.PORT || 3000

app.listen(port, () => {
  console.log(`Express server listening on port ${port}`)
})
