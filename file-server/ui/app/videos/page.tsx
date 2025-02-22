import {
  Anchor,
  Breadcrumbs,
  Stack,
  Title,
} from "@mantine/core"
import { Playlist } from "@/app/types"
import Videos from "@/components/Videos"

export default async function VideosPage() {
  const data = await fetch(`${process.env.API_URL}/api/playlists`, {
    next: { revalidate: 0 },
  })
  const playlists: Playlist[] = await data.json()

  return (
    <Stack gap="lg">
      <Breadcrumbs>
        {[
          {
            title: "Top",
            href: "/",
          },
          {
            title: "Videos",
            href: "/videos",
          },
        ].map(({ title, href }) => (
          <Anchor key={title} href={href}>
            {title}
          </Anchor>
        ))}
      </Breadcrumbs>

      {playlists.map(({ title, videos }) => (
        <Stack key={title}>
          <Title order={2}>{title}</Title>
          <Videos videos={videos} />
        </Stack>
      ))}
    </Stack>
  )
}
