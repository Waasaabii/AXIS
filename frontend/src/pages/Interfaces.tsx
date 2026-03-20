import { useState } from 'react'
import useSWR from 'swr'
import { fetcher } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { toast } from 'sonner'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Trash2, Pencil, AlertTriangle, RefreshCw, FolderPlus, HelpCircle, ChevronDown } from 'lucide-react'

/* ── Filter presets ── */
const FILTER_PRESETS = [
  { label: '🇭🇰 香港', value: '(?i)港|hk|hong ?kong' },
  { label: '🇺🇸 美国', value: '(?i)美|us|united ?states|america' },
  { label: '🇯🇵 日本', value: '(?i)日|jp|japan|tokyo' },
  { label: '🇸🇬 新加坡', value: '(?i)新|sg|singapore' },
  { label: '🇬🇧 英国', value: '(?i)英|uk|gb|britain|london' },
  { label: '🇩🇪 德国', value: '(?i)德|de|germany|frankfurt' },
  { label: '🇹🇼 台湾', value: '(?i)台|tw|taiwan' },
  { label: '🇰🇷 韩国', value: '(?i)韩|kr|korea|seoul' },
]

function FilterField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1">
        <Label>节点筛选 (正则)</Label>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className="inline-flex items-center justify-center rounded-full text-zinc-400 hover:text-zinc-600"
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-72 p-3 text-xs text-zinc-600 leading-relaxed shadow-lg">
            使用正则表达式筛选匹配节点。<br/>
            • <code className="bg-zinc-100 px-1 rounded text-zinc-800">(?i)</code> 表示忽略大小写<br/>
            • 用 <code className="bg-zinc-100 px-1 rounded text-zinc-800">|</code> 分隔多个关键词<br/>
            • 示例: <code className="bg-zinc-100 px-1 rounded text-zinc-800">(?i)港|hk|hong ?kong</code><br/>
            匹配含 港、HK、Hong Kong 的节点
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="relative">
        <Input 
          className="font-mono text-sm pr-10"
          placeholder="留空为全选 (不筛选)" 
          value={value} 
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value)} 
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button 
              type="button" 
              className="absolute right-0 top-0 h-full px-3 text-zinc-400 hover:text-zinc-600 flex items-center justify-center"
              title="选择预设地区"
            >
              <ChevronDown className="h-4 w-4" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <div className="px-2 py-1.5 text-xs font-semibold text-zinc-500">地区预设</div>
            {FILTER_PRESETS.map(p => (
              <DropdownMenuItem key={p.value} onSelect={() => onChange(p.value)} className="cursor-pointer">
                {p.label}
              </DropdownMenuItem>
            ))}
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => onChange('')} className="cursor-pointer">
              <span>🧹 清空 (全选)</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

/* ── Egress Group form ── */
interface GroupForm { name: string; provider: string; mode: string; filter: string }
const emptyGroupForm: GroupForm = { name: '', provider: '', mode: 'manual', filter: '' }

