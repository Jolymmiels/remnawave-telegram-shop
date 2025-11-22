import React, { useState, useEffect } from 'react'
import {
  Stack,
  Paper,
  TextInput,
  Button,
  Table,
  Group,
  Badge,
  ActionIcon,
  Modal,
  Text,
  Pagination,
  Loader,
  Alert,
  Menu,
  rem,
  Card,
  Flex,
  Box,
  SimpleGrid,
} from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import {
  IconSearch,
  IconDots,
  IconEdit,
  IconTrash,
  IconBan,
  IconCheck,
  IconEye,
  IconCoin,
  IconUsers,
} from '@tabler/icons-react'
import { http } from '../../lib/http'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import UserEditModal from './UserEditModal'
import { getTelegramSafeAreaStyles, getTelegramSafeAreaDropdownStyles } from '../../lib/telegram-safe-area'

interface User {
  id: number
  telegram_id: number
  expire_at?: string
  created_at: string
  subscription_link?: string
  language: string
  is_blocked: boolean
  payments_count: number
  referrals_count: number
  total_spent: number
}

interface UserSearchResponse {
  users: User[]
  total: number
}

interface Payment {
  id: number
  amount: number
  currency: string
  status: string
  invoice_type: string
  created_at: string
  paid_at?: string
}

