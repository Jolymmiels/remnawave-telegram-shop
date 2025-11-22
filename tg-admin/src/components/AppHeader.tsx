import React from 'react'
import { Group, Title, Box } from '@mantine/core'

interface AppHeaderProps {
  title: string
}

const AppHeader: React.FC<AppHeaderProps> = ({ title }) => {
  return (
    <Box 
      h="100%" 
      px="md"
      style={{
        paddingTop: 'max(var(--mantine-spacing-sm), var(--tg-viewport-safe-area-inset-top, 0px))'
      }}
    >
    </Box>
  )
}

export default AppHeader