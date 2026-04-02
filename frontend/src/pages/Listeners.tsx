import { useState } from 'react'
import useSWR from 'swr'
import { api, ApiError, type ListenerUser, type ListenerView, type TransitRouteView } from '@/services/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { useClipboard } from '@/hooks/useClipboard'
import { generatePassword, generateUsername } from '@/lib/random'
import { toast } from 'sonner'
import { Network, Radio, Users, ShieldAlert, Plus, Trash2, Pencil, AlertTriangle, Eye, EyeOff, Copy, Shuffle, GitBranch } from 'lucide-react'

function buildListenerWarning(listener: ListenerView) {
  if (listener.transitMissing) return '这条入口绑定的中转线路已经不存在了，请重新选择。'
  if (listener.transitDisabled) return '这条入口绑定的中转线路目前被停用了。'
  if (listener.transitProxyMissing) return '这条入口依赖的中转节点当前已经不可用，请重新选择线路。'
  if (listener.groupMissing) return '这条入口原先绑定的出口线路已经不存在了，请重新选择线路。'
  if (listener.providerMissing) return '这条入口依赖的出口 provider 已经不存在了。'
  if (listener.providerDisabled) return '这条入口依赖的出口 provider 目前被停用了。'
  if (listener.landingMissing) return '这条入口绑定的落地节点已经被删除了。'
  if (listener.landingDisabled) return '这条入口绑定的落地节点目前被停用了。'
  return ''
}

function formatRouteMode(routeMode: string) {
  return routeMode === 'transit' ? '中转模式' : '直连模式'
}