const UserManagement: React.FC = () => {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [totalUsers, setTotalUsers] = useState(0)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [userPayments, setUserPayments] = useState<Payment[]>([])
  const [paymentsLoading, setPaymentsLoading] = useState(false)
  
  const [detailsOpened, { open: openDetails, close: closeDetails }] = useDisclosure(false)
  const [editOpened, { open: openEdit, close: closeEdit }] = useDisclosure(false)
  const [deleteOpened, { open: openDelete, close: closeDelete }] = useDisclosure(false)
  
  const isMobile = useMediaQuery('(max-width: 768px)')
  const itemsPerPage = 20

  const fetchUsers = async (page: number = 1, query: string = '') => {
    // Prevent duplicate requests
    if (loading) return
    
    setLoading(true)
    try {
      const offset = (page - 1) * itemsPerPage
      const params = new URLSearchParams({
        limit: itemsPerPage.toString(),
        offset: offset.toString(),
      })
      if (query.trim()) {
        params.append('q', query.trim())
      }
      
      const response: UserSearchResponse = await http.get(`/api/users/search?${params}`)
      setUsers(response.users)
      setTotalUsers(response.total)
    } catch (error) {
      console.error('Failed to fetch users:', error)
      notifications.show({
        title: 'Ошибка',
        message: 'Не удалось загрузить пользователей',
        color: 'red',
      })
    } finally {
      setLoading(false)
    }
  }

  const fetchUserPayments = async (telegramId: number) => {
    // Prevent duplicate requests
    if (paymentsLoading) return
    
    setPaymentsLoading(true)
    try {
      const payments: Payment[] = await http.get(`/api/users/${telegramId}/payments`)
      setUserPayments(payments)
    } catch (error) {
      console.error('Failed to fetch user payments:', error)
      setUserPayments([])
    } finally {
      setPaymentsLoading(false)
    }
  }

  const handleSearch = () => {
    setCurrentPage(1)
    fetchUsers(1, searchQuery)
  }

  const handleUserAction = async (user: User, action: 'block' | 'unblock' | 'delete') => {
    try {
      if (action === 'delete') {
        await http.delete(`/api/users/${user.telegram_id}/delete`)
        notifications.show({
          title: 'Успешно',
          message: 'Пользователь удален',
          color: 'green',
        })
      } else if (action === 'block') {
        await http.post(`/api/users/${user.telegram_id}/block`, {})
        notifications.show({
          title: 'Успешно',
          message: 'Пользователь заблокирован',
          color: 'green',
        })
      } else if (action === 'unblock') {
        await http.post(`/api/users/${user.telegram_id}/unblock`, {
          expire_at: null // Restore subscription
        })
        notifications.show({
          title: 'Успешно',
          message: 'Пользователь разблокирован',
          color: 'green',
        })
      }
      
      // Refresh users list
      fetchUsers(currentPage, searchQuery)
      closeDelete()
    } catch (error) {
      console.error(`Failed to ${action} user:`, error)
      notifications.show({
        title: 'Ошибка',
        message: `Не удалось ${action === 'delete' ? 'удалить' : action === 'block' ? 'заблокировать' : 'разблокировать'} пользователя`,
        color: 'red',
      })
    }
  }

  const openUserDetails = (user: User) => {
    setSelectedUser(user)
    fetchUserPayments(user.telegram_id)
    openDetails()
  }

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Не указано'
    return new Date(dateString).toLocaleString('ru-RU')
  }

  const getStatusBadge = (user: User) => {
    if (!user.expire_at) {
      return <Badge color="gray">Без подписки</Badge>
    }
    
    const expireDate = new Date(user.expire_at)
    const now = new Date()
    
    if (expireDate < now) {
      return <Badge color="red">Истекла</Badge>
    } else {
      return <Badge color="green">Активна</Badge>
    }
  }

  useEffect(() => {
    fetchUsers()
  }, [currentPage])

  const totalPages = Math.ceil(totalUsers / itemsPerPage)

  // Mobile Card Component
  const UserCard: React.FC<{ user: User }> = ({ user }) => (
    <Card padding="sm" shadow="sm" withBorder style={{ opacity: user.is_blocked ? 0.6 : 1 }}>
      <Flex justify="space-between" align="flex-start" mb="xs">
        <Box>
          <Flex gap="xs" align="center">
            <Text fw={500} size="sm">ID: {user.telegram_id}</Text>
            {user.is_blocked && <Badge color="red" size="xs">Заблокирован</Badge>}
          </Flex>
          {getStatusBadge(user)}
        </Box>
        <Menu shadow="md" width={200}>
          <Menu.Target>
            <ActionIcon variant="subtle" size="sm">
              <IconDots size={16} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown
            style={getTelegramSafeAreaDropdownStyles()}
          >
            <Menu.Item
              leftSection={<IconEye style={{ width: rem(14), height: rem(14) }} />}
              onClick={() => openUserDetails(user)}
            >
              Подробно
            </Menu.Item>
            <Menu.Item
              leftSection={<IconEdit style={{ width: rem(14), height: rem(14) }} />}
              onClick={() => {
                setSelectedUser(user)
                openEdit()
              }}
            >
              Редактировать
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              leftSection={<IconBan style={{ width: rem(14), height: rem(14) }} />}
              color="orange"
              onClick={() => handleUserAction(user, 'block')}
            >
              Заблокировать
            </Menu.Item>
            <Menu.Item
              leftSection={<IconCheck style={{ width: rem(14), height: rem(14) }} />}
              color="green"
              onClick={() => handleUserAction(user, 'unblock')}
            >
              Разблокировать
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item
              leftSection={<IconTrash style={{ width: rem(14), height: rem(14) }} />}
              color="red"
              onClick={() => {
                setSelectedUser(user)
                openDelete()
              }}
            >
              Удалить
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Flex>

      <Stack gap="xs">
        <Group justify="space-between">
          <Text size="xs" c="dimmed">Истекает</Text>
          <Text size="xs">{formatDate(user.expire_at)}</Text>
        </Group>
        
        <Group justify="space-between">
          <Text size="xs" c="dimmed">Язык</Text>
          <Badge variant="light" size="xs">{(user.language || 'EN').toUpperCase()}</Badge>
        </Group>
        
        <SimpleGrid cols={3}>
          <Box ta="center">
            <Group gap={2} justify="center">
              <IconCoin size={12} />
              <Text size="xs" c="dimmed">Платежи</Text>
            </Group>
            <Text size="sm" fw={500}>{user.payments_count}</Text>
          </Box>
          
          <Box ta="center">
            <Group gap={2} justify="center">
              <IconUsers size={12} />
              <Text size="xs" c="dimmed">Рефералы</Text>
            </Group>
            <Text size="sm" fw={500}>{user.referrals_count}</Text>
          </Box>
          
          <Box ta="center">
            <Text size="xs" c="dimmed">Потратил</Text>
            <Text size="sm" fw={500}>{user.total_spent.toFixed(2)} ₽</Text>
          </Box>
        </SimpleGrid>
      </Stack>
    </Card>
  )

  return (
    <Stack>
      <Paper p="md" shadow="sm">
        <Group justify="space-between" mb="md">
          <Text size="sm" c="dimmed">Всего пользователей: {totalUsers}</Text>
        </Group>
        
        <Group mb="md">
          <TextInput
            placeholder="Поиск по Telegram ID..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.currentTarget.value)}
            leftSection={<IconSearch size={16} />}
            flex={1}
            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
          />
          <Button onClick={handleSearch} loading={loading}>
            Поиск
          </Button>
        </Group>

        {loading ? (
          <Group justify="center" py="xl">
            <Loader />
          </Group>
        ) : isMobile ? (
          // Mobile card view
          <>
            <Stack gap="sm">
              {users.map((user) => (
                <UserCard key={user.id} user={user} />
              ))}
            </Stack>
            {totalPages > 1 && (
              <Group justify="center" mt="md">
                <Pagination
                  value={currentPage}
                  onChange={setCurrentPage}
                  total={totalPages}
                  size="sm"
                />
              </Group>
            )}
          </>
        ) : (
          // Desktop table view
          <>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Telegram ID</Table.Th>
                  <Table.Th>Статус подписки</Table.Th>
                  <Table.Th>Истекает</Table.Th>
                  <Table.Th>Язык</Table.Th>
                  <Table.Th>Платежи</Table.Th>
                  <Table.Th>Рефералы</Table.Th>
                  <Table.Th>Потратил</Table.Th>
                  <Table.Th>Действия</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {users.map((user) => (
                  <Table.Tr key={user.id} style={{ opacity: user.is_blocked ? 0.6 : 1, backgroundColor: user.is_blocked ? 'rgba(250, 82, 82, 0.05)' : 'transparent' }}>
                    <Table.Td>
                      <Group gap="xs">
                        <Text fw={500}>{user.telegram_id}</Text>
                        {user.is_blocked && <Badge color="red" size="sm">Заблокирован</Badge>}
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      {getStatusBadge(user)}
                    </Table.Td>
                    <Table.Td>
                      {formatDate(user.expire_at)}
                    </Table.Td>
                    <Table.Td>
                      <Badge variant="light" size="sm">
                        {(user.language || 'EN').toUpperCase()}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Group gap={4}>
                        <IconCoin size={14} />
                        <Text size="sm">{user.payments_count}</Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Group gap={4}>
                        <IconUsers size={14} />
                        <Text size="sm">{user.referrals_count}</Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" fw={500}>
                        {user.total_spent.toFixed(2)} ₽
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Menu shadow="md" width={200}>
                        <Menu.Target>
                          <ActionIcon variant="subtle" size="sm">
                            <IconDots size={16} />
                          </ActionIcon>
                        </Menu.Target>

                        <Menu.Dropdown
                          style={getTelegramSafeAreaDropdownStyles()}
                        >
                          <Menu.Item
                            leftSection={<IconEye style={{ width: rem(14), height: rem(14) }} />}
                            onClick={() => openUserDetails(user)}
                          >
                            Подробно
                          </Menu.Item>
                          <Menu.Item
                            leftSection={<IconEdit style={{ width: rem(14), height: rem(14) }} />}
                            onClick={() => {
                              setSelectedUser(user)
                              openEdit()
                            }}
                          >
                            Редактировать
                          </Menu.Item>
                          <Menu.Divider />
                          <Menu.Item
                            leftSection={<IconBan style={{ width: rem(14), height: rem(14) }} />}
                            color="orange"
                            onClick={() => handleUserAction(user, 'block')}
                          >
                            Заблокировать
                          </Menu.Item>
                          <Menu.Item
                            leftSection={<IconCheck style={{ width: rem(14), height: rem(14) }} />}
                            color="green"
                            onClick={() => handleUserAction(user, 'unblock')}
                          >
                            Разблокировать
                          </Menu.Item>
                          <Menu.Divider />
                          <Menu.Item
                            leftSection={<IconTrash style={{ width: rem(14), height: rem(14) }} />}
                            color="red"
                            onClick={() => {
                              setSelectedUser(user)
                              openDelete()
                            }}
                          >
                            Удалить
                          </Menu.Item>
                        </Menu.Dropdown>
                      </Menu>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>

            {totalPages > 1 && (
              <Group justify="center" mt="md">
                <Pagination
                  value={currentPage}
                  onChange={setCurrentPage}
                  total={totalPages}
                  size="sm"
                />
              </Group>
            )}
          </>
        )}
      </Paper>

      {/* User Details Modal */}
      <Modal 
        opened={detailsOpened} 
        onClose={closeDetails} 
        title="Детали пользователя" 
        size="lg"
        styles={getTelegramSafeAreaStyles()}
      >
        {selectedUser && (
          <Stack>
            <Group>
              <Text fw={500}>Telegram ID:</Text>
              <Text>{selectedUser.telegram_id}</Text>
            </Group>
            <Group>
              <Text fw={500}>Статус подписки:</Text>
              {getStatusBadge(selectedUser)}
            </Group>
            {selectedUser.is_blocked && (
              <Group>
                <Text fw={500}>Статус блокировки:</Text>
                <Badge color="red">Пользователь заблокирован</Badge>
              </Group>
            )}
            <Group>
              <Text fw={500}>Дата регистрации:</Text>
              <Text>{formatDate(selectedUser.created_at)}</Text>
            </Group>
            <Group>
              <Text fw={500}>Подписка истекает:</Text>
              <Text>{formatDate(selectedUser.expire_at)}</Text>
            </Group>
            <Group>
              <Text fw={500}>Язык:</Text>
              <Badge variant="light">{(selectedUser.language || 'EN').toUpperCase()}</Badge>
            </Group>
            
            <Text fw={500} mt="md">История платежей:</Text>
            {paymentsLoading ? (
              <Loader size="sm" />
            ) : userPayments.length > 0 ? (
              <Table>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Сумма</Table.Th>
                    <Table.Th>Валюта</Table.Th>
                    <Table.Th>Статус</Table.Th>
                    <Table.Th>Тип</Table.Th>
                    <Table.Th>Дата</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {userPayments.map((payment) => (
                    <Table.Tr key={payment.id}>
                      <Table.Td>{payment.amount}</Table.Td>
                      <Table.Td>{payment.currency}</Table.Td>
                      <Table.Td>
                        <Badge
                          color={payment.status === 'paid' ? 'green' : 'yellow'}
                          size="sm"
                        >
                          {payment.status}
                        </Badge>
                      </Table.Td>
                      <Table.Td>{payment.invoice_type}</Table.Td>
                      <Table.Td>{formatDate(payment.paid_at || payment.created_at)}</Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            ) : (
              <Alert>Платежи не найдены</Alert>
            )}
          </Stack>
        )}
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal 
        opened={deleteOpened} 
        onClose={closeDelete} 
        title="Подтверждение удаления"
        styles={getTelegramSafeAreaStyles()}
      >
        {selectedUser && (
          <Stack>
            <Text>
              Вы действительно хотите удалить пользователя с Telegram ID {selectedUser.telegram_id}?
            </Text>
            <Text size="sm" c="dimmed">
              Это действие нельзя будет отменить.
            </Text>
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={closeDelete}>
                Отмена
              </Button>
              <Button
                color="red"
                onClick={() => selectedUser && handleUserAction(selectedUser, 'delete')}
              >
                Удалить
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>

      {/* Edit User Modal */}
      <UserEditModal
        user={selectedUser}
        opened={editOpened}
        onClose={closeEdit}
        onUserUpdated={() => fetchUsers(currentPage, searchQuery)}
      />
    </Stack>
  )
}

export default UserManagement