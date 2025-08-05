import { Playlist } from "@/src/types"
import { Anchor, Breadcrumbs, Pagination, Stack, Title } from "@mantine/core"
import { LoaderFunctionArgs } from "@remix-run/node"
import { useLoaderData } from "@remix-run/react"
import Videos from "@/src/components/Videos"
import axios from "axios"

const limit = 8

export const loader = async ({ params, request }: LoaderFunctionArgs) => {
  const playlistId = params.id || ""
  const url = new URL(request.url)

  const currentPageN = Number(url.searchParams.get("page")) || 1
  const offset = (currentPageN - 1) * limit

  const playlist: Playlist = await axios
    .get(
      `${process.env.API_URL}/api/playlists/${playlistId}?limit=${limit}&offset=${offset}`
    )
    .then((res) => res.data)
    .catch(e => console.log(e))
  return { ...playlist, currentPageN }
}

export default function PlaylistPage() {
  const {
    playlist: { title },
    videos,
    numVideos,
    currentPageN,
  } = useLoaderData<typeof loader>()

  return (
    <Stack>
      <Breadcrumbs>
        {[
          {
            title: "Top",
            href: "/",
          },
        ].map(({ title, href }) => (
          <Anchor key={title} href={href}>
            {title}
          </Anchor>
        ))}
      </Breadcrumbs>

      <Title order={2}>{title}</Title>
      <Videos videos={videos} />

      <Pagination
        total={Math.ceil(numVideos / limit)}
        defaultValue={currentPageN}
        getItemProps={(page) => ({
          component: "a",
          href: `?page=${page}`,
        })}
      />
    </Stack>
  )
}
