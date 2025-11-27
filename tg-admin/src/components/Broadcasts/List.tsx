import React, { useEffect, useState } from 'react'
import { Box, Text, Badge, Stack, Card, Group, Divider, Progress, Tooltip, ActionIcon } from '@mantine/core'
import { IconCheck, IconClock, IconLoader, IconX, IconTrash } from '@tabler/icons-react'
import { useBroadcasts } from '@/context/BroadcastsContext'
import { useFormat } from '@/hooks/useFormat'
import CreateForm from './CreateForm'

const BroadcastsList: React.FC = () => {
  const { items, loading, error, initialized, load, refreshActive, remove } = useBroadcasts()
  const { date, time } = useFormat()
  const [deletingId, setDeletingId] = useState<number | null>(null)

  const handleDelete = async (id: number) => {
    if (deletingId) return
    setDeletingId(id)
    try {
      await remove(id)
    } finally {
      setDeletingId(null)
    }
  }

  useEffect(() => {
    if (!initialized && !loading) {
      load()
    }
  }, [initialized, loading, load])

  // Auto-refresh only active broadcasts without full reload
  useEffect(() => {
    const hasInProgress = items.some(b => b.Status === 'in_progress')
    if (!hasInProgress) return

    const interval = setInterval(() => {
      refreshActive()
    }, 2000)

    return () => clearInterval(interval)
  }, [items, refreshActive])

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

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'pending':
        return <Badge color="gray" variant="light" leftSection={<IconClock size={12} />}>Ожидает</Badge>
      case 'in_progress':
        return <Badge color="blue" variant="light" leftSection={<IconLoader size={12} className="spin" />}>Отправка...</Badge>
      case 'completed':
        return <Badge color="green" variant="light" leftSection={<IconCheck size={12} />}>Завершено</Badge>
      case 'failed':
        return <Badge color="red" variant="light" leftSection={<IconX size={12} />}>Ошибка</Badge>
      default:
        return <Badge color="gray" variant="light">{status}</Badge>
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
        {items.map((broadcast) => {
          const processed = broadcast.SentCount + broadcast.FailedCount + broadcast.BlockedCount
          const progress = broadcast.TotalCount > 0 ? (processed / broadcast.TotalCount) * 100 : 0

          return (
            <Card key={broadcast.ID} shadow="sm" padding="md" radius="md" withBorder>
              {/* Header with ID, Type, Status, Date, Language */}
              <Group justify="space-between" align="flex-start" mb="sm">
                <Group gap="md">
                  <Text fw={600} size="sm" c="dimmed">
                    #{broadcast.ID}
                  </Text>
                  <Badge color={getTypeColor(broadcast.Type)} variant="light">
                    {getTypeName(broadcast.Type)}
                  </Badge>
                  {broadcast.Language && (
                    <Badge color="gray" variant="outline">
                      {broadcast.Language.toUpperCase()}
                    </Badge>
                  )}
                  {getStatusBadge(broadcast.Status)}
                </Group>
                <Group gap="sm">
                  <Text size="sm" c="dimmed">
                    {date(broadcast.CreatedAt)} в {time(broadcast.CreatedAt)}
                  </Text>
                  <Tooltip label="Удалить">
                    <ActionIcon 
                      variant="subtle" 
                      color="red" 
                      size="sm"
                      loading={deletingId === broadcast.ID}
                      onClick={() => handleDelete(broadcast.ID)}
                      disabled={broadcast.Status === 'in_progress'}
                    >
                      <IconTrash size={16} />
                    </ActionIcon>
                  </Tooltip>
                </Group>
              </Group>

              {/* Progress bar for non-pending broadcasts */}
              {broadcast.Status !== 'pending' && (
                <Box mb="sm">
                  <Group justify="space-between" mb={4}>
                    <Text size="xs" c="dimmed">
                      Прогресс: {processed} / {broadcast.TotalCount}
                    </Text>
                    <Text size="xs" c="dimmed">
                      {progress.toFixed(1)}%
                    </Text>
                  </Group>
                  <Progress 
                    value={progress} 
                    size="sm" 
                    color={broadcast.Status === 'completed' ? 'green' : broadcast.Status === 'failed' ? 'red' : 'blue'}
                    animated={broadcast.Status === 'in_progress'}
                  />
                  <Group gap="lg" mt={4}>
                    <Tooltip label="Успешно отправлено">
                      <Text size="xs" c="green">✓ {broadcast.SentCount}</Text>
                    </Tooltip>
                    <Tooltip label="Ошибки отправки">
                      <Text size="xs" c="red">✗ {broadcast.FailedCount}</Text>
                    </Tooltip>
                    <Tooltip label="Заблокировали бота">
                      <Text size="xs" c="orange">⊘ {broadcast.BlockedCount}</Text>
                    </Tooltip>
                  </Group>
                </Box>
              )}

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
          )
        })}

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