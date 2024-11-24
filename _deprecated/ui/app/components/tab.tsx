import { Flex, Tabs, Text } from "@mantine/core"
import { useLocation, useNavigate } from "@remix-run/react"
import { IoChatbubblesOutline, IoAppsOutline, IoHome } from "react-icons/io5"

export default function Tab() {
  const navigate = useNavigate()
  const location = useLocation()

  const tabItems = [
    {
      name: "Home",
      icon: <IoHome size="20px" />,
      link: "/",
    },
    {
      name: "Chat",
      icon: <IoChatbubblesOutline size="20px" />,
      link: "/chat",
    },
    {
      name: "Commands",
      icon: <IoAppsOutline size="20px" />,
      link: "/commands",
    },
  ]

  let tabDefaultValue = tabItems[0].name
  for (const tab of tabItems) {
    if (location.pathname.includes(tab.link)) {
      tabDefaultValue = tab.name
    }
  }

  return (
    <Tabs defaultValue={tabDefaultValue} inverted>
      <Tabs.List pos="absolute" bottom="0" w="100%">
        {tabItems.map(({ name, icon, link }) => (
          <Tabs.Tab
            key={name}
            value={name}
            pb="4px"
            flex="1"
            onClick={() => navigate(link)}
          >
            <Flex direction="column" align="center">
              {icon}
              <Text size="xs">{name}</Text>
            </Flex>
          </Tabs.Tab>
        ))}
      </Tabs.List>
    </Tabs>
  )
}
