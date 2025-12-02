import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Stack,
  Paper,
  Table,
  Group,
  Badge,
  Text,
  Loader,
  Alert,
  Card,
  SimpleGrid,
  ScrollArea,
  ActionIcon,
  Tooltip,
  Button,
  Pagination,
} from '@mantine/core'
import { IconArrowLeft, IconTicket, IconEye } from '@tabler/icons-react'
import { http } from '@/lib/http'
import { backButton } from '@telegram-apps/sdk'
import { useMediaQuery } from '@mantine/hooks'
import { useFormat } from '@/hooks/useFormat'

interface Promo {
  id: number
  code: string
  bonus_days: number
  max_uses: number | null
  used_count: number
  expires_at: string | null
  active: boolean
  created_at: string
}

interface PromoUsage {
  id: number
  promo_id: number
  telegram_id: number
  tg_username?: string | null
  tg_first_name?: string | null
  tg_last_name?: string | null
  used_at: string
}

const PromoDetailsPage: React.FC = () => {
  const { promoId } = useParams<{ promoId: string }>()
  const navigate = useNavigate()
  const isMobile = useMediaQuery('(max-width: 768px)')
  const { date, time } = useFormat()
  const [promo, setPromo] = useState<Promo | null>(null)
  const [usages, setUsages] = useState<PromoUsage[]>([])
  const [totalUsages, setTotalUsages] = useState(0)
  const [loading, setLoading] = useState(true)
  const [usagesLoading, setUsagesLoading] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const itemsPerPage = 50

  const goBack = () => {
    navigate('/promos')
  }

  useEffect(() => {
    if (backButton.isSupported()) {
      if (!backButton.isMounted()) {
        backButton.mount()
      }
      backButton.show()
      backButton.onClick(goBack)
    }

    return () => {
      if (backButton.isSupported() && backButton.isMounted()) {
        backButton.offClick(goBack)
        backButton.hide()
      }
    }
  }, [])

  useEffect(() => {
    const fetchPromo = async () => {
      if (!promoId) return
      try {
        setLoading(true)
        const promos: Promo[] = await http.get('/api/promos')
        const foundPromo = promos.find(p => p.id === parseInt(promoId))
        if (foundPromo) {
          setPromo(foundPromo)
        }
      } catch (error) {
        console.error('Failed to fetch promo:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchPromo()
  }, [promoId])

  useEffect(() => {
    const fetchUsages = async () => {
      if (!promoId) return
      try {
        setUsagesLoading(true)
        const result: { usages: PromoUsage[], total: number } = await http.get(
          `/api/promos/${promoId}/usages?page=${currentPage}&limit=${itemsPerPage}`
        )
        setUsages(result.usages || [])
        setTotalUsages(result.total)
      } catch (error) {
        console.error('Failed to fetch usages:', error)
        setUsages([])
        setTotalUsages(0)
      } finally {
        setUsagesLoading(false)
      }
    }
    fetchUsages()
  }, [promoId, currentPage])

  if (loading) {
    return (
      <Stack align="center" justify="center" h={200}>
        <Loader size="lg" />
      </Stack>
    )
  }

  if (!promo) {
    return (
      <Stack>
        <Alert color="red">Промокод не найден</Alert>
        <Button variant="subtle" leftSection={<IconArrowLeft size={16} />} onClick={goBack}>
          Назад к промокодам
        </Button>
      </Stack>
    )
  }

  const getStatusColor = () => {
    if (!promo.active) return 'gray'
    if (promo.expires_at && new Date(promo.expires_at) < new Date()) return 'red'
    if (promo.max_uses && promo.used_count >= promo.max_uses) return 'orange'
    return 'green'
  }

  const getStatusText = () => {
    if (!promo.active) return 'Неактивен'
    if (promo.expires_at && new Date(promo.expires_at) < new Date()) return 'Истек'
    if (promo.max_uses && promo.used_count >= promo.max_uses) return 'Лимит достигнут'
    return 'Активен'
  }

  return (
    <Stack gap="md">
      <Paper p="md" radius="md">
        <Group justify="space-between" mb="md">
          <Group gap="xs">
            <IconTicket size={24} />
            <Text size="xl" fw={700}>{promo.code}</Text>
          </Group>
          <Badge color={getStatusColor()} size="lg">{getStatusText()}</Badge>
        </Group>

        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="xs">
          <Group gap={4}>
            <Text size="sm" c="dimmed">Бонусные дни:</Text>
            <Text size="sm" fw={500}>+{promo.bonus_days}</Text>
          </Group>
          <Group gap={4}>
            <Text size="sm" c="dimmed">Использований:</Text>
            <Text size="sm" fw={500}>
              {promo.used_count}{promo.max_uses ? ` / ${promo.max_uses}` : ' (без лимита)'}
            </Text>
          </Group>
          {promo.expires_at && (
            <Group gap={4}>
              <Text size="sm" c="dimmed">Истекает:</Text>
              <Text size="sm" fw={500}>{date(promo.expires_at)} {time(promo.expires_at)}</Text>
            </Group>
          )}
          <Group gap={4}>
            <Text size="sm" c="dimmed">Создан:</Text>
            <Text size="sm">{date(promo.created_at)} {time(promo.created_at)}</Text>
          </Group>
        </SimpleGrid>
      </Paper>

      <Paper p="md" radius="md">
        <Text size="lg" fw={600} mb="md">История использований</Text>

        {usagesLoading ? (
          <Stack align="center" py="xl">
            <Loader size="sm" />
          </Stack>
        ) : usages.length === 0 ? (
          <Alert color="gray">Промокод ещё не использовался</Alert>
        ) : isMobile ? (
          <Stack gap="sm">
            {usages.map((usage) => (
              <Card key={usage.id} padding="sm" withBorder>
                <Group justify="space-between" align="flex-start">
                  <Stack gap={2}>
                    <Text size="sm" fw={600}>{usage.telegram_id}</Text>
                    {(usage.tg_username || usage.tg_first_name || usage.tg_last_name) && (
                      <Text size="xs" c="dimmed">
                        {usage.tg_username && `@${usage.tg_username}`}
                        {usage.tg_username && (usage.tg_first_name || usage.tg_last_name) && ' · '}
                        {[usage.tg_first_name, usage.tg_last_name].filter(Boolean).join(' ')}
                      </Text>
                    )}
                    <Text size="xs" c="dimmed">{date(usage.used_at)} {time(usage.used_at)}</Text>
                  </Stack>
                  <Tooltip label="Подробнее">
                    <ActionIcon size="sm" variant="subtle" onClick={() => navigate(`/user/${usage.telegram_id}`)}>
                      <IconEye size={16} />
                    </ActionIcon>
                  </Tooltip>
                </Group>
              </Card>
            ))}
            {Math.ceil(totalUsages / itemsPerPage) > 1 && (
              <Group justify="center" mt="md">
                <Pagination value={currentPage} onChange={setCurrentPage} total={Math.ceil(totalUsages / itemsPerPage)} size="sm" />
              </Group>
            )}
          </Stack>
        ) : (
          <>
            <ScrollArea>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Пользователь</Table.Th>
                    <Table.Th>Дата использования</Table.Th>
                    <Table.Th w={40}></Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {usages.map((usage) => (
                    <Table.Tr key={usage.id}>
                      <Table.Td>
                        <Stack gap={2}>
                          <Text size="sm" fw={500}>{usage.telegram_id}</Text>
                          {(usage.tg_username || usage.tg_first_name || usage.tg_last_name) && (
                            <Text size="xs" c="dimmed">
                              {usage.tg_username && `@${usage.tg_username}`}
                              {usage.tg_username && (usage.tg_first_name || usage.tg_last_name) && ' · '}
                              {[usage.tg_first_name, usage.tg_last_name].filter(Boolean).join(' ')}
                            </Text>
                          )}
                        </Stack>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm">{date(usage.used_at)} {time(usage.used_at)}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Tooltip label="Подробнее">
                          <ActionIcon size="sm" variant="subtle" onClick={() => navigate(`/user/${usage.telegram_id}`)}>
                            <IconEye size={16} />
                          </ActionIcon>
                        </Tooltip>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </ScrollArea>
            {Math.ceil(totalUsages / itemsPerPage) > 1 && (
              <Group justify="center" mt="md">
                <Pagination value={currentPage} onChange={setCurrentPage} total={Math.ceil(totalUsages / itemsPerPage)} size="sm" />
              </Group>
            )}
          </>
        )}
      </Paper>
    </Stack>
  )
}

export default PromoDetailsPage
