import { useState } from 'react'
import useSWR from 'swr'
import { api, ApiError, type ProviderListItem } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { toast } from 'sonner'
import { RefreshCw, Plus, Trash2 } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

function formatSubscriptionType(type: string) {
  switch (type) {
    case 'mihomo-http':
      return 'Mihomo 配置订阅'
    case 'clash-http':
      return '节点链接订阅'
    default:
      return type
  }
}

export default function Subscriptions() {
  const { data: providers, mutate: mutateProviders } = useSWR('/api/providers', api.getProviders)
  const { mutate: mutateConfig } = useSWR('/api/config', api.getConfig)

  const [showAddSub, setShowAddSub] = useState(false)
  const [subForm, setSubForm] = useState({ name: '', url: '', type: 'mihomo-http', interval: '3600' })
  const [addingSubscription, setAddingSubscription] = useState(false)
  const [pendingDeleteName, setPendingDeleteName] = useState<string | null>(null)
  const [deletingSubscription, setDeletingSubscription] = useState(false)

  const mutateAll = () => { mutateProviders(); mutateConfig() }

  const handleRefreshProvider = async (name: string) => {
    try {
      const data = await api.refreshProvider(name)
      mutateProviders()
      if (data.ok) {
        toast.success(`订阅 ${name} 已刷新，节点数: ${data.provider?.nodeCount ?? 0}`)
      } else {
        toast.error(`刷新失败: ${data.error || '未知错误'}`)
      }
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '刷新失败')
    }
  }

  const handleToggleSubscription = async (name: string, enabled: boolean) => {
    try {
      const data = await api.toggleSubscription(name, { enabled })
      if (!data.ok) throw new Error('操作失败')
      mutateAll()
      toast.success(enabled ? `订阅 ${name} 已启用` : `订阅 ${name} 已停用`)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '操作失败')
    }
  }

  const handleAddSubscription = async () => {
    if (!subForm.name.trim() || !subForm.url.trim()) {
      toast.error('名称和 URL 不能为空')
      return
    }
    setAddingSubscription(true)
    try {
      const data = await api.addSubscription({
        name: subForm.name.trim(),
        url: subForm.url.trim(),
        type: subForm.type,
        interval: Number(subForm.interval) || 3600,
      })
      if (!data.ok) throw new Error('添加失败')
      mutateAll()
      setShowAddSub(false)
      setSubForm({ name: '', url: '', type: 'mihomo-http', interval: '3600' })
      toast.success('订阅已添加并正在拉取节点...')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '添加订阅失败')
    } finally {
      setAddingSubscription(false)
    }
  }

  const handleDeleteSubscription = async () => {
    if (!pendingDeleteName) return
    setDeletingSubscription(true)
    try {
      const data = await api.deleteSubscription(pendingDeleteName)
      if (!data.ok) throw new Error('删除失败')
      mutateAll()
      toast.success(`订阅 ${pendingDeleteName} 已删除`)
      setPendingDeleteName(null)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '删除订阅失败')
    } finally {
      setDeletingSubscription(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">订阅管理</h1>
        <p className="text-zinc-500">管理远程代理订阅源，支持同时启用多个订阅。</p>
      </div>

      <div className="flex items-center justify-between">
        <div />
        <Button size="sm" onClick={() => setShowAddSub(true)}>
          <Plus className="h-4 w-4 mr-1" /> 添加订阅
        </Button>
      </div>

      <div className="text-xs text-zinc-500 bg-zinc-50 border rounded-lg px-4 py-2">
        <strong>Mihomo 配置订阅</strong> — 适用于直接返回 Mihomo 配置内容的订阅链接 &nbsp;|&nbsp;
        <strong>节点链接订阅</strong> — 适用于返回节点链接列表的订阅
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        {!providers ? (
          <p className="text-sm text-zinc-500">加载中...</p>
        ) : providers.length === 0 ? (
          <p className="text-sm text-zinc-500 border rounded-lg p-8 text-center bg-zinc-50 md:col-span-2">
            还没有配置任何订阅，点击上方按钮添加。
          </p>
        ) : (
          providers.map((p: ProviderListItem) => (
            <Card key={p.name} className={!p.enabled ? 'opacity-60' : ''}>
              <CardHeader className="pb-3">
                <div className="flex justify-between items-start">
                  <div className="min-w-0 flex-1 mr-2">
                    <CardTitle className="text-base">{p.name}</CardTitle>
                    <CardDescription className="truncate max-w-[280px] mt-1" title={p.urlMasked}>
                      {p.urlMasked}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Switch
                      checked={p.enabled}
                      onCheckedChange={(checked: boolean) => handleToggleSubscription(p.name, checked)}
                    />
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex gap-2 text-xs text-zinc-500 mb-4 flex-wrap">
                  <span className="bg-zinc-100 px-2 py-1 rounded">节点: {p.nodeCount}</span>
                  <span className="bg-zinc-100 px-2 py-1 rounded">间隔: {p.interval}s</span>
                  <span className="bg-zinc-100 px-2 py-1 rounded">{formatSubscriptionType(p.type)}</span>
                </div>
                <div className="text-xs space-y-1 mb-4 text-zinc-600">
                  <p>刷新: {p.refreshedAt ? format(new Date(p.refreshedAt), 'PP HH:mm:ss', { locale: zhCN }) : '未刷新'}</p>
                  {p.lastError && <p className="text-red-500">错误: {p.lastError}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => handleRefreshProvider(p.name)} disabled={!p.enabled}>
                    <RefreshCw className="h-3.5 w-3.5 mr-1" /> 刷新订阅
                  </Button>
                  <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => setPendingDeleteName(p.name)}>
                    <Trash2 className="h-3.5 w-3.5 mr-1" /> 删除
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>

      {/* Add Subscription Dialog */}
      <Dialog open={showAddSub} onOpenChange={setShowAddSub}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加订阅源</DialogTitle>
            <DialogDescription>输入远程订阅的名称和 URL 地址。添加后将自动拉取节点。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>订阅名称</Label>
              <Input placeholder="例如: my-airport" value={subForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubForm({ ...subForm, name: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label>订阅 URL</Label>
              <Input placeholder="https://..." value={subForm.url} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubForm({ ...subForm, url: e.target.value })} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>类型</Label>
                <Select value={subForm.type} onValueChange={(val: string) => setSubForm({ ...subForm, type: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="mihomo-http">Mihomo 配置订阅</SelectItem>
                    <SelectItem value="clash-http">节点链接订阅</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>刷新间隔 (秒)</Label>
                <Input type="number" value={subForm.interval} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubForm({ ...subForm, interval: e.target.value })} />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddSub(false)}>取消</Button>
            <Button onClick={handleAddSubscription} disabled={addingSubscription}>
              {addingSubscription ? '添加中...' : '添加'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(pendingDeleteName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteName(null)
        }}
        title="确认删除订阅？"
        description={pendingDeleteName ? `订阅“${pendingDeleteName}”删除后，关联的出口组和入口不会一起删除，但会暂时失去绑定。` : ''}
        confirmLabel="删除订阅"
        destructive
        confirming={deletingSubscription}
        onConfirm={handleDeleteSubscription}
      />
    </div>
  )
}