function PasswordCell({ value }: { value: string }) {
  const [visible, setVisible] = useState(false)
  const { copy } = useClipboard()

  const handleCopy = () => void copy(value, '密码已复制')

  return (
    <span className="inline-flex items-center gap-1">
      <span
        className="cursor-pointer select-all font-mono text-xs"
        onClick={() => setVisible(!visible)}
        title={visible ? '点击隐藏' : '点击显示'}
      >
        {visible ? value : '••••••••'}
      </span>
      <button type="button" onClick={() => setVisible(!visible)} className="rounded p-0.5 text-zinc-400 transition-colors hover:bg-zinc-200 hover:text-zinc-600" title={visible ? '隐藏' : '显示'}>
        {visible ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
      </button>
      <button type="button" onClick={handleCopy} className="rounded p-0.5 text-zinc-400 transition-colors hover:bg-zinc-200 hover:text-zinc-600" title="复制密码">
        <Copy className="h-3.5 w-3.5" />
      </button>
    </span>
  )
}

interface ListenerForm {
  name: string
  type: string
  listen: string
  port: string
  udp: boolean
  route_mode: string
  egress_group: string
  transit_route: string
  username: string
  password: string
}

const emptyListenerForm: ListenerForm = {
  name: '',
  type: 'socks',
  listen: '0.0.0.0',
  port: '',
  udp: true,
  route_mode: 'direct',
  egress_group: '',
  transit_route: '',
  username: '',
  password: '',
}

export default function Listeners() {
  const { data: listeners, mutate: mutateListeners } = useSWR('/api/listeners', api.getListeners)
  const { data: groups, mutate: mutateGroups } = useSWR('/api/groups', api.getGroups)
  const { data: transitRoutes, mutate: mutateTransitRoutes } = useSWR('/api/transit-routes', api.getTransitRoutes)
  const { copy } = useClipboard()

  const groupNames = groups?.map((group) => group.name) || []
  const availableTransitRoutes = transitRoutes || []
  const transitRouteNames = availableTransitRoutes.map((route) => route.name)
  const canCreateListener = groupNames.length > 0 || transitRouteNames.length > 0

  const mutateAll = () => {
    void mutateListeners()
    void mutateGroups()
    void mutateTransitRoutes()
  }

  const [showAddListener, setShowAddListener] = useState(false)
  const [showEditListener, setShowEditListener] = useState(false)
  const [listenerForm, setListenerForm] = useState<ListenerForm>({ ...emptyListenerForm })
  const [editingListenerName, setEditingListenerName] = useState('')
  const [listenerSubmitting, setListenerSubmitting] = useState(false)
  const [pendingDeleteListenerName, setPendingDeleteListenerName] = useState<string | null>(null)
  const [deletingListener, setDeletingListener] = useState(false)

  const applyRouteMode = (routeMode: string) => {
    setListenerForm((current) => ({
      ...current,
      route_mode: routeMode,
      egress_group: routeMode === 'direct' ? current.egress_group : '',
      transit_route: routeMode === 'transit' ? current.transit_route : '',
    }))
  }

  const openAddListenerDialog = () => {
    const routeMode = groupNames.length > 0 ? 'direct' : 'transit'
    setListenerForm({
      ...emptyListenerForm,
      route_mode: routeMode,
      username: generateUsername(),
      password: generatePassword(),
    })
    setShowAddListener(true)
  }

  const buildPayload = () => {
    const payload = {
      name: listenerForm.name.trim(),
      type: listenerForm.type,
      listen: listenerForm.listen || '0.0.0.0',
      port: Number(listenerForm.port),
      udp: listenerForm.udp,
      route_mode: listenerForm.route_mode,
      egress_group: listenerForm.route_mode === 'direct' ? listenerForm.egress_group : '',
      transit_route: listenerForm.route_mode === 'transit' ? listenerForm.transit_route : '',
      users: [] as ListenerUser[],
    }
    if (listenerForm.username.trim() && listenerForm.password.trim()) {
      payload.users = [{ username: listenerForm.username.trim(), password: listenerForm.password.trim() }]
    }
    return payload
  }

  const handleAddListener = async () => {
    if (!listenerForm.name.trim() || !listenerForm.port) {
      toast.error('入口名称和端口都需要填写。')
      return
    }
    if (listenerForm.route_mode === 'direct' && !listenerForm.egress_group) {
      toast.error('直连模式下必须选择一条出口线路。')
      return
    }
    if (listenerForm.route_mode === 'transit' && !listenerForm.transit_route) {
      toast.error('中转模式下必须选择一条中转线路。')
      return
    }

    setListenerSubmitting(true)
    try {
      const data = await api.addListener(buildPayload())
      if (!data.ok) throw new Error('添加失败')
      mutateAll()
      setShowAddListener(false)
      setListenerForm({ ...emptyListenerForm })
      toast.success('本地入口已创建。')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '添加失败')
    } finally {
      setListenerSubmitting(false)
    }
  }

  const openEditListener = (listener: ListenerView) => {
    const firstUser = listener.users?.[0]
    setEditingListenerName(listener.name)
    setListenerForm({
      name: listener.name,
      type: listener.type || 'socks',
      listen: listener.listen || '0.0.0.0',
      port: String(listener.port),
      udp: listener.udp !== false,
      route_mode: listener.routeMode || 'direct',
      egress_group: listener.routeMode === 'direct' ? listener.egressGroup || '' : '',
      transit_route: listener.routeMode === 'transit' ? listener.transitRoute || '' : '',
      username: firstUser?.username || '',
      password: firstUser?.password || '',
    })
    setShowEditListener(true)
  }

  const handleEditListener = async () => {
    if (!listenerForm.port) {
      toast.error('端口不能为空。')
      return
    }
    if (listenerForm.route_mode === 'direct' && !listenerForm.egress_group) {
      toast.error('直连模式下必须选择一条出口线路。')
      return
    }
    if (listenerForm.route_mode === 'transit' && !listenerForm.transit_route) {
      toast.error('中转模式下必须选择一条中转线路。')
      return
    }

    setListenerSubmitting(true)
    try {
      const data = await api.updateListener(editingListenerName, buildPayload())
      if (!data.ok) throw new Error('修改失败')
      mutateAll()
      setShowEditListener(false)
      setListenerForm({ ...emptyListenerForm })
      toast.success('本地入口已更新。')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '修改失败')
    } finally {
      setListenerSubmitting(false)
    }
  }

  const handleDeleteListener = async () => {
    if (!pendingDeleteListenerName) return
    setDeletingListener(true)
    try {
      const data = await api.deleteListener(pendingDeleteListenerName)
      if (!data.ok) throw new Error('删除失败')
      mutateAll()
      toast.success(`已删除本地入口“${pendingDeleteListenerName}”。`)
      setPendingDeleteListenerName(null)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '删除失败')
    } finally {
      setDeletingListener(false)
    }
  }

  const renderTargetField = () => {
    if (listenerForm.route_mode === 'transit') {
      return (
        <div className="space-y-2">
          <Label>中转线路</Label>
          <Select value={listenerForm.transit_route} onValueChange={(value) => setListenerForm({ ...listenerForm, transit_route: value })}>
            <SelectTrigger>
              <SelectValue placeholder="选择一条中转线路" />
            </SelectTrigger>
            <SelectContent>
              {availableTransitRoutes.map((route: TransitRouteView) => (
                <SelectItem key={route.name} value={route.name}>{route.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-xs leading-5 text-zinc-500">这个入口接入后，会先走上游固定节点，再接后半段出口链路。</p>
        </div>
      )
    }

    return (
      <div className="space-y-2">
        <Label>出口线路</Label>
        <Select value={listenerForm.egress_group} onValueChange={(value) => setListenerForm({ ...listenerForm, egress_group: value })}>
          <SelectTrigger>
            <SelectValue placeholder="选择一条出口线路" />
          </SelectTrigger>
          <SelectContent>
            {groupNames.map((name) => (
              <SelectItem key={name} value={name}>{name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs leading-5 text-zinc-500">这个入口连上后，会直接走你选中的本地出口线路。</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Local Access"
        title="给浏览器和设备准备一个可连接的本地入口"
        description="本地入口就是你最终要在系统、浏览器或其他设备里填写的代理地址。现在可以按场景选择直连模式，或者直接绑定一条已经整理好的中转线路。"
        actions={
          <Button onClick={openAddListenerDialog} disabled={!canCreateListener}>
            <Plus className="h-4 w-4" />
            添加本地入口
          </Button>
        }
      />

      <NoticeCard
        icon={<Radio className="h-4 w-4" />}
        title="什么时候需要多个本地入口"
        description="如果你想给不同设备、不同权限或不同链路单独分配入口，就值得创建多个本地入口。比如“电脑默认”“电视流媒体”“经过中转的远端业务”这种分法会更清晰。"
      />

      {!canCreateListener && (
        <NoticeCard
          tone="warning"
          icon={<AlertTriangle className="h-4 w-4" />}
          title="还不能创建本地入口"
          description="因为当前既没有可用出口线路，也没有可用中转线路。先去整理好至少一种目标链路，这里才能继续往下配。"
        />
      )}

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">本地入口列表</h2>
        </div>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {!listeners ? (
            <p className="text-sm text-zinc-500">加载中...</p>
          ) : listeners.length === 0 ? (
            <EmptyStateCard
              className="md:col-span-2 xl:col-span-3"
              icon={<Radio className="h-5 w-5" />}
              title="还没有任何本地入口"
              description="先创建一个入口，让浏览器、系统代理或其他设备真正能连到 AXIS。创建时只需要决定协议、端口，以及是直连出口还是绑定中转线路。"
              action={
                <Button onClick={openAddListenerDialog} disabled={!canCreateListener}>
                  <Plus className="h-4 w-4" />
                  添加第一个本地入口
                </Button>
              }
            />
          ) : (
            listeners.map((listener) => {
              const warning = buildListenerWarning(listener)
              const degraded = listener.status !== 'configured'
              return (
                <Card key={listener.name} className={degraded ? 'border-amber-200 bg-amber-50/30' : ''}>
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div>
                        <CardTitle className="flex items-center gap-2 text-base">
                          <Network className="h-4 w-4 text-zinc-400" />
                          {listener.name}
                        </CardTitle>
                        <CardDescription className="mt-1 font-mono">{listener.listen}:{listener.port}</CardDescription>
                      </div>
                      <div className="flex items-center gap-2">
                        {listener.status !== 'configured' ? (
                          <div className="rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700">
                            {listener.status === 'disabled' ? '停用' : listener.status === 'orphaned' ? '孤立' : '降级'}
                          </div>
                        ) : null}
                        <div className="rounded border border-blue-100 bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700">{listener.type}</div>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="flex flex-1 flex-col justify-between">
                    <div className="space-y-4">
                      <div className="flex flex-wrap gap-2 text-xs">
                        <span className="inline-flex items-center gap-1 rounded border bg-zinc-100 px-2 py-1 text-zinc-600">
                          {listener.routeMode === 'transit' ? <GitBranch className="h-3 w-3" /> : <Radio className="h-3 w-3" />}
                          {formatRouteMode(listener.routeMode)}
                        </span>
                        <span className="inline-flex items-center gap-1 rounded border bg-zinc-100 px-2 py-1 text-zinc-600">
                          <Users className="h-3 w-3" />
                          账号 {listener.userCount}
                        </span>
                        <span className="inline-flex items-center rounded border bg-zinc-100 px-2 py-1 text-zinc-600">UDP {listener.udp ? '开' : '关'}</span>
                      </div>

                      <div className="space-y-2 rounded-lg border bg-zinc-50 p-3 text-sm">
                        <div className="flex justify-between gap-3">
                          <span className="text-zinc-500">目标链路</span>
                          <span className="font-medium text-zinc-900">{listener.targetName || listener.egressGroup}</span>
                        </div>
                        <div className="flex justify-between gap-3">
                          <span className="text-zinc-500">实际出口组</span>
                          <span className="font-medium text-zinc-900">{listener.egressGroup || '暂未确定'}</span>
                        </div>
                        <div className="flex justify-between gap-3">
                          <span className="text-zinc-500">当前出口</span>
                          <span className="max-w-[140px] truncate text-zinc-900" title={listener.currentProxy || '还没有选中的节点'}>{listener.currentProxy || '还没有选中的节点'}</span>
                        </div>
                        <div className="flex justify-between gap-3">
                          <span className="text-zinc-500">实际路径</span>
                          <span className="break-all text-right text-zinc-900" title={listener.routeSummary || listener.targetName}>{listener.routeSummary || listener.targetName}</span>
                        </div>
                      </div>

                      {listener.users && listener.users.length > 0 ? (
                        <div className="space-y-2 rounded-lg border bg-zinc-50 p-3 text-sm">
                          {listener.users.map((user, index) => (
                            <div key={index} className="space-y-1.5">
                              <div className="flex items-center justify-between">
                                <span className="text-zinc-500">用户名</span>
                                <div className="flex items-center gap-1">
                                  <span className="select-all font-mono text-xs text-zinc-900">{user.username}</span>
                                  <button type="button" onClick={() => void copy(user.username, '用户名已复制')} className="rounded p-0.5 text-zinc-400 transition-colors hover:bg-zinc-200 hover:text-zinc-600" title="复制用户名">
                                    <Copy className="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              </div>
                              <div className="flex items-center justify-between">
                                <span className="text-zinc-500">密码</span>
                                <PasswordCell value={user.password} />
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>

                    {warning ? (
                      <div className="mt-4 flex items-start gap-2 rounded bg-amber-50 p-2 text-xs text-amber-700">
                        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                        <p>{warning}</p>
                      </div>
                    ) : null}

                    {listener.userCount === 0 && listener.status === 'configured' ? (
                      <div className="mt-4 flex items-start gap-2 rounded bg-amber-50 p-2 text-xs text-amber-700">
                        <ShieldAlert className="h-4 w-4 shrink-0" />
                        <p>这个入口没有账号密码，知道地址的人都可以直接连接。</p>
                      </div>
                    ) : null}

                    <div className="mt-4 flex items-center gap-2 border-t pt-3">
                      <Button variant="outline" size="sm" onClick={() => openEditListener(listener)}>
                        <Pencil className="mr-1 h-3.5 w-3.5" />
                        编辑入口
                      </Button>
                      <Button variant="outline" size="sm" className="text-red-600 hover:bg-red-50 hover:text-red-700" onClick={() => setPendingDeleteListenerName(listener.name)}>
                        <Trash2 className="mr-1 h-3.5 w-3.5" />
                        删除入口
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )
            })
          )}
        </div>
      </section>

      <Dialog open={showAddListener} onOpenChange={setShowAddListener}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加一个本地入口</DialogTitle>
            <DialogDescription>配置浏览器、系统或其他设备要连接的本地代理地址，并决定它是直连出口还是绑定中转链路。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>入口名称</Label>
              <Input placeholder="例如：电脑默认、电视流媒体、远端中转业务" value={listenerForm.name} onChange={(event) => setListenerForm({ ...listenerForm, name: event.target.value })} />
              <p className="text-xs leading-5 text-zinc-500">建议按设备或用途命名，后面更容易区分。</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>协议</Label>
                <Select value={listenerForm.type} onValueChange={(value) => setListenerForm({ ...listenerForm, type: value })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="socks">SOCKS5</SelectItem>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="mixed">Mixed</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>端口</Label>
                <Input type="number" placeholder="10801" value={listenerForm.port} onChange={(event) => setListenerForm({ ...listenerForm, port: event.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>监听地址</Label>
                <Input placeholder="0.0.0.0" value={listenerForm.listen} onChange={(event) => setListenerForm({ ...listenerForm, listen: event.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>链路模式</Label>
                <Select value={listenerForm.route_mode} onValueChange={applyRouteMode}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {groupNames.length > 0 ? <SelectItem value="direct">直连出口</SelectItem> : null}
                    {transitRouteNames.length > 0 ? <SelectItem value="transit">中转线路</SelectItem> : null}
                  </SelectContent>
                </Select>
              </div>
            </div>
            {renderTargetField()}
            <div className="flex items-center gap-3">
              <Switch checked={listenerForm.udp} onCheckedChange={(checked) => setListenerForm({ ...listenerForm, udp: checked })} />
              <Label>允许 UDP</Label>
            </div>
            <div className="border-t pt-4">
              <p className="mb-3 text-sm text-zinc-500">访问账号（如果留空，这个入口会变成无密码访问）</p>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>用户名</Label>
                  <div className="flex gap-1">
                    <Input placeholder="username" value={listenerForm.username} onChange={(event) => setListenerForm({ ...listenerForm, username: event.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, username: generateUsername() })} title="随机用户名">
                      <Shuffle className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>密码</Label>
                  <div className="flex gap-1">
                    <Input placeholder="password" value={listenerForm.password} onChange={(event) => setListenerForm({ ...listenerForm, password: event.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, password: generatePassword() })} title="随机密码">
                      <Shuffle className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddListener(false)}>取消</Button>
            <Button onClick={handleAddListener} disabled={listenerSubmitting}>{listenerSubmitting ? '创建中...' : '创建入口'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showEditListener} onOpenChange={setShowEditListener}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑入口：{editingListenerName}</DialogTitle>
            <DialogDescription>你可以修改协议、端口、链路模式或访问账号。清空账号字段即可移除认证。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>协议</Label>
                <Select value={listenerForm.type} onValueChange={(value) => setListenerForm({ ...listenerForm, type: value })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="socks">SOCKS5</SelectItem>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="mixed">Mixed</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>端口</Label>
                <Input type="number" value={listenerForm.port} onChange={(event) => setListenerForm({ ...listenerForm, port: event.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>监听地址</Label>
                <Input placeholder="0.0.0.0" value={listenerForm.listen} onChange={(event) => setListenerForm({ ...listenerForm, listen: event.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>链路模式</Label>
                <Select value={listenerForm.route_mode} onValueChange={applyRouteMode}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {groupNames.length > 0 ? <SelectItem value="direct">直连出口</SelectItem> : null}
                    {transitRouteNames.length > 0 ? <SelectItem value="transit">中转线路</SelectItem> : null}
                  </SelectContent>
                </Select>
              </div>
            </div>
            {renderTargetField()}
            <div className="flex items-center gap-3">
              <Switch checked={listenerForm.udp} onCheckedChange={(checked) => setListenerForm({ ...listenerForm, udp: checked })} />
              <Label>允许 UDP</Label>
            </div>
            <div className="border-t pt-4">
              <p className="mb-3 text-sm text-zinc-500">访问账号（清空两个字段即可移除认证）</p>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>用户名</Label>
                  <div className="flex gap-1">
                    <Input placeholder="username" value={listenerForm.username} onChange={(event) => setListenerForm({ ...listenerForm, username: event.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, username: generateUsername() })} title="随机用户名">
                      <Shuffle className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>密码</Label>
                  <div className="flex gap-1">
                    <Input placeholder="password" value={listenerForm.password} onChange={(event) => setListenerForm({ ...listenerForm, password: event.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, password: generatePassword() })} title="随机密码">
                      <Shuffle className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditListener(false)}>取消</Button>
            <Button onClick={handleEditListener} disabled={listenerSubmitting}>{listenerSubmitting ? '保存中...' : '保存入口'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(pendingDeleteListenerName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteListenerName(null)
        }}
        title="删除这个本地入口后会发生什么？"
        description={pendingDeleteListenerName ? `“${pendingDeleteListenerName}”删除后，设备就不能再通过这个地址连接 AXIS。已经绑定的出口或中转线路不会被一起删除。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingListener}
        onConfirm={handleDeleteListener}
      />
    </div>
  )
}
