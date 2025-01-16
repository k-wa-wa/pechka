import { Center, Paper, Title } from "@mantine/core"

export default function Header() {
  return (
    <Paper shadow="sm" py="4px">
      <Center>
        <Title fw={500} size="28px">
          HLS Video Server
        </Title>
      </Center>
    </Paper>
  )
}
