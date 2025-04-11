export type Video = {
  id: string
  title: string
  description: string
  url: string
}

export type Playlist = {
  title: string
  videos: Video[]
}

export type VideoTimestamp = {
  timestampId: string
  videoId: string
  timestamp: string
  description: string
}
