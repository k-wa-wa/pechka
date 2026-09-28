import { networkInterfaces } from 'node:os'
import qrcodeTerminal from 'qrcode-terminal'

/** LAN からスマホ等でアクセスするための IPv4 アドレス（見つからなければ null）。 */
function getLanIp() {
  const addrs = []
  for (const iface of Object.values(networkInterfaces())) {
    for (const addr of iface ?? []) {
      if (addr.family === 'IPv4' && !addr.internal) {
        addrs.push(addr.address)
      }
    }
  }
  // 一般的なプライベート LAN レンジ (192.168.x.x, 10.x.x.x, 172.16-31.x.x) を優先
  const privateIp = addrs.find((a) =>
    /^192\.168\./.test(a) || /^10\./.test(a) || /^172\.(1[6-9]|2\d|3[01])\./.test(a)
  )
  return privateIp || addrs[0] || null
}

const port = process.env.PORT || 3000
const lanIp = getLanIp()
const localUrl = `http://localhost:${port}`
const lanUrl = lanIp ? `http://${lanIp}:${port}` : null

console.log(`
┌────────────────────────────────────────────────────────────┐
│  pechka (Local Dev Server)                                 │
│                                                            │
│  • Local:         ${localUrl.padEnd(41)}│
${lanUrl ? `│  • LAN URL:       ${lanUrl.padEnd(41)}│\n` : ''}│                                                            │
│  スマホなどの端末から同じ Wi-Fi 経由でアクセス可能         │
└────────────────────────────────────────────────────────────┘
`)

if (lanUrl) {
  console.log('スマホでスキャンしてアクセス:')
  qrcodeTerminal.generate(lanUrl, { small: true }, (qr) => console.log(qr))
  console.log('')
}
