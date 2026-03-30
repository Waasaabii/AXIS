import { useState } from 'react'
import useSWR from 'swr'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { Link2, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/ConfirmDialog'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from '@/components/ui/switch'
import { api, ApiError, type ProviderListItem } from '@/services/api'

function formatSubscriptionType(type: string) {
  switch (type) {
    case 'mihomo-http':
      return '整份 Mihomo 配置'
    case 'clash-http':
      return '节点链接列表'
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
        toast.success(`已刷新“${name}”，当前识别到 ${data.provider?.nodeCount ?? 0} 个节点。`)
      } else {
        toast.error(`刷新没有完成：${data.error || '请稍后再试。'}`)
      }
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '刷新失败，请稍后再试。')
    }
  }

  const handleToggleSubscription = async (name: string, enabled: boolean) => {
    try {
      const data = await api.toggleSubscription(name, { enabled })
      if (!data.ok) throw new Error('操作失败')
      mutateAll()
      toast.success(enabled ? `“${name}”已恢复启用。` : `“${name}”已暂停使用。`)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '操作失败，请稍后再试。')
    }
  }

  const handleAddSubscription = async () => {
    if (!subForm.name.trim() || !subForm.url.trim()) {
      toast.error('订阅名称和链接都需要填写。')
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
      toast.success('订阅已经添加，AXIS 会开始拉取节点。')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '添加订阅失败，请检查链接后重试。')
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
      toast.success(`“${pendingDeleteName}”已删除。`)
      setPendingDeleteName(null)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '删除订阅失败，请稍后再试。')
    } finally {
      setDeletingSubscription(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Subscriptions"
        title="先把你的节点带进来"
        description="订阅是 AXIS 获取节点的入口。导入后，你才能在后面的页面整理出口线路、绑定本地入口，或者继续配置落地路径。"
        actions={
          <Button onClick={() => setShowAddSub(true)}>
            <Plus className="h-4 w-4" />
            添加订阅
          </Button>
        }
      />

      <NoticeCard
        icon={<Link2 className="h-4 w-4" />}
        title="怎么判断该填哪种订阅"
        description="如果服务商给你的是完整的 Mihomo/Clash 配置，选“整份 Mihomo 配置”；如果给的是节点链接集合，选“节点链接列表”。绝大多数情况下，直接保留默认选项即可。"
      />

      {!providers ? (
        <div className="text-sm text-zinc-500">正在读取订阅列表...</div>
      ) : providers.length === 0 ? (
        <EmptyStateCard
          icon={<Link2 className="h-5 w-5" />}
          title="还没有任何订阅"
          description="先添加一个订阅链接，让 AXIS 拿到你的节点。完成这一步后，你才可以去整理出口线路和本地入口。"
          action={
            <Button onClick={() => setShowAddSub(true)}>
              <Plus className="h-4 w-4" />
              现在添加订阅
            </Button>
          }
        />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          {providers.map((provider: ProviderListItem) => (
            <Card key={provider.name} className={`border-zinc-200 shadow-sm ${!provider.enabled ? 'opacity-75' : ''}`}>
              <CardHeader className="gap-3 pb-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 space-y-1">
                    <CardTitle className="text-base text-zinc-950">{provider.name}</CardTitle>
                    <CardDescription className="truncate text-sm text-zinc-500" title={provider.urlMasked}>
                      {provider.urlMasked}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 px-2 py-1">
                    <span className="text-xs text-zinc-500">启用</span>
                    <Switch
                      checked={provider.enabled}
                      onCheckedChange={(checked: boolean) => handleToggleSubscription(provider.name, checked)}
                    />
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-wrap gap-2 text-xs text-zinc-600">
                  <InfoPill label={`节点 ${provider.nodeCount}`} />
                  <InfoPill label={`刷新间隔 ${provider.interval}s`} />
                  <InfoPill label={formatSubscriptionType(provider.type)} />
                </div>

                <div className="rounded-2xl border border-zinc-200 bg-zinc-50/80 px-4 py-3 text-sm text-zinc-600">
                  <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                    <span>最近刷新</span>
                    <span className="font-medium text-zinc-900">
                      {provider.refreshedAt ? format(new Date(provider.refreshedAt), 'PP HH:mm:ss', { locale: zhCN }) : '还没有刷新记录'}
                    </span>
                  </div>
                  {provider.lastError ? (
                    <div className="mt-3 rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                      最近一次失败原因：{provider.lastError}
                    </div>
                  ) : null}
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => handleRefreshProvider(provider.name)} disabled={!provider.enabled}>
                    <RefreshCw className="h-3.5 w-3.5" />
                    重新拉取节点
                  </Button>
                  <Button variant="outline" size="sm" className="text-red-600 hover:bg-red-50 hover:text-red-700" onClick={() => setPendingDeleteName(provider.name)}>
                    <Trash2 className="h-3.5 w-3.5" />
                    删除订阅
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={showAddSub} onOpenChange={setShowAddSub}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>添加一个新的订阅</DialogTitle>
            <DialogDescription>填写名称和订阅链接即可。添加成功后，AXIS 会开始拉取节点。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <FieldBlock
              label="订阅名称"
              hint="这是给你自己看的名字。建议用服务商名称或用途命名。"
              input={<Input placeholder="例如：主机场、备用订阅、流媒体专线" value={subForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubForm({ ...subForm, name: e.target.value })} />}
            />
            <FieldBlock
              label="订阅链接"
              hint="通常由服务商提供，形如 https://..."
              input={<Input placeholder="https://..." value={subForm.url} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubForm({ ...subForm, url: e.target.value })} />}
            />
            <div className="grid gap-4 md:grid-cols-2">
              <FieldBlock
                label="订阅类型"
                hint="如果不确定，先保留默认选项。"
                input={
                  <Select value={subForm.type} onValueChange={(val: string) => setSubForm({ ...subForm, type: val })}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="mihomo-http">整份 Mihomo 配置</SelectItem>
                      <SelectItem value="clash-http">节点链接列表</SelectItem>
                    </SelectContent>
                  </Select>
                }
              />
              <FieldBlock
                label="刷新间隔"
                hint="单位是秒，默认 3600 秒即可。"
                input={<Input type="number" value={subForm.interval} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubForm({ ...subForm, interval: e.target.value })} />}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddSub(false)}>取消</Button>
            <Button onClick={handleAddSubscription} disabled={addingSubscription}>
              {addingSubscription ? '添加中...' : '添加并开始拉取'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(pendingDeleteName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteName(null)
        }}
        title="删除这个订阅后会发生什么？"
        description={pendingDeleteName ? `“${pendingDeleteName}”删除后，相关出口线路和本地入口不会一起被删，但它们会暂时失去可用节点，需要你重新绑定或补上新的订阅。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingSubscription}
        onConfirm={handleDeleteSubscription}
      />
    </div>
  )
}

function InfoPill({ label }: { label: string }) {
  return <span className="rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1">{label}</span>
}

function FieldBlock({ label, hint, input }: { label: string; hint: string; input: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <Label>{label}</Label>
        <p className="text-xs leading-5 text-zinc-500">{hint}</p>
      </div>
      {input}
    </div>
  )
}
