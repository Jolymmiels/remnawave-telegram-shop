import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
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
  Menu,
  rem,
  Card,
  Flex,
  Box,
  SimpleGrid,
  Select,
} from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import {
  IconSearch,
  IconDots,
  IconTrash,
  IconBan,
  IconCheck,
  IconEye,
  IconCoin,
  IconUsers,
  IconRefresh,
  IconSortDescending,
  IconSortAscending,
} from '@tabler/icons-react'
import { http } from '../../lib/http'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { getTelegramSafeAreaStyles, getTelegramSafeAreaDropdownStyles } from '../../lib/telegram-safe-area'

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
  tg_username?: string | null
  tg_first_name?: string | null
  tg_last_name?: string | null
}

interface UserSearchResponse {
  users: User[]
  total: number
}



const UserManagement: React.FC = () => {
  const navigate = useNavigate()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<string>('date')
  const [sortOrder, setSortOrder] = useState<string>('desc')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [currentPage, setCurrentPage] = useState(1)
  const [totalUsers, setTotalUsers] = useState(0)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  
  const [deleteOpened, { open: openDelete, close: closeDelete }] = useDisclosure(false)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const [syncLoading, setSyncLoading] = useState(false)
  
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
        sort: sortBy,
        order: sortOrder,
      })
      if (query.trim()) {
        params.append('q', query.trim())
      }
      if (statusFilter) {
        params.append('status', statusFilter)
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

  const handleSearch = () => {
    setCurrentPage(1)
    fetchUsers(1, searchQuery)
  }

  const handleSync = async () => {
    try {
      setSyncLoading(true)
      await http.post('/api/sync', {})
      notifications.show({
        title: 'Синхронизация',
        message: 'Синхронизация запущена',
        color: 'blue',
      })
      setTimeout(() => fetchUsers(currentPage, searchQuery), 2000)
    } catch (error: any) {
      notifications.show({
        title: 'Ошибка',
        message: error?.message || 'Не удалось запустить синхронизацию',
        color: 'red',
      })
    } finally {
      setSyncLoading(false)
    }
  }

  const handleUserAction = async (user: User, action: 'block' | 'unblock' | 'delete') => {
    try {
      if (action === 'delete') {
        setDeleteLoading(true)
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
    } catch (error: any) {
      console.error(`Failed to ${action} user:`, error)
      notifications.show({
        title: 'Ошибка',
        message: error?.message || `Не удалось ${action === 'delete' ? 'удалить' : action === 'block' ? 'заблокировать' : 'разблокировать'} пользователя`,
        color: 'red',
      })
    } finally {
      setDeleteLoading(false)
    }
  }

  const openUserDetails = (user: User) => {
    navigate(`/user/${user.telegram_id}`)
  }

  const formatDate = (dateString?: string | null) => {
    if (!dateString) return 'Не указано'
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
    fetchUsers(currentPage, searchQuery)
  }, [currentPage, sortBy, sortOrder, statusFilter])

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
          {(user.tg_username || user.tg_first_name || user.tg_last_name) && (
            <Text size="xs" c="dimmed">
              {user.tg_username && `@${user.tg_username}`}
              {user.tg_username && (user.tg_first_name || user.tg_last_name) && ' · '}
              {[user.tg_first_name, user.tg_last_name].filter(Boolean).join(' ')}
            </Text>
          )}
          <Group gap={4}>
            {getStatusBadge(user)}
            <Badge variant="light" size="xs">{(user.language || 'en').toUpperCase()}</Badge>
          </Group>
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
        
        <SimpleGrid cols={2}>
          <Box ta="center">
            <Group gap={2} justify="center">
              <IconCoin size={12} />
              <Text size="xs" c="dimmed">Платежи</Text>
            </Group>
            <Text size="sm" fw={500}>{user.payments_count} · <Text span c="green">{user.total_spent.toFixed(0)} ₽</Text></Text>
          </Box>
          
          <Box ta="center">
            <Group gap={2} justify="center">
              <IconUsers size={12} />
              <Text size="xs" c="dimmed">Рефералы</Text>
            </Group>
            <Text size="sm" fw={500}>{user.referrals_count}</Text>
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
          <Button 
            size="xs" 
            variant="light" 
            leftSection={<IconRefresh size={14} />}
            loading={syncLoading}
            onClick={handleSync}
          >
            Синхронизация
          </Button>
        </Group>
        
        <Group mb="md" gap="xs" wrap="wrap">
          <TextInput
            placeholder="Поиск по Telegram ID..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.currentTarget.value)}
            leftSection={<IconSearch size={16} />}
            style={{ flex: 1, minWidth: isMobile ? '100%' : 200 }}
            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
          />
          <Select
            placeholder="Сортировка"
            value={sortBy}
            onChange={(value) => {
              setSortBy(value || 'date')
              setCurrentPage(1)
            }}
            data={[
              { value: 'date', label: 'По дате' },
              { value: 'spent', label: 'По сумме' },
              { value: 'referrals', label: 'По рефералам' },
            ]}
            style={{ width: isMobile ? '100%' : 140 }}
          />
          <ActionIcon
            variant="light"
            size="lg"
            onClick={() => {
              setSortOrder(sortOrder === 'desc' ? 'asc' : 'desc')
              setCurrentPage(1)
            }}
            title={sortOrder === 'desc' ? 'По убыванию' : 'По возрастанию'}
          >
            {sortOrder === 'desc' ? <IconSortDescending size={18} /> : <IconSortAscending size={18} />}
          </ActionIcon>
          <Select
            placeholder="Статус"
            value={statusFilter}
            onChange={(value) => {
              setStatusFilter(value || '')
              setCurrentPage(1)
            }}
            data={[
              { value: '', label: 'Все' },
              { value: 'active', label: 'Активные' },
              { value: 'expired', label: 'Истекшие' },
              { value: 'no_subscription', label: 'Без подписки' },
            ]}
            clearable
            style={{ width: isMobile ? '100%' : 140 }}
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
                      <Stack gap={2}>
                        <Group gap="xs">
                          <Text fw={500}>{user.telegram_id}</Text>
                          {user.is_blocked && <Badge color="red" size="sm">Заблокирован</Badge>}
                        </Group>
                        {(user.tg_username || user.tg_first_name || user.tg_last_name) && (
                          <Text size="xs" c="dimmed">
                            {user.tg_username && `@${user.tg_username}`}
                            {user.tg_username && (user.tg_first_name || user.tg_last_name) && ' · '}
                            {[user.tg_first_name, user.tg_last_name].filter(Boolean).join(' ')}
                          </Text>
                        )}
                      </Stack>
                    </Table.Td>
                    <Table.Td>
                      <Group gap={4}>
                        {getStatusBadge(user)}
                        <Badge variant="light" size="sm">{(user.language || 'en').toUpperCase()}</Badge>
                      </Group>
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
              <Button variant="default" onClick={closeDelete} disabled={deleteLoading}>
                Отмена
              </Button>
              <Button
                color="red"
                loading={deleteLoading}
                onClick={() => selectedUser && handleUserAction(selectedUser, 'delete')}
              >
                Удалить
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>
    </Stack>
  )
}

export default UserManagement