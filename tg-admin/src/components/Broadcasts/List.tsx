import React, { useEffect } from 'react'
import { Box, Title, Text, Badge, Stack, Card, Group, Divider } from '@mantine/core'
import { useBroadcasts } from '@/context/BroadcastsContext'
import { useFormat } from '@/hooks/useFormat'
import CreateForm from './CreateForm'

const BroadcastsList: React.FC = () => {
  const { items, loading, error, initialized, load } = useBroadcasts()
  const { date, time } = useFormat()

  useEffect(() => {
    // Only load if not already initialized to prevent duplicate requests
    if (!initialized && !loading) {
      load()
    }
  }, [initialized, loading, load])

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'all': return 'blue'
      case 'active': return 'green'
      case 'inactive': return 'red'
      default: return 'gray'
    }
  }

  const getTypeName = (type: string) => {
    switch (type) {
      case 'all': return 'Всем'
      case 'active': return 'Активным'
      case 'inactive': return 'Неактивным'
      default: return type
    }
  }

  if (loading) {
    return <Text>Загрузка...</Text>
  }

  if (error) {
    return <Text c="red">Ошибка: {error}</Text>
  }

  return (
    <Stack>
      <Box>
        {/*<Title order={2} mb="md">Вещания</Title>*/}
        <CreateForm />
      </Box>

      <Stack gap="md">
        {items.map((broadcast) => (
          <Card key={broadcast.ID} shadow="sm" padding="md" radius="md" withBorder>
            {/* Header with ID, Type, Date, Language */}
            <Group justify="space-between" align="flex-start" mb="sm">
              <Group gap="md">
                <Text fw={600} size="sm" c="dimmed">
                  ID: {broadcast.ID}
                </Text>
                <Badge color={getTypeColor(broadcast.Type)} variant="light">
                  {getTypeName(broadcast.Type)}
                </Badge>
                {broadcast.Language && (
                  <Badge color="gray" variant="outline">
                    {broadcast.Language.toUpperCase()}
                  </Badge>
                )}
              </Group>
              <Text size="sm" c="dimmed">
                {date(broadcast.CreatedAt)} в {time(broadcast.CreatedAt)}
              </Text>
            </Group>

            <Divider my="sm" />

            {/* Body - Broadcast content */}
            <Box>
              <Text size="sm" mb="xs" fw={500} c="dimmed">
                Содержание:
              </Text>
              <Text 
                style={{ 
                  whiteSpace: 'pre-wrap', 
                  wordBreak: 'break-word',
                  lineHeight: 1.5 
                }}
              >
                {broadcast.Content}
              </Text>
            </Box>
          </Card>
        ))}

        {items.length === 0 && (
          <Card shadow="sm" padding="xl" radius="md" withBorder>
            <Text ta="center" c="dimmed">
              Нет вещаний
            </Text>
          </Card>
        )}
      </Stack>
    </Stack>
  )
}

export default BroadcastsList