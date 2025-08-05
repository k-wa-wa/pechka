import { Video, VideoTimestamp } from "@/src/types"
import { Stack, Breadcrumbs, Anchor } from "@mantine/core"
import VideoView from "@/src/components/VideoView"
import { LoaderFunction, LoaderFunctionArgs } from "@remix-run/node"
import { useLoaderData } from "@remix-run/react"
import axios from "axios"

async function fetchVideoData(videoId: string): Promise<Video> {
  return await axios
    .get(`${process.env.API_URL}/api/videos/${videoId}`)
    .then((res) => res.data)
    .catch(e => console.log(e))
}

async function fetchVideoTimestamps(
  videoId: string
): Promise<VideoTimestamp[]> {
  return await axios
    .get(`${process.env.API_URL}/api/videos/${videoId}/timestamps`)
    .then((res) => res.data)
    .catch(e => console.log(e))
}

export const loader = async ({ params }: LoaderFunctionArgs) => {
  const videoId = params.id || ""
  const [video, timestamps] = await Promise.all([
    fetchVideoData(videoId),
    fetchVideoTimestamps(videoId),
  ])
  return { video, timestamps }
}

export default function VideoPage() {
  const { video, timestamps } = useLoaderData<typeof loader>()

  return (
    <Stack>
      <Breadcrumbs>
        {[
          {
            title: "Top",
            href: "/",
          },
          {
            title: video.title,
            href: `/videos/${video.id}`,
          },
        ].map(({ title, href }) => (
          <Anchor key={title} href={href}>
            {title}
          </Anchor>
        ))}
      </Breadcrumbs>

      <VideoView video={video} timestamps={timestamps} />
    </Stack>
  )
}
