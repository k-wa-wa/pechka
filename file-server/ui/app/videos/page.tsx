import {
  Anchor,
  Breadcrumbs,
  Card,
  CardSection,
  Flex,
  Group,
  Stack,
  Text,
  Title,
} from "@mantine/core"
import Player from "next-video/player"
import Link from "next/link"
import { Playlist } from "@/app/types"

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
          <Flex wrap="wrap" gap="sm" justify="center">
            {videos.map(({ id, title, description, url }) => (
              <Card
                key={id}
                shadow="sm"
                padding="lg"
                radius="md"
                withBorder
                maw="400px"
              >
                {/* TODO: サムネは画面いっぱいに広げたい */}
                <CardSection>
                  <Player src={url} />
                </CardSection>
                <Group mt="md">
                  <Text fw={500}>
                    <Link href={`/videos/${id}`}>{title}</Link>
                  </Text>
                </Group>
                <Text size="sm" c="dimmed">
                  {description}
                </Text>
              </Card>
            ))}
          </Flex>
        </Stack>
      ))}
    </Stack>
  )
}
