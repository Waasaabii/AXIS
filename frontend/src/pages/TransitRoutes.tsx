import { useState } from 'react'
import useSWR from 'swr'
import { api, ApiError, type AddTransitRouteRequest, type GroupView, type ProviderListItem, type TransitRouteView, type UpdateTransitRouteRequest } from '@/services/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { toast } from 'sonner'
import { AlertTriangle, GitBranch, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'

interface TransitRouteForm {
  name: string
  upstream_provider: string
  upstream_proxy_name: string
  egress_group: string
  notes: string
  enabled: boolean
}

const emptyRouteForm: TransitRouteForm = {
  name: '',
  upstream_provider: '',
  upstream_proxy_name: '',
  egress_group: '',
  notes: '',
  enabled: true,
}

function formatTimestamp(value?: string) {
  if (!value) return '尚未检测'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '尚未检测'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

function buildTransitWarning(route: TransitRouteView) {
  if (route.providerMissing) return '这条中转线路依赖的来源订阅已经不存在了。'
  if (route.providerDisabled) return '这条中转线路依赖的来源订阅目前被停用了。'
  if (route.transitProxyMissing) return '这条中转线路原先选中的中转节点在当前来源里已经不可用。'
  if (route.egressGroupMissing) return '这条中转线路绑定的落地出口组已经不存在了。'
  if (route.egressProviderMissing) return '这条中转线路依赖的落地出口 provider 已经不存在了。'
  if (route.egressProviderDisabled) return '这条中转线路依赖的落地出口 provider 目前被停用了。'
  if (route.landingMissing) return '这条中转线路绑定的落地节点已经被删除了。'
  if (route.landingDisabled) return '这条中转线路绑定的落地节点目前被停用了。'
  if (route.status === 'disabled') return '这条中转线路当前被手动停用，监听器即使绑定也不会继续使用。'
  return ''
}

export default function TransitRoutes() {
  const { data: routes, mutate: mutateRoutes } = useSWR('/api/transit-routes', api.getTransitRoutes)
  const { data: providers, mutate: mutateProviders } = useSWR('/api/providers', api.getProviders)
  const { data: groups, mutate: mutateGroups } = useSWR('/api/groups', api.getGroups)

  const providerList: ProviderListItem[] = providers || []
  const groupList: GroupView[] = groups || []
  const groupNames = groupList.map((group) => group.name)

  const mutateAll = () => {
    void mutateRoutes()
    void mutateProviders()
    void mutateGroups()
  }

  const [showAddRoute, setShowAddRoute] = useState(false)
  const [showEditRoute, setShowEditRoute] = useState(false)
  const [routeForm, setRouteForm] = useState<TransitRouteForm>({ ...emptyRouteForm })
  const [editingRouteName, setEditingRouteName] = useState('')
  const [routeSubmitting, setRouteSubmitting] = useState(false)
  const [testingRouteName, setTestingRouteName] = useState('')
  const [pendingDeleteRouteName, setPendingDeleteRouteName] = useState<string | null>(null)
  const [deletingRoute, setDeletingRoute] = useState(false)

  const selectedProvider = providerList.find((provider) => provider.name === routeForm.upstream_provider)
  const availableNodes = selectedProvider?.nodes || []

  const resetForm = () => {
    setRouteForm({ ...emptyRouteForm })
    setEditingRouteName('')
  }

  const openAddRoute = () => {
    resetForm()
    setShowAddRoute(true)
  }

  const openEditRoute = (route: TransitRouteView) => {
    setEditingRouteName(route.name)
    setRouteForm({
      name: route.name,
      upstream_provider: route.upstreamProvider,
      upstream_proxy_name: route.upstreamProxyName,
      egress_group: route.egressGroup,
      notes: route.notes || '',
      enabled: route.enabled !== false,
    })
    setShowEditRoute(true)
  }

  const handleProviderChange = (providerName: string) => {
    const provider = providerList.find((item) => item.name === providerName)
    const nextNodeNames = new Set((provider?.nodes || []).map((node) => node.name))
    setRouteForm((current) => ({
      ...current,
      upstream_provider: providerName,
      upstream_proxy_name: nextNodeNames.has(current.upstream_proxy_name) ? current.upstream_proxy_name : '',
    }))
  }

  const handleSave = async (mode: 'add' | 'edit') => {
    if (!routeForm.name.trim() && mode === 'add') {
      toast.error('中转线路名称不能为空。')
      return
    }
    if (!routeForm.upstream_provider || !routeForm.upstream_proxy_name || !routeForm.egress_group) {
      toast.error('中转来源、中转节点和落地出口组都需要填写。')
      return
    }

    setRouteSubmitting(true)
    try {
      const payload: AddTransitRouteRequest | UpdateTransitRouteRequest = {
        ...(mode === 'add' ? { name: routeForm.name.trim() } : {}),
        upstream_provider: routeForm.upstream_provider,
        upstream_proxy_name: routeForm.upstream_proxy_name,
        egress_group: routeForm.egress_group,
        notes: routeForm.notes.trim(),
        enabled: routeForm.enabled,
      }
      const response = mode === 'add'
        ? await api.addTransitRoute(payload as AddTransitRouteRequest)
        : await api.updateTransitRoute(editingRouteName, payload as UpdateTransitRouteRequest)
      if (!response.ok) {
        throw new Error(mode === 'add' ? '创建失败' : '保存失败')
      }
      mutateAll()
      if (mode === 'add') {
        setShowAddRoute(false)
        toast.success('中转线路已创建。')
      } else {
        setShowEditRoute(false)
        toast.success('中转线路已更新。')
      }
      resetForm()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : mode === 'add' ? '创建失败' : '保存失败')
    } finally {
      setRouteSubmitting(false)
    }
  }

  const handleToggleRoute = async (route: TransitRouteView, enabled: boolean) => {
    try {
      const response = await api.updateTransitRoute(route.name, { enabled })
      if (!response.ok) throw new Error('更新失败')
      mutateAll()
      toast.success(enabled ? `已启用中转线路“${route.name}”。` : `已停用中转线路“${route.name}”。`)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '更新失败')
    }
  }

  const handleDeleteRoute = async () => {
    if (!pendingDeleteRouteName) return
    setDeletingRoute(true)
    try {
      const response = await api.deleteTransitRoute(pendingDeleteRouteName)
      if (!response.ok) throw new Error('删除失败')
      mutateAll()
      toast.success(`已删除中转线路“${pendingDeleteRouteName}”。`)
      setPendingDeleteRouteName(null)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '删除失败')
    } finally {
      setDeletingRoute(false)
    }
  }

  const handleHealthcheck = async (route: TransitRouteView) => {
    setTestingRouteName(route.name)
    try {
      const result = await api.healthcheckTransitRoute(route.name)
      mutateAll()
      if (!result.ok) throw new Error(result.error || '检测失败')
      const delay = result.route?.lastTestDelay
      toast.success(Number.isFinite(delay) ? `检测完成，延迟 ${delay}ms。` : '检测完成。')
    } catch (err) {
      mutateAll()
      toast.error(err instanceof ApiError ? err.message : '检测失败')
    } finally {
      setTestingRouteName('')
    }
  }

  const canCreateRoute = providerList.length > 0 && groupNames.length > 0

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Transit Chain"
        title="把固定上游节点和本地出口组拼成一条中转链路"
        description="中转线路适合“入口业务先走固定上游节点，再接你现有出口线路”的场景。监听器绑定后，就不再直接使用普通出口组，而是走这条拼好的链路。"
        actions={
          <Button onClick={openAddRoute} disabled={!canCreateRoute}>
            <Plus className="h-4 w-4" />
            添加中转线路
          </Button>
        }
      />

      <NoticeCard
        icon={<GitBranch className="h-4 w-4" />}
        title="什么时候值得用中转线路"
        description="当你需要把某类流量先送到指定节点，再接上现有出口组统一出站时，就适合单独建一条中转线路。比如固定香港入口，再通过已有的美国出口组落地。"
      />

      {!canCreateRoute && (
        <NoticeCard
          tone="warning"
          icon={<AlertTriangle className="h-4 w-4" />}
          title="还不能创建中转线路"
          description="请先准备至少一个来源订阅或单节点，并且至少有一条可用出口线路，这里才能完成中转链路配置。"
        />
      )}

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">中转线路列表</h2>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          {!routes ? (
            <p className="text-sm text-zinc-500">加载中...</p>
          ) : routes.length === 0 ? (
            <EmptyStateCard
              className="lg:col-span-2"
              icon={<GitBranch className="h-5 w-5" />}
              title="还没有任何中转线路"
              description="先把一条上游固定节点和一条本地出口组组合起来，后面监听器就能切换到中转模式。"
              action={
                <Button onClick={openAddRoute} disabled={!canCreateRoute}>
                  <Plus className="h-4 w-4" />
                  添加第一条中转线路
                </Button>
              }
            />
          ) : (
            routes.map((route) => {
              const warning = buildTransitWarning(route)
              const degraded = route.status !== 'configured'
              const failed = route.lastTestStatus === 'failed'
              const lastTestDelay = typeof route.lastTestDelay === 'number' ? route.lastTestDelay : null
              return (
                <Card key={route.name} className={degraded ? 'border-amber-200 bg-amber-50/30' : ''}>
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <CardTitle className="flex items-center gap-2 text-base">
                          <GitBranch className="h-4 w-4 text-zinc-400" />
                          {route.name}
                        </CardTitle>
                        <CardDescription className="mt-1">{route.upstreamProvider} {'->'} {route.egressGroup}</CardDescription>
                      </div>
                      <div className="flex items-center gap-2">
                        {route.status !== 'configured' && (
                          <div className="rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700">
                            {route.status === 'disabled' ? '停用' : route.status === 'orphaned' ? '孤立' : '降级'}
                          </div>
                        )}
                        <Switch checked={route.enabled !== false} onCheckedChange={(checked) => handleToggleRoute(route, checked)} />
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="space-y-2 rounded-lg border bg-zinc-50 p-3 text-sm">
                      <div className="flex justify-between gap-3">
                        <span className="text-zinc-500">中转来源</span>
                        <span className="font-medium text-zinc-900">{route.upstreamProvider}</span>
                      </div>
                      <div className="flex justify-between gap-3">
                        <span className="text-zinc-500">中转节点</span>
                        <span className="max-w-[190px] truncate font-medium text-zinc-900" title={route.upstreamProxyName}>{route.upstreamProxyName}</span>
                      </div>
                      <div className="flex justify-between gap-3">
                        <span className="text-zinc-500">落地出口组</span>
                        <span className="font-medium text-zinc-900">{route.egressGroup}</span>
                      </div>
                      <div className="flex justify-between gap-3">
                        <span className="text-zinc-500">当前出口</span>
                        <span className="max-w-[190px] truncate font-medium text-zinc-900" title={route.currentProxy || '暂未确定'}>{route.currentProxy || '暂未确定'}</span>
                      </div>
                      <div className="flex justify-between gap-3">
                        <span className="text-zinc-500">实际路径</span>
                        <span className="text-right text-zinc-900">{route.routeSummary || `${route.upstreamProxyName} -> ${route.egressGroup}`}</span>
                      </div>
                    </div>

                    {route.notes ? (
                      <p className="rounded border bg-zinc-50 p-2 text-xs text-zinc-600">
                        备注：{route.notes}
                      </p>
                    ) : null}

                    <div className={`rounded-lg border p-3 text-sm ${failed ? 'border-amber-200 bg-amber-50/80' : 'bg-zinc-50'}`}>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-zinc-500">最后检测</span>
                        <span className="text-xs font-medium text-zinc-900">{formatTimestamp(route.lastTestedAt)}</span>
                      </div>
                      <div className="mt-2 flex items-center justify-between gap-3">
                        <span className="text-zinc-500">检测结果</span>
                        <span className={`text-xs font-medium ${route.lastTestStatus === 'success' ? 'text-emerald-700' : failed ? 'text-amber-700' : 'text-zinc-500'}`}>
                          {route.lastTestStatus === 'success' ? '成功' : failed ? '失败' : '未检测'}
                        </span>
                      </div>
                      {lastTestDelay !== null && lastTestDelay > 0 ? (
                        <div className="mt-2 flex items-center justify-between gap-3">
                          <span className="text-zinc-500">链路延迟</span>
                          <span className="text-xs font-medium text-zinc-900">{lastTestDelay} ms</span>
                        </div>
                      ) : null}
                      {route.lastTestMessage ? (
                        <div className={`mt-2 flex items-start gap-2 text-xs ${failed ? 'text-amber-700' : 'text-zinc-600'}`}>
                          {failed ? <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" /> : null}
                          <p>{route.lastTestMessage}</p>
                        </div>
                      ) : null}
                    </div>

                    {warning ? (
                      <div className="flex items-start gap-2 rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-700">
                        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                        <p>{warning}</p>
                      </div>
                    ) : null}

                    <div className="flex items-center gap-2 border-t pt-2">
                      <Button variant="ghost" size="sm" disabled={route.status !== 'configured' || testingRouteName === route.name} onClick={() => handleHealthcheck(route)}>
                        <RefreshCw className={`mr-1 h-3.5 w-3.5 ${testingRouteName === route.name ? 'animate-spin' : ''}`} />
                        {testingRouteName === route.name ? '检测中...' : '检测线路'}
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => openEditRoute(route)}>
                        <Pencil className="mr-1 h-3.5 w-3.5" />
                        编辑
                      </Button>
                      <Button variant="outline" size="sm" className="text-red-600 hover:bg-red-50 hover:text-red-700" onClick={() => setPendingDeleteRouteName(route.name)}>
                        <Trash2 className="mr-1 h-3.5 w-3.5" />
                        删除
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )
            })
          )}
        </div>
      </section>

      <Dialog open={showAddRoute} onOpenChange={setShowAddRoute}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加中转线路</DialogTitle>
            <DialogDescription>配置“中转来源节点 {'->'} 落地出口组”的链路，后续监听器就能直接绑定这条线路。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>线路名称</Label>
              <Input placeholder="例如：hk-transit-us" value={routeForm.name} onChange={(event) => setRouteForm({ ...routeForm, name: event.target.value })} />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>中转来源</Label>
                <Select value={routeForm.upstream_provider} onValueChange={handleProviderChange}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择来源订阅或单节点" />
                  </SelectTrigger>
                  <SelectContent>
                    {providerList.map((provider) => (
                      <SelectItem key={provider.name} value={provider.name}>{provider.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>中转节点</Label>
                <Select value={routeForm.upstream_proxy_name} onValueChange={(value) => setRouteForm({ ...routeForm, upstream_proxy_name: value })}>
                  <SelectTrigger>
                    <SelectValue placeholder={routeForm.upstream_provider ? '选择具体节点' : '先选择中转来源'} />
                  </SelectTrigger>
                  <SelectContent>
                    {availableNodes.map((node) => (
                      <SelectItem key={node.name} value={node.name}>{node.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>落地出口组</Label>
              <Select value={routeForm.egress_group} onValueChange={(value) => setRouteForm({ ...routeForm, egress_group: value })}>
                <SelectTrigger>
                  <SelectValue placeholder="选择一条落地出口组" />
                </SelectTrigger>
                <SelectContent>
                  {groupNames.map((name) => (
                    <SelectItem key={name} value={name}>{name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>备注</Label>
              <Input placeholder="例如：香港上游固定节点，再接美国出口组统一落地" value={routeForm.notes} onChange={(event) => setRouteForm({ ...routeForm, notes: event.target.value })} />
            </div>
            <div className="flex items-center gap-3">
              <Switch checked={routeForm.enabled} onCheckedChange={(checked) => setRouteForm({ ...routeForm, enabled: checked })} />
              <Label>创建后立即启用</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddRoute(false)}>取消</Button>
            <Button onClick={() => void handleSave('add')} disabled={routeSubmitting}>{routeSubmitting ? '创建中...' : '创建线路'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showEditRoute} onOpenChange={setShowEditRoute}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑中转线路：{editingRouteName}</DialogTitle>
            <DialogDescription>你可以修改上游来源、固定节点、落地出口组和备注。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>线路名称</Label>
              <Input value={routeForm.name} disabled />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>中转来源</Label>
                <Select value={routeForm.upstream_provider} onValueChange={handleProviderChange}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择来源订阅或单节点" />
                  </SelectTrigger>
                  <SelectContent>
                    {providerList.map((provider) => (
                      <SelectItem key={provider.name} value={provider.name}>{provider.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>中转节点</Label>
                <Select value={routeForm.upstream_proxy_name} onValueChange={(value) => setRouteForm({ ...routeForm, upstream_proxy_name: value })}>
                  <SelectTrigger>
                    <SelectValue placeholder={routeForm.upstream_provider ? '选择具体节点' : '先选择中转来源'} />
                  </SelectTrigger>
                  <SelectContent>
                    {availableNodes.map((node) => (
                      <SelectItem key={node.name} value={node.name}>{node.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>落地出口组</Label>
              <Select value={routeForm.egress_group} onValueChange={(value) => setRouteForm({ ...routeForm, egress_group: value })}>
                <SelectTrigger>
                  <SelectValue placeholder="选择一条落地出口组" />
                </SelectTrigger>
                <SelectContent>
                  {groupNames.map((name) => (
                    <SelectItem key={name} value={name}>{name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>备注</Label>
              <Input value={routeForm.notes} onChange={(event) => setRouteForm({ ...routeForm, notes: event.target.value })} />
            </div>
            <div className="flex items-center gap-3">
              <Switch checked={routeForm.enabled} onCheckedChange={(checked) => setRouteForm({ ...routeForm, enabled: checked })} />
              <Label>保持启用</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditRoute(false)}>取消</Button>
            <Button onClick={() => void handleSave('edit')} disabled={routeSubmitting}>{routeSubmitting ? '保存中...' : '保存修改'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(pendingDeleteRouteName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteRouteName(null)
        }}
        title="删除这条中转线路后会发生什么？"
        description={pendingDeleteRouteName ? `“${pendingDeleteRouteName}”删除后，所有绑定它的本地入口都会失去目标链路，删除前请先调整监听器。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingRoute}
        onConfirm={handleDeleteRoute}
      />
    </div>
  )
}
