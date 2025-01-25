function pad(num: number) {
  return num < 10 ? "0" + num : num
}

export function timeToHHMMSS(time: number): string {
  const hours = Math.floor(time / 3600)
  const minutes = Math.floor((time % 3600) / 60)
  const secs = Math.floor(time % 60)

  return `${pad(hours)}:${pad(minutes)}:${pad(secs)}`
}
