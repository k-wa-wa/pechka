import { Center } from "@mantine/core"
import { Link } from "@remix-run/react"
import Tab from "~/components/tab"

export default function Index() {
  return (
    <>
      <Center h="100%">
        <Link to="/new-chat">→chat</Link>
      </Center>

      <Tab />
    </>
  )
}
