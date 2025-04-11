import { Card, CardSection, Grid, GridCol, Group, Text } from "@mantine/core"
import { Video } from "@/src/types"
import HlsPlayer from "@/src/components/HlsPlayer"

type Props = {
  videos: Video[]
}
export default function Videos({ videos }: Props) {
  return (
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
                <a href={`/videos/${id}`}>{title}</a>
              </Text>
            </Group>
            <Text size="sm" c="dimmed">
              {description}
            </Text>
          </Card>
        </GridCol>
      ))}
    </Grid>
  )
}