export default function Interfaces() {
  const { data: groups, mutate: mutateGroups } = useSWR('/api/groups', fetcher)
  const { data: configData, mutate: mutateConfig } = useSWR('/api/config', fetcher)

  const subscriptionNames: string[] = configData?.config?.subscriptions?.map((s: any) => s.name) || []

  const mutateAll = () => { mutateGroups(); mutateConfig() }

  /* ── Group state ── */
  const [showAddGroup, setShowAddGroup] = useState(false)
  const [showEditGroup, setShowEditGroup] = useState(false)
  const [groupForm, setGroupForm] = useState<GroupForm>({ ...emptyGroupForm })
  const [editingGroupName, setEditingGroupName] = useState('')
  const [groupSubmitting, setGroupSubmitting] = useState(false)

  /* ════════════════════  Group handlers  ════════════════════ */

  const handleAddGroup = async () => {
    if (!groupForm.name.trim() || !groupForm.provider) { toast.error('名称和订阅源不能为空'); return }
    setGroupSubmitting(true)
    try {
      const res = await fetch('/api/egress-groups', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: groupForm.name.trim(), provider: groupForm.provider, mode: groupForm.mode, filter: groupForm.filter }) })
      const data = await res.json()
      if (!data.ok) throw new Error(data.error || '添加失败')
      mutateAll()
      setShowAddGroup(false)
      setGroupForm({ ...emptyGroupForm })
      toast.success('出口组已添加')
    } catch (err: any) { toast.error(err.message || '添加失败') } finally { setGroupSubmitting(false) }
  }

  const openEditGroup = (g: any) => {
    setEditingGroupName(g.name)
    setGroupForm({ name: g.name, provider: g.provider, mode: g.mode, filter: g.filter || '' })
    setShowEditGroup(true)
  }

  const handleEditGroup = async () => {
    if (!groupForm.provider) { toast.error('订阅源不能为空'); return }
    setGroupSubmitting(true)
    try {
      const res = await fetch(`/api/egress-groups/${encodeURIComponent(editingGroupName)}/update`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ provider: groupForm.provider, mode: groupForm.mode, filter: groupForm.filter }) })
      const data = await res.json()
      if (!data.ok) throw new Error(data.error || '修改失败')
      mutateAll()
      setShowEditGroup(false)
      toast.success('出口组已更新')
    } catch (err: any) { toast.error(err.message || '修改失败') } finally { setGroupSubmitting(false) }
  }

  const handleDeleteGroup = async (name: string) => {
    if (!confirm(`确认删除出口组 "${name}"？\n引用该出口组的端口不会被删除，但会变为孤立状态。`)) return
    try {
      const res = await fetch(`/api/egress-groups/${encodeURIComponent(name)}`, { method: 'DELETE' })
      const data = await res.json()
      if (!data.ok) throw new Error(data.error || '删除失败')
      mutateAll()
      toast.success(`出口组 ${name} 已删除`)
    } catch (err: any) { toast.error(err.message || '删除失败') }
  }

  const handleSelectGroup = async (groupName: string, proxyName: string) => {
    try {
      const res = await fetch(`/api/groups/${encodeURIComponent(groupName)}/select`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ proxyName }) })
      if (!res.ok) throw new Error('切换失败')
      mutateGroups()
      toast.success(`出口组 ${groupName} 已切换至 ${proxyName}`)
    } catch { toast.error('切换失败') }
  }

  const handleHealthcheck = async (groupName: string) => {
    try {
      const res = await fetch(`/api/groups/${encodeURIComponent(groupName)}/healthcheck`, { method: 'POST' })
      if (!res.ok) throw new Error('健康检查失败')
      mutateGroups()
      toast.success(`出口组 ${groupName} 已执行健康检查`)
    } catch { toast.error('健康检查失败') }
  }

  /* ════════════════════  Render  ════════════════════ */

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">出口管理</h1>
        <p className="text-zinc-500">配置逻辑出口组。</p>
      </div>

      {/* ──────── Egress Groups Section ──────── */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">逻辑出口组</h2>
          <Button size="sm" onClick={() => { setGroupForm({ ...emptyGroupForm }); setShowAddGroup(true) }} disabled={subscriptionNames.length === 0}>
            <FolderPlus className="h-4 w-4 mr-1" /> 添加出口组
          </Button>
        </div>

        {subscriptionNames.length === 0 && (
          <div className="border rounded-lg p-4 bg-amber-50 text-amber-700 text-sm">
            请先在「订阅管理」页面添加订阅源后，再创建出口组。
          </div>
        )}

        <div className="grid gap-4 lg:grid-cols-2">
          {!groups ? (
            <p className="text-sm text-zinc-500">加载中...</p>
          ) : groups.length === 0 ? (
            <p className="text-sm text-zinc-500 border rounded-lg p-8 text-center bg-zinc-50 lg:col-span-2">
              还没有配置任何出口组。
            </p>
          ) : (
            groups.map((g: any) => (
              <Card key={g.name} className={g.providerMissing || g.providerDisabled || g.candidateCount === 0 ? 'border-red-100 bg-gradient-to-b from-white to-red-50/30 shadow-[0_8px_30px_-6px_rgba(239,68,68,0.15)] relative z-10 transition-shadow' : 'shadow-sm'}>
                <CardHeader className="pb-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        {g.name}
                        {g.providerMissing && <span className="text-xs bg-red-100 text-red-700 px-1.5 py-0.5 rounded font-normal">订阅已删除</span>}
                        {g.providerDisabled && <span className="text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded font-normal">订阅已停用</span>}
                      </CardTitle>
                      <CardDescription className="mt-1">订阅源: {g.provider}</CardDescription>
                    </div>
                    <div className="px-2 py-1 text-xs rounded-full bg-zinc-100 border text-zinc-600 font-medium">{g.mode}</div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-2 text-xs text-zinc-500 mb-3 flex-wrap">
                    <span className="bg-zinc-100 px-2 py-1 rounded">候选: {g.candidateCount}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded truncate max-w-[150px]" title={g.current || '未选择'}>当前: {g.current || '未选择'}</span>
                  </div>
                  {g.filter && <p className="text-xs text-zinc-500 font-mono mb-3 bg-zinc-50 p-2 rounded">筛选: {g.filter}</p>}

                  {(g.providerMissing || g.providerDisabled) && (
                    <div className="flex items-start gap-2 text-xs text-amber-600 bg-amber-50 p-2 rounded mb-3">
                      <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                      <p>{g.providerMissing ? '关联的订阅源已删除，请编辑重新绑定。' : '关联的订阅源已停用，启用后恢复。'}</p>
                    </div>
                  )}

                  {!g.providerMissing && !g.providerDisabled && (
                    <div className="flex items-center gap-2 mb-3">
                      <Select 
                        value={g.candidateCount > 0 ? g.current : undefined} 
                        onValueChange={(val: string) => handleSelectGroup(g.name, val)}
                        disabled={g.candidateCount === 0}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder={g.candidateCount === 0 ? "无可用节点匹配" : "选择节点"} />
                        </SelectTrigger>
                        <SelectContent>
                          {g.candidates?.map((c: any) => <SelectItem key={c.name} value={c.name}>{c.name} ({c.type})</SelectItem>)}
                        </SelectContent>
                      </Select>
                      <Button variant="secondary" size="icon" onClick={() => handleHealthcheck(g.name)} title="健康检查" disabled={g.candidateCount === 0}>
                        <RefreshCw className={`h-4 w-4 ${g.candidateCount === 0 ? 'opacity-50' : ''}`} />
                      </Button>
                    </div>
                  )}

                  <div className="flex items-center gap-2 pt-2 border-t">
                    <Button variant="outline" size="sm" onClick={() => openEditGroup(g)}><Pencil className="h-3.5 w-3.5 mr-1" /> 编辑</Button>
                    <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => handleDeleteGroup(g.name)}><Trash2 className="h-3.5 w-3.5 mr-1" /> 删除</Button>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </section>

      {/* ──────── Add Group Dialog ──────── */}
      <Dialog open={showAddGroup} onOpenChange={setShowAddGroup}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加出口组</DialogTitle>
            <DialogDescription>创建逻辑出口组，绑定到订阅源。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>出口组名称</Label>
              <Input placeholder="例如: egress-hk-manual" value={groupForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setGroupForm({ ...groupForm, name: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label>订阅源</Label>
              <Select value={groupForm.provider} onValueChange={(val: string) => setGroupForm({ ...groupForm, provider: val })}>
                <SelectTrigger><SelectValue placeholder="选择订阅源" /></SelectTrigger>
                <SelectContent>{subscriptionNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>模式</Label>
                <Select value={groupForm.mode} onValueChange={(val: string) => setGroupForm({ ...groupForm, mode: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="manual">manual (手动)</SelectItem>
                    <SelectItem value="auto">auto (自动)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <FilterField value={groupForm.filter} onChange={(v) => setGroupForm({ ...groupForm, filter: v })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddGroup(false)}>取消</Button>
            <Button onClick={handleAddGroup} disabled={groupSubmitting}>{groupSubmitting ? '添加中...' : '添加'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ──────── Edit Group Dialog ──────── */}
      <Dialog open={showEditGroup} onOpenChange={setShowEditGroup}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑出口组: {editingGroupName}</DialogTitle>
            <DialogDescription>修改订阅源绑定、模式或筛选规则。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>订阅源</Label>
              <Select value={groupForm.provider} onValueChange={(val: string) => setGroupForm({ ...groupForm, provider: val })}>
                <SelectTrigger><SelectValue placeholder="选择订阅源" /></SelectTrigger>
                <SelectContent>{subscriptionNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>模式</Label>
                <Select value={groupForm.mode} onValueChange={(val: string) => setGroupForm({ ...groupForm, mode: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="manual">manual (手动)</SelectItem>
                    <SelectItem value="auto">auto (自动)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <FilterField value={groupForm.filter} onChange={(v) => setGroupForm({ ...groupForm, filter: v })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditGroup(false)}>取消</Button>
            <Button onClick={handleEditGroup} disabled={groupSubmitting}>{groupSubmitting ? '保存中...' : '保存'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
