export type Video = {
  id: string
  title: string
  description: string
  url: string
}

export type Playlist = {
  playlist: {
    id: string
    title: string
  }
  videos: Video[]
  numVideos: number
}

export type VideoTimestamp = {
  timestampId: string
  videoId: string
  timestamp: string
  description: string
}
