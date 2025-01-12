'use server';

import Player from "next-video/player"

type HlsResources = {
  path: string
}

export default async function Home() {
  const data = await fetch(`${process.env.API_URL}/api/hls/hls-resources`)
  const hlsResources: HlsResources[] = await data.json()
  console.log(hlsResources)

  return (
    <div>
      {hlsResources.map(({ path }) => (
        <div key={path}>
          {path}
          <Player src={path} />
        </div>
      ))}
    </div>
  )
}
