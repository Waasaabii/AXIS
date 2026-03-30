import { useState } from 'react'
import useSWR from 'swr'
import { api, ApiError, type GroupView } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { toast } from 'sonner'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Trash2, Pencil, AlertTriangle, RefreshCw, FolderPlus, HelpCircle, ChevronDown, Network } from 'lucide-react'

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

function formatGroupMode(mode: string) {
  return mode === 'auto' ? '自动选择' : '手动选择'
}

function FilterField({ value, onChange, disabled }: { value: string; onChange: (v: string) => void; disabled?: boolean }) {
  return (
    <div className={`space-y-2 ${disabled ? 'opacity-50 pointer-events-none' : ''}`}>
      <div className="flex items-center gap-1">
        <Label>节点筛选规则</Label>
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
          placeholder="留空表示不过滤，所有节点都可进入这条线路" 
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
  const { data: groups, mutate: mutateGroups } = useSWR('/api/groups', api.getGroups)
  const { data: configData, mutate: mutateConfig } = useSWR('/api/config', api.getConfig)

  const subscriptionNames = configData?.config.subscriptions.map((subscription) => subscription.name) || []

  const mutateAll = () => { mutateGroups(); mutateConfig() }

  /* ── Group state ── */
  const [showAddGroup, setShowAddGroup] = useState(false)
  const [showEditGroup, setShowEditGroup] = useState(false)
  const [groupForm, setGroupForm] = useState<GroupForm>({ ...emptyGroupForm })
  const [editingGroupName, setEditingGroupName] = useState('')
  const [groupSubmitting, setGroupSubmitting] = useState(false)
  const [pendingDeleteGroupName, setPendingDeleteGroupName] = useState<string | null>(null)
  const [deletingGroup, setDeletingGroup] = useState(false)

  /* ════════════════════  Group handlers  ════════════════════ */

  const handleAddGroup = async () => {
    if (!groupForm.name.trim() || !groupForm.provider) { toast.error('名称和订阅源不能为空'); return }
    setGroupSubmitting(true)
    try {
      const data = await api.addEgressGroup({ name: groupForm.name.trim(), provider: groupForm.provider, mode: groupForm.mode, filter: groupForm.filter })
      if (!data.ok) throw new Error('添加失败')
      mutateAll()
      setShowAddGroup(false)
      setGroupForm({ ...emptyGroupForm })
      toast.success('出口线路已创建。')
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '添加失败') } finally { setGroupSubmitting(false) }
  }

  const openEditGroup = (g: GroupView) => {
    setEditingGroupName(g.name)
    setGroupForm({ name: g.name, provider: g.provider, mode: g.mode, filter: g.filter || '' })
    setShowEditGroup(true)
  }

  const handleEditGroup = async () => {
    if (!groupForm.provider) { toast.error('订阅源不能为空'); return }
    setGroupSubmitting(true)
    try {
      const data = await api.updateEgressGroup(editingGroupName, { provider: groupForm.provider, mode: groupForm.mode, filter: groupForm.filter })
      if (!data.ok) throw new Error('修改失败')
      mutateAll()
      setShowEditGroup(false)
      toast.success('出口线路已更新。')
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '修改失败') } finally { setGroupSubmitting(false) }
  }

  const handleDeleteGroup = async () => {
    if (!pendingDeleteGroupName) return
    setDeletingGroup(true)
    try {
      const data = await api.deleteEgressGroup(pendingDeleteGroupName)
      if (!data.ok) throw new Error('删除失败')
      mutateAll()
      toast.success(`已删除出口线路“${pendingDeleteGroupName}”。`)
      setPendingDeleteGroupName(null)
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '删除失败') } finally { setDeletingGroup(false) }
  }

  const handleSelectGroup = async (groupName: string, proxyName: string) => {
    try {
      await api.selectGroup(groupName, { proxyName })
      mutateGroups()
      toast.success(`线路“${groupName}”当前改为使用“${proxyName}”。`)
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '切换失败') }
  }

  const handleHealthcheck = async (groupName: string) => {
    try {
      await api.healthcheckGroup(groupName)
      mutateGroups()
      toast.success(`线路“${groupName}”已经重新检测可用节点。`)
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '健康检查失败') }
  }

  /* ════════════════════  Render  ════════════════════ */

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Routes"
        title="把节点整理成好理解的出口线路"
        description="你可以把同一份订阅里的节点按地区或用途分成不同线路。后面创建本地入口时，只需要选这条线路，不用再面对整份节点列表。"
        actions={
          <Button onClick={() => { setGroupForm({ ...emptyGroupForm }); setShowAddGroup(true) }} disabled={subscriptionNames.length === 0}>
            <FolderPlus className="h-4 w-4" />
            添加出口线路
          </Button>
        }
      />

      <NoticeCard
        icon={<Network className="h-4 w-4" />}
        title="什么时候需要新建一条出口线路"
        description="当你想把节点按地区、用途或稳定性分开管理时，就值得新建一条线路。比如“香港日常”“日本流媒体”“自动选择”这类名字，都比直接面对原始节点列表更好用。"
      />

      {/* ──────── Egress Groups Section ──────── */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">出口线路列表</h2>
        </div>

        {subscriptionNames.length === 0 && (
          <NoticeCard
            tone="warning"
            icon={<AlertTriangle className="h-4 w-4" />}
            title="还不能创建出口线路"
            description="因为系统里还没有可用订阅。先去“订阅与节点”页导入订阅，AXIS 拿到节点后，这里才能继续往下配。"
          />
        )}

        <div className="grid gap-4 lg:grid-cols-2">
          {!groups ? (
            <p className="text-sm text-zinc-500">加载中...</p>
          ) : groups.length === 0 ? (
            <EmptyStateCard
              className="lg:col-span-2"
              icon={<Network className="h-5 w-5" />}
              title="还没有任何出口线路"
              description="先新建一条线路，把节点按地区或用途整理好。后面本地入口就只需要绑定线路，不用直接碰原始节点列表。"
              action={
                <Button onClick={() => { setGroupForm({ ...emptyGroupForm }); setShowAddGroup(true) }} disabled={subscriptionNames.length === 0}>
                  <FolderPlus className="h-4 w-4" />
                  添加第一条线路
                </Button>
              }
            />
          ) : (
            groups.map((g: GroupView) => (
              <Card key={g.name} className={g.providerMissing || g.providerDisabled || g.candidateCount === 0 ? 'border-red-100 bg-gradient-to-b from-white to-red-50/30 shadow-[0_8px_30px_-6px_rgba(239,68,68,0.15)] relative z-10 transition-shadow' : 'shadow-sm'}>
                <CardHeader className="pb-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        {g.name}
                        {g.providerMissing && <span className="text-xs bg-red-100 text-red-700 px-1.5 py-0.5 rounded font-normal">订阅已删除</span>}
                        {g.providerDisabled && <span className="text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded font-normal">订阅已停用</span>}
                      </CardTitle>
                      <CardDescription className="mt-1">数据来源：{g.provider}</CardDescription>
                    </div>
                    <div className="px-2 py-1 text-xs rounded-full bg-zinc-100 border text-zinc-600 font-medium">{formatGroupMode(g.mode)}</div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-2 text-xs text-zinc-500 mb-3 flex-wrap">
                    <span className="bg-zinc-100 px-2 py-1 rounded">可用节点 {g.candidateCount}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded truncate max-w-[180px]" title={g.current || '还没有选中的节点'}>当前出口 {g.current || '还没有选中的节点'}</span>
                  </div>
                  {g.filter && <p className="text-xs text-zinc-500 font-mono mb-3 bg-zinc-50 p-2 rounded">筛选规则：{g.filter}</p>}

                  {(g.providerMissing || g.providerDisabled) && (
                    <div className="flex items-start gap-2 text-xs text-amber-600 bg-amber-50 p-2 rounded mb-3">
                      <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                      <p>{g.providerMissing ? '这条线路原先使用的订阅已经不存在了，请重新选择数据来源。' : '这条线路依赖的订阅目前被停用了，恢复启用后这里会重新可用。'}</p>
                    </div>
                  )}

                  {!g.providerMissing && !g.providerDisabled && (
                    <div className="flex items-center gap-2 mb-3">
                      <Select 
                        value={g.candidateCount > 0 ? g.current : undefined} 
                        onValueChange={(val: string) => handleSelectGroup(g.name, val)}
                        disabled={g.candidateCount === 0 || g.mode === 'auto'}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder={g.candidateCount === 0 ? "没有节点匹配这条线路" : g.mode === 'auto' ? "系统会自动选择更合适的节点" : "选择一个当前要使用的节点"} />
                        </SelectTrigger>
                        <SelectContent>
                          {g.candidates?.map((c) => <SelectItem key={c.name} value={c.name}>{c.name} ({c.type})</SelectItem>)}
                        </SelectContent>
                      </Select>
                      <Button variant="secondary" size="icon" onClick={() => handleHealthcheck(g.name)} title="重新检测当前线路里的节点" disabled={g.candidateCount === 0 || g.mode === 'auto'}>
                        <RefreshCw className={`h-4 w-4 ${(g.candidateCount === 0 || g.mode === 'auto') ? 'opacity-50' : ''}`} />
                      </Button>
                    </div>
                  )}

                  <div className="flex items-center gap-2 pt-2 border-t">
                    <Button variant="outline" size="sm" onClick={() => openEditGroup(g)}><Pencil className="h-3.5 w-3.5 mr-1" /> 编辑线路</Button>
                    <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => setPendingDeleteGroupName(g.name)}><Trash2 className="h-3.5 w-3.5 mr-1" /> 删除线路</Button>
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
            <DialogTitle>新建一条出口线路</DialogTitle>
            <DialogDescription>为一组节点取个更好理解的名字，后面本地入口就能直接使用这条线路。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>线路名称</Label>
              <Input placeholder="例如：香港日常、日本流媒体、自动选择" value={groupForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setGroupForm({ ...groupForm, name: e.target.value })} />
              <p className="text-xs leading-5 text-zinc-500">建议直接用用途或地区命名，后面绑定入口时会更容易看懂。</p>
            </div>
            <div className="space-y-2">
              <Label>数据来源</Label>
              <Select value={groupForm.provider} onValueChange={(val: string) => setGroupForm({ ...groupForm, provider: val })}>
                <SelectTrigger><SelectValue placeholder="选择订阅源" /></SelectTrigger>
                <SelectContent>{subscriptionNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent>
              </Select>
              <p className="text-xs leading-5 text-zinc-500">这条线路会从你选中的订阅里筛选节点。</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <div className="flex items-center gap-1">
                  <Label>选择方式</Label>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button type="button" className="inline-flex items-center justify-center rounded-full text-zinc-400 hover:text-zinc-600 focus:outline-none">
                        <HelpCircle className="h-3.5 w-3.5" />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent className="w-72 p-3 text-xs text-zinc-600 leading-relaxed shadow-lg">
                      <p className="mb-2"><span className="font-semibold text-zinc-900">手动选择</span><br/>你自己指定当前要走哪个节点。</p>
                      <p><span className="font-semibold text-zinc-900">自动选择</span><br/>系统会自动选择更稳定或延迟更低的节点，适合不想手动切换的场景。</p>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <Select value={groupForm.mode} onValueChange={(val: string) => setGroupForm({ ...groupForm, mode: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="manual">手动指定节点</SelectItem>
                    <SelectItem value="auto">系统自动选择</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <FilterField value={groupForm.filter} onChange={(v) => setGroupForm({ ...groupForm, filter: v })} disabled={groupForm.mode === 'auto'} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddGroup(false)}>取消</Button>
            <Button onClick={handleAddGroup} disabled={groupSubmitting}>{groupSubmitting ? '创建中...' : '创建线路'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ──────── Edit Group Dialog ──────── */}
      <Dialog open={showEditGroup} onOpenChange={setShowEditGroup}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑线路：{editingGroupName}</DialogTitle>
            <DialogDescription>你可以在这里调整数据来源、选择方式和筛选规则。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>数据来源</Label>
              <Select value={groupForm.provider} onValueChange={(val: string) => setGroupForm({ ...groupForm, provider: val })}>
                <SelectTrigger><SelectValue placeholder="选择订阅源" /></SelectTrigger>
                <SelectContent>{subscriptionNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <div className="flex items-center gap-1">
                  <Label>选择方式</Label>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button type="button" className="inline-flex items-center justify-center rounded-full text-zinc-400 hover:text-zinc-600 focus:outline-none">
                        <HelpCircle className="h-3.5 w-3.5" />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent className="w-72 p-3 text-xs text-zinc-600 leading-relaxed shadow-lg">
                      <p className="mb-2"><span className="font-semibold text-zinc-900">手动选择</span><br/>你自己指定当前要走哪个节点。</p>
                      <p><span className="font-semibold text-zinc-900">自动选择</span><br/>系统会自动选择更稳定或延迟更低的节点。</p>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <Select value={groupForm.mode} onValueChange={(val: string) => setGroupForm({ ...groupForm, mode: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="manual">手动指定节点</SelectItem>
                    <SelectItem value="auto">系统自动选择</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <FilterField value={groupForm.filter} onChange={(v) => setGroupForm({ ...groupForm, filter: v })} disabled={groupForm.mode === 'auto'} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditGroup(false)}>取消</Button>
            <Button onClick={handleEditGroup} disabled={groupSubmitting}>{groupSubmitting ? '保存中...' : '保存线路'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(pendingDeleteGroupName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteGroupName(null)
        }}
        title="删除这条出口线路后会发生什么？"
        description={pendingDeleteGroupName ? `“${pendingDeleteGroupName}”删除后，已经绑定它的本地入口不会一起被删，但会暂时失去可用线路，需要你重新选择。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingGroup}
        onConfirm={handleDeleteGroup}
      />
    </div>
  )
}
