import { Playlist } from "@/src/types"
import { Anchor, Breadcrumbs, Pagination, Stack, Title } from "@mantine/core"
import { LoaderFunctionArgs } from "@remix-run/node"
import { useLoaderData } from "@remix-run/react"
import Videos from "@/src/components/Videos"

const limit = 8

export const loader = async ({ params, request }: LoaderFunctionArgs) => {
  const playlistId = params.id || ""
  const url = new URL(request.url)

  const currentPageN = Number(url.searchParams.get("page")) || 1
  const offset = (currentPageN - 1) * limit

  const data = await fetch(
    `${process.env.API_URL}/api/playlists/${playlistId}?limit=${limit}&offset=${offset}`
  )
  const playlist: Playlist = await data.json()
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
