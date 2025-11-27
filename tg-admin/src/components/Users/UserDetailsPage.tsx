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
  Box,
  CopyButton,
  ActionIcon,
  Tooltip,
  ScrollArea,
  Modal,
  Button,
} from '@mantine/core'
import {
  IconArrowLeft,
  IconCoin,
  IconUsers,
  IconCopy,
  IconCheck,
  IconExternalLink,
  IconDeviceMobile,
  IconTrash,
  IconUserX,
} from '@tabler/icons-react'
import { http } from '../../lib/http'
import { backButton } from '@telegram-apps/sdk'
import { notifications } from '@mantine/notifications'

interface User {
  id: number
  telegram_id: number
  expire_at?: string | null
  created_at: string
  subscription_link?: string | null
  language: string
  is_blocked: boolean
  payments_count: number
  referrals_count: number
  total_spent: number
}

interface Payment {
  id: number
  amount: number
  customer_id: number
  created_at: string
  month: number
  paid_at?: string | null
  currency: string
  expire_at?: string | null
  status: string
  invoice_type: string
  crypto_invoice_id?: number | null
  crypto_invoice_link?: string | null
  yookasa_url?: string | null
  yookasa_id?: string | null
}

interface Device {
  hwid: string
  user_uuid: string
  platform: string
  os_version: string
  device_model: string
  created_at: string
  updated_at: string
}

