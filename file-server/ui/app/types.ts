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