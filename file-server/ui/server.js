import { createRequestHandler } from "@remix-run/express"
import morgan from "morgan"
import express from "express"
const app = express()

morgan.token('jsonLogFormat', (req, res) => {
  return JSON.stringify({
    timestamp: new Date().toISOString(),
    remote_ip: req.ip,
    request_id: req.get('x-request-id'),
    host: req.get('host'),
    request: {
      method: req.method,
      path: req.originalUrl,
      protocol: req.protocol,
    },
    status: res.statusCode,
    bytes: res.get('Content-Length'),
    referer: req.get('referer'),
    user_agent: req.get('user-agent'),
    request_time_ms: morgan['response-time'](req, res),
  });
});
app.use(morgan(':jsonLogFormat'));

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
