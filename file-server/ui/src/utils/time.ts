function pad(num: number) {
  return num < 10 ? "0" + num : num
}

export function timeToHHMMSS(time: number): string {
  const hours = Math.floor(time / 3600)
  const minutes = Math.floor((time % 3600) / 60)
  const secs = Math.floor(time % 60)

  return `${pad(hours)}:${pad(minutes)}:${pad(secs)}`
}

export function HHMMSStoTime(HHMMSS: string): number {
  const [h, m, s] = HHMMSS.split(":")
  return (Number(h) || 0) * 3600 + (Number(m) || 0) * 60 + (Number(s) || 0)
}
