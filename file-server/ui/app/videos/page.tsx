import {
  Anchor,
  Breadcrumbs,
  Card,
  CardSection,
  Grid,
  GridCol,
  Group,
  Stack,
  Text,
  Title,
} from "@mantine/core"
import Link from "next/link"
import { Playlist } from "@/app/types"
import HlsPlayer from "@/components/HlsPlayer"

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
          <Grid>
            {videos.map(({ id, title, description, url }) => (
              <GridCol key={id} span={{ base: 6, sm: 4, md: 3 }}>
                <Card shadow="sm" padding="lg" radius="md" withBorder w="100%">
                  {/* TODO: サムネは画面いっぱいに広げたい */}
                  <CardSection>
                    <HlsPlayer id={id} src={url} />
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
              </GridCol>
            ))}
          </Grid>
        </Stack>
      ))}
    </Stack>
  )
}
