import { Center, Tabs, Text } from "@mantine/core"
import { Link } from "@remix-run/react"
import { ReactNode } from "react"
import { IoChatbubblesOutline, IoAppsOutline } from "react-icons/io5"

function TabItem({ tabValue, icon }: { tabValue: string; icon: ReactNode }) {
  return (
    <Tabs.Tab value={tabValue} style={{ paddingBottom: "4px" }}>
      <span
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
        }}
      >
        {icon}
        <Text size="xs">{tabValue}</Text>
      </span>
    </Tabs.Tab>
  )
}

export default function Demo() {
  return (
    <>
      <Center h="100%">
        <Link to="/chat">→chat</Link>
      </Center>
      <Tabs defaultValue="chat" inverted>
        <Tabs.List
          style={{
            position: "absolute",
            bottom: 0,
            width: "100vw",
          }}
          grow
        >
          <TabItem
            tabValue="chat"
            icon={<IoChatbubblesOutline size="20px" />}
          />
          <TabItem tabValue="commands" icon={<IoAppsOutline size="20px" />} />
        </Tabs.List>
      </Tabs>
    </>
  )
}