const UserDetailsPage: React.FC = () => {
  const { telegramId } = useParams<{ telegramId: string }>()
  const navigate = useNavigate()
  const [user, setUser] = useState<User | null>(null)
  const [payments, setPayments] = useState<Payment[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [loading, setLoading] = useState(true)
  const [paymentsLoading, setPaymentsLoading] = useState(true)
  const [devicesLoading, setDevicesLoading] = useState(true)
  const [deleteModalOpened, setDeleteModalOpened] = useState(false)
  const [deviceToDelete, setDeviceToDelete] = useState<Device | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [revokeModalOpened, setRevokeModalOpened] = useState(false)
  const [revoking, setRevoking] = useState(false)

  const goBack = () => {
    navigate('/user-management')
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
    const fetchData = async () => {
      if (!telegramId) return

      try {
        setLoading(true)
        const response: { users: User[], total: number } = await http.get(`/api/users/search?q=${telegramId}&limit=1`)
        if (response.users && response.users.length > 0) {
          setUser(response.users[0])
        }
      } catch (error) {
        console.error('Failed to fetch user:', error)
      } finally {
        setLoading(false)
      }

      try {
        setPaymentsLoading(true)
        const paymentsData: Payment[] = await http.get(`/api/users/${telegramId}/payments`)
        setPayments(paymentsData)
      } catch (error) {
        console.error('Failed to fetch payments:', error)
        setPayments([])
      } finally {
        setPaymentsLoading(false)
      }

      try {
        setDevicesLoading(true)
        const devicesData: Device[] = await http.get(`/api/users/${telegramId}/devices`)
        setDevices(devicesData)
      } catch (error) {
        console.error('Failed to fetch devices:', error)
        setDevices([])
      } finally {
        setDevicesLoading(false)
      }
    }

    fetchData()
  }, [telegramId])

  const maskHwid = (hwid: string) => {
    if (hwid.length <= 8) return hwid
    return hwid.substring(0, 4) + '***' + hwid.substring(hwid.length - 4)
  }

  const handleDeleteDevice = async () => {
    if (!deviceToDelete || !telegramId) return
    
    setDeleting(true)
    try {
      await http.delete(`/api/users/${telegramId}/devices/${deviceToDelete.hwid}`)
      setDevices(devices.filter(d => d.hwid !== deviceToDelete.hwid))
      notifications.show({
        title: 'Успешно',
        message: 'Устройство удалено',
        color: 'green',
      })
    } catch (error) {
      console.error('Failed to delete device:', error)
      notifications.show({
        title: 'Ошибка',
        message: 'Не удалось удалить устройство',
        color: 'red',
      })
    } finally {
      setDeleting(false)
      setDeleteModalOpened(false)
      setDeviceToDelete(null)
    }
  }

  const openDeleteModal = (device: Device) => {
    setDeviceToDelete(device)
    setDeleteModalOpened(true)
  }

  const handleRevokeSubscription = async () => {
    if (!telegramId) return
    
    setRevoking(true)
    try {
      await http.post(`/api/users/${telegramId}/revoke-subscription`, {})
      setUser(prev => prev ? { ...prev, expire_at: null, subscription_link: null } : null)
      notifications.show({
        title: 'Успешно',
        message: 'Подписка отозвана',
        color: 'green',
      })
    } catch (error) {
      console.error('Failed to revoke subscription:', error)
      notifications.show({
        title: 'Ошибка',
        message: 'Не удалось отозвать подписку',
        color: 'red',
      })
    } finally {
      setRevoking(false)
      setRevokeModalOpened(false)
    }
  }

  const formatDate = (dateString?: string | null) => {
    if (!dateString) return '-'
    return new Date(dateString).toLocaleString('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getStatusBadge = (user: User) => {
    if (!user.expire_at) {
      return <Badge color="gray" size="sm">Без подписки</Badge>
    }
    const expireDate = new Date(user.expire_at)
    const now = new Date()
    if (expireDate < now) {
      return <Badge color="red" size="sm">Истекла</Badge>
    }
    return <Badge color="green" size="sm">Активна</Badge>
  }

  const getPaymentStatusBadge = (status: string) => {
    switch (status) {
      case 'paid':
        return <Badge color="green" size="xs">Оплачено</Badge>
      case 'pending':
        return <Badge color="yellow" size="xs">Ожидание</Badge>
      case 'cancelled':
        return <Badge color="red" size="xs">Отменен</Badge>
      default:
        return <Badge color="gray" size="xs">{status}</Badge>
    }
  }

  const getInvoiceTypeBadge = (type: string) => {
    switch (type) {
      case 'yookasa':
        return <Badge variant="light" color="violet" size="xs">YooKassa</Badge>
      case 'crypto':
        return <Badge variant="light" color="blue" size="xs">Crypto</Badge>
      case 'tribute':
        return <Badge variant="light" color="orange" size="xs">Tribute</Badge>
      default:
        return <Badge variant="light" color="gray" size="xs">{type}</Badge>
    }
  }

  if (loading) {
    return (
      <Stack align="center" justify="center" h={300}>
        <Loader size="lg" />
      </Stack>
    )
  }

  if (!user) {
    return (
      <Alert color="red" title="Ошибка">
        Пользователь не найден
      </Alert>
    )
  }

  return (
    <Stack gap="md">
      <Paper p="md" shadow="sm">
        <Group justify="space-between" mb="md">
          <Group gap="xs">
            <Text size="xl" fw={700}>ID: {user.telegram_id}</Text>
            {user.is_blocked && <Badge color="red">Заблокирован</Badge>}
          </Group>
          <Group gap="xs">
            {getStatusBadge(user)}
            {user.expire_at && new Date(user.expire_at) > new Date() && (
              <Tooltip label="Отозвать подписку">
                <ActionIcon
                  size="sm"
                  variant="subtle"
                  color="red"
                  onClick={() => setRevokeModalOpened(true)}
                >
                  <IconUserX size={16} />
                </ActionIcon>
              </Tooltip>
            )}
          </Group>
        </Group>

        <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
          <Card padding="sm" withBorder>
            <Text size="xs" c="dimmed" mb={4}>Регистрация</Text>
            <Text size="sm" fw={500}>{formatDate(user.created_at)}</Text>
          </Card>

          <Card padding="sm" withBorder>
            <Text size="xs" c="dimmed" mb={4}>Подписка до</Text>
            <Text size="sm" fw={500}>{formatDate(user.expire_at)}</Text>
          </Card>

          <Card padding="sm" withBorder>
            <Text size="xs" c="dimmed" mb={4}>Язык</Text>
            <Badge variant="light" size="sm">{(user.language || 'EN').toUpperCase()}</Badge>
          </Card>

          <Card padding="sm" withBorder>
            <Text size="xs" c="dimmed" mb={4}>Потрачено</Text>
            <Text size="sm" fw={700} c="green">{(user.total_spent ?? 0).toFixed(2)} ₽</Text>
          </Card>
        </SimpleGrid>

        <SimpleGrid cols={2} spacing="md" mt="md">
          <Card padding="sm" withBorder>
            <Group gap={4} mb={4}>
              <IconCoin size={14} />
              <Text size="xs" c="dimmed">Платежей</Text>
            </Group>
            <Text size="lg" fw={700}>{user.payments_count ?? 0}</Text>
          </Card>

          <Card padding="sm" withBorder>
            <Group gap={4} mb={4}>
              <IconUsers size={14} />
              <Text size="xs" c="dimmed">Рефералов</Text>
            </Group>
            <Text size="lg" fw={700}>{user.referrals_count ?? 0}</Text>
          </Card>
        </SimpleGrid>
      </Paper>

      <Paper p="md" shadow="sm">
        <Text size="lg" fw={600} mb="md">История платежей</Text>

        {paymentsLoading ? (
          <Stack align="center" py="xl">
            <Loader size="sm" />
          </Stack>
        ) : payments.length === 0 ? (
          <Alert color="gray">Платежи не найдены</Alert>
        ) : (
          <ScrollArea>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Дата</Table.Th>
                  <Table.Th>Сумма</Table.Th>
                  <Table.Th>Тип</Table.Th>
                  <Table.Th>Статус</Table.Th>
                  <Table.Th>Месяцев</Table.Th>
                  <Table.Th>YooKassa ID</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {payments.map((payment) => (
                  <Table.Tr key={payment.id}>
                    <Table.Td>
                      <Text size="xs">{formatDate(payment.paid_at || payment.created_at)}</Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" fw={500}>
                        {payment.amount} {payment.currency}
                      </Text>
                    </Table.Td>
                    <Table.Td>{getInvoiceTypeBadge(payment.invoice_type)}</Table.Td>
                    <Table.Td>{getPaymentStatusBadge(payment.status)}</Table.Td>
                    <Table.Td>
                      <Text size="sm">{payment.month}</Text>
                    </Table.Td>
                    <Table.Td>
                      {payment.yookasa_id ? (
                        <Group gap={4}>
                          <Text size="xs" style={{ fontFamily: 'monospace' }}>
                            {payment.yookasa_id.substring(0, 8)}...
                          </Text>
                          <CopyButton value={payment.yookasa_id}>
                            {({ copied, copy }) => (
                              <Tooltip label={copied ? 'Скопировано' : 'Копировать'}>
                                <ActionIcon
                                  size="xs"
                                  variant="subtle"
                                  color={copied ? 'green' : 'gray'}
                                  onClick={copy}
                                >
                                  {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                                </ActionIcon>
                              </Tooltip>
                            )}
                          </CopyButton>
                          <Tooltip label="Открыть в YooKassa">
                            <ActionIcon
                              size="xs"
                              variant="subtle"
                              color="blue"
                              component="a"
                              href={`https://yookassa.ru/my/payments?search=${payment.yookasa_id}`}
                              target="_blank"
                            >
                              <IconExternalLink size={12} />
                            </ActionIcon>
                          </Tooltip>
                        </Group>
                      ) : (
                        <Text size="xs" c="dimmed">-</Text>
                      )}
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        )}
      </Paper>

      <Paper p="md" shadow="sm">
        <Group gap="xs" mb="md">
          <IconDeviceMobile size={20} />
          <Text size="lg" fw={600}>Устройства</Text>
        </Group>

        {devicesLoading ? (
          <Stack align="center" py="xl">
            <Loader size="sm" />
          </Stack>
        ) : devices.length === 0 ? (
          <Alert color="gray">Устройства не найдены</Alert>
        ) : (
          <ScrollArea>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>HWID</Table.Th>
                  <Table.Th>Устройство</Table.Th>
                  <Table.Th>Платформа</Table.Th>
                  <Table.Th>Добавлено</Table.Th>
                  <Table.Th>Действия</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {devices.map((device) => (
                  <Table.Tr key={device.hwid}>
                    <Table.Td>
                      <Group gap={4}>
                        <Text size="xs" style={{ fontFamily: 'monospace' }}>
                          {maskHwid(device.hwid)}
                        </Text>
                        <CopyButton value={device.hwid}>
                          {({ copied, copy }) => (
                            <Tooltip label={copied ? 'Скопировано' : 'Копировать HWID'}>
                              <ActionIcon
                                size="xs"
                                variant="subtle"
                                color={copied ? 'green' : 'gray'}
                                onClick={copy}
                              >
                                {copied ? <IconCheck size={12} /> : <IconCopy size={12} />}
                              </ActionIcon>
                            </Tooltip>
                          )}
                        </CopyButton>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm">{device.device_model || '-'}</Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge variant="light" size="xs">
                        {device.platform || 'Unknown'}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Text size="xs">{formatDate(device.created_at)}</Text>
                    </Table.Td>
                    <Table.Td>
                      <Tooltip label="Удалить устройство">
                        <ActionIcon
                          size="sm"
                          variant="subtle"
                          color="red"
                          onClick={() => openDeleteModal(device)}
                        >
                          <IconTrash size={14} />
                        </ActionIcon>
                      </Tooltip>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        )}
      </Paper>

      <Modal
        opened={deleteModalOpened}
        onClose={() => setDeleteModalOpened(false)}
        title="Подтверждение удаления"
        centered
      >
        <Text mb="md">
          Вы уверены, что хотите удалить устройство{' '}
          <Text span fw={600}>{deviceToDelete ? maskHwid(deviceToDelete.hwid) : ''}</Text>?
        </Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={() => setDeleteModalOpened(false)}>
            Отмена
          </Button>
          <Button color="red" onClick={handleDeleteDevice} loading={deleting}>
            Удалить
          </Button>
        </Group>
      </Modal>

      <Modal
        opened={revokeModalOpened}
        onClose={() => setRevokeModalOpened(false)}
        title="Отзыв подписки"
        centered
      >
        <Text mb="md">
          Вы уверены, что хотите отозвать подписку пользователя{' '}
          <Text span fw={600}>{user?.telegram_id}</Text>?
          <Text size="sm" c="dimmed" mt="xs">
            Это действие сгенерирует новую ссылку на подписку и сбросит текущую.
          </Text>
        </Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={() => setRevokeModalOpened(false)}>
            Отмена
          </Button>
          <Button color="red" onClick={handleRevokeSubscription} loading={revoking}>
            Отозвать
          </Button>
        </Group>
      </Modal>
    </Stack>
  )
}

export default UserDetailsPage
