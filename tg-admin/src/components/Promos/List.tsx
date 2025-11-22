import React, { useEffect, useState } from 'react'
import {
  Box,
  Title,
  Text,
  Badge,
  Stack,
  Card,
  Group,
  ActionIcon,
  Switch,
  Button,
  Modal,
  Alert
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconTrash, IconCheck, IconX, IconInfoCircle } from '@tabler/icons-react'
import { usePromos, Promo } from '@/context/PromosContext'
import { useFormat } from '@/hooks/useFormat'
import { useTelegram } from '@/hooks/useTelegram'
import CreateForm from './CreateForm'
import { getTelegramSafeAreaStyles } from '../../lib/telegram-safe-area'

const PromosList: React.FC = () => {
  const { items, loading, error, initialized, load, update, delete: deletePromo } = usePromos()
  const { date, time } = useFormat()
  const { hapticFeedback } = useTelegram()
  const [deleteModal, setDeleteModal] = useState<{ open: boolean; promo: Promo | null }>({
    open: false,
    promo: null
  })

  useEffect(() => {
    // Only load if not already initialized to prevent duplicate requests
    if (!initialized && !loading) {
      load()
    }
  }, [initialized, loading, load])

  const handleToggleActive = async (id: number, currentActive: boolean) => {
    hapticFeedback.soft()
    try {
      await update(id, !currentActive)
      notifications.show({
        title: 'Успех',
        message: `Промокод ${currentActive ? 'деактивирован' : 'активирован'}`,
        color: 'green',
        icon: <IconCheck size={16} />
      })
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: error instanceof Error ? error.message : 'Ошибка при обновлении промокода',
        color: 'red',
        icon: <IconX size={16} />
      })
    }
  }

  const handleDelete = async () => {
    if (!deleteModal.promo) return
    
    hapticFeedback.soft()
    try {
      await deletePromo(deleteModal.promo.id)
      setDeleteModal({ open: false, promo: null })
      notifications.show({
        title: 'Успех',
        message: 'Промокод удален',
        color: 'green',
        icon: <IconCheck size={16} />
      })
    } catch (error) {
      notifications.show({
        title: 'Ошибка',
        message: error instanceof Error ? error.message : 'Ошибка при удалении',
        color: 'red',
        icon: <IconX size={16} />
      })
    }
  }

  const getStatusColor = (promo: Promo) => {
    if (!promo.active) return 'gray'
    if (promo.expires_at && new Date(promo.expires_at) < new Date()) return 'red'
    if (promo.max_uses && promo.used_count >= promo.max_uses) return 'orange'
    return 'green'
  }

  const getStatusText = (promo: Promo) => {
    if (!promo.active) return 'Неактивен'
    if (promo.expires_at && new Date(promo.expires_at) < new Date()) return 'Истек'
    if (promo.max_uses && promo.used_count >= promo.max_uses) return 'Достигнут лимит'
    return 'Активен'
  }

  const isPromoExpired = (promo: Promo) => {
    return promo.expires_at && new Date(promo.expires_at) < new Date()
  }

  const isPromoLimitReached = (promo: Promo) => {
    return promo.max_uses && promo.used_count >= promo.max_uses
  }

  if (loading) {
    return <Text>Loading...</Text>
  }

  if (error) {
    return (
      <Alert icon={<IconInfoCircle size={16} />} title="Ошибка" color="red">
        {error}
      </Alert>
    )
  }

  return (
    <Stack>
      <Box>
        <CreateForm />
      </Box>

      <Stack gap="md">
        {items.length === 0 ? (
          <Text c="dimmed" ta="center" py="xl">
            No promo codes found. Create one above to get started.
          </Text>
        ) : (
          items.map((promo) => (
            <Card key={promo.id} shadow="sm" padding="md" radius="md" withBorder>
              <Group justify="space-between" mb="md">
                <Group>
                  <Text size="lg" fw={600}>{promo.code}</Text>
                  <Badge color={getStatusColor(promo)} variant="light">
                    {getStatusText(promo)}
                  </Badge>
                </Group>
                
                <Group>
                  <Switch
                    checked={promo.active}
                    onChange={() => handleToggleActive(promo.id, promo.active)}
                    disabled={isPromoExpired(promo) || false}
                  />
                  <ActionIcon 
                    color="red" 
                    variant="light"
                    onClick={() => {
                      hapticFeedback.soft()
                      setDeleteModal({ open: true, promo })
                    }}
                  >
                    <IconTrash size={16} />
                  </ActionIcon>
                </Group>
              </Group>

              <Stack gap="xs">
                <Group>
                  <Text size="sm" c="dimmed">Бонусные дни:</Text>
                  <Text size="sm" fw={500}>+{promo.bonus_days} days</Text>
                </Group>
                
                <Group>
                  <Text size="sm" c="dimmed">Испольований:</Text>
                  <Text size="sm" fw={500}>
                    {promo.used_count}{promo.max_uses ? ` / ${promo.max_uses}` : ' (unlimited)'}
                  </Text>
                </Group>

                {promo.expires_at && (
                  <Group>
                    <Text size="sm" c="dimmed">Истекает:</Text>
                    <Text 
                      size="sm" 
                      fw={500}
                      c={isPromoExpired(promo) ? 'red' : undefined}
                    >
                      {date(promo.expires_at)} at {time(promo.expires_at)}
                    </Text>
                  </Group>
                )}

                <Group>
                  <Text size="sm" c="dimmed">Создано:</Text>
                  <Text size="sm">{date(promo.created_at)} at {time(promo.created_at)}</Text>
                </Group>
              </Stack>

              {(isPromoExpired(promo) || isPromoLimitReached(promo)) && (
                <Alert 
                  icon={<IconInfoCircle size={16} />} 
                  color={isPromoExpired(promo) ? 'red' : 'orange'}
                  mt="md"
                >
                  {isPromoExpired(promo) && 'Промокод истек.'}
                  {isPromoLimitReached(promo) && 'Достигнету лимит использований.'}
                </Alert>
              )}
            </Card>
          ))
        )}
      </Stack>

      <Modal
        opened={deleteModal.open}
        onClose={() => setDeleteModal({ open: false, promo: null })}
        title="Удалить промокод"
        centered
        styles={getTelegramSafeAreaStyles()}
      >
        <Stack>
          <Text>
            Вы уверены, что хотите удалить промокод? <strong>{deleteModal.promo?.code}</strong>?
            Это действие не может быть отменено.
          </Text>
          
          {deleteModal.promo && deleteModal.promo.used_count > 0 && (
            <Alert icon={<IconInfoCircle size={16} />} color="yellow">
              Этот промокод был использован {deleteModal.promo.used_count} раз.
            </Alert>
          )}

          <Group>
            <Button
              color="red"
              onClick={handleDelete}
            >
              Удалить
            </Button>
            <Button
              variant="subtle"
              onClick={() => {
                hapticFeedback.soft()
                setDeleteModal({ open: false, promo: null })
              }}
            >
              Отмена
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}

export default PromosList