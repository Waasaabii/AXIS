import { useState, type Dispatch, type SetStateAction } from 'react'
import useSWR from 'swr'
import { api, type GroupView, type LandingProxyView, type ProviderListItem } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
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
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Trash2, Pencil, AlertTriangle, RefreshCw, FolderPlus, HelpCircle, ChevronDown, Network, Route, Plus, ShieldCheck, ArrowUp, ArrowDown, X } from 'lucide-react'
import { mutateMany } from '@/lib/swr'
import { toastApiError } from '@/lib/toast-api-error'

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
  if (mode === 'fallback') return '顺序容灾'
  return mode === 'auto' ? '自动选择' : '手动选择'
}

const NO_LANDING_VALUE = '__direct__'

function landingSelectValue(value?: string) {
  return value && value.trim() ? value : NO_LANDING_VALUE
}

function buildRouteSummary(group: GroupView) {
  return group.routeSummary || `${group.current || group.name} -> 公网`
}

function buildGroupWarning(group: GroupView) {
  if (group.providerMissing) return '订阅已删除，请重新选择节点来源。'
  if (group.providerDisabled) return '订阅已停用，启用后这里会恢复。'
  if (group.landingMissing) return '最终出口节点已删除，请重新选择。'
  if (group.landingDisabled) return '最终出口节点已停用，这条线路暂时不会继续转发。'
  return ''
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
interface GroupForm { name: string; provider: string; mode: string; filter: string; landing_proxy: string }
const emptyGroupForm: GroupForm = { name: '', provider: '', mode: 'manual', filter: '', landing_proxy: '' }

interface FallbackGroupForm {
  name: string
  provider: string
  proxies: string[]
  search: string
  landing_proxy: string
  interval: string
}

const emptyFallbackGroupForm: FallbackGroupForm = { name: '', provider: '', proxies: [], search: '', landing_proxy: '', interval: '60' }

interface LandingForm {
  name: string
  type: string
  server: string
  port: string
  username: string
  password: string
  tls: boolean
  sni: string
  skip_cert_verify: boolean
  enabled: boolean
}

const emptyLandingForm: LandingForm = {
  name: '',
  type: 'socks5',
  server: '',
  port: '',
  username: '',
  password: '',
  tls: false,
  sni: '',
  skip_cert_verify: false,
  enabled: true,
}

export default function Interfaces() {
  const { data: groups, mutate: mutateGroups } = useSWR(apiKeys.groups, api.getGroups)
  const { data: landingProxies, mutate: mutateLandingProxies } = useSWR(apiKeys.landingProxies, api.getLandingProxies)
  const { data: configData, mutate: mutateConfig } = useSWR(apiKeys.config, api.getConfig)
  const { data: providers } = useSWR(apiKeys.providers, api.getProviders)

  const subscriptionNames = configData?.config.subscriptions.map((subscription) => subscription.name) || []
  const providerList: ProviderListItem[] = providers || []

  const mutateAll = () => void mutateMany(mutateGroups, mutateLandingProxies, mutateConfig)

  /* ── Landing proxy state ── */
  const [showAddLanding, setShowAddLanding] = useState(false)
  const [showEditLanding, setShowEditLanding] = useState(false)
  const [landingForm, setLandingForm] = useState<LandingForm>({ ...emptyLandingForm })
  const [editingLandingName, setEditingLandingName] = useState('')
  const [landingSubmitting, setLandingSubmitting] = useState(false)
  const [pendingDeleteLandingName, setPendingDeleteLandingName] = useState<string | null>(null)
  const [deletingLanding, setDeletingLanding] = useState(false)

  /* ── Group state ── */
  const [showAddGroup, setShowAddGroup] = useState(false)
  const [showEditGroup, setShowEditGroup] = useState(false)
  const [groupForm, setGroupForm] = useState<GroupForm>({ ...emptyGroupForm })
  const [editingGroupName, setEditingGroupName] = useState('')
  const [groupSubmitting, setGroupSubmitting] = useState(false)
  const [showAddFallbackGroup, setShowAddFallbackGroup] = useState(false)
  const [showEditFallbackGroup, setShowEditFallbackGroup] = useState(false)
  const [fallbackForm, setFallbackForm] = useState<FallbackGroupForm>({ ...emptyFallbackGroupForm })
  const [editingFallbackGroupName, setEditingFallbackGroupName] = useState('')
  const [fallbackSubmitting, setFallbackSubmitting] = useState(false)
  const [pendingDeleteGroupName, setPendingDeleteGroupName] = useState<string | null>(null)
  const [deletingGroup, setDeletingGroup] = useState(false)

  const selectedFallbackProvider = providerList.find((provider) => provider.name === fallbackForm.provider)
  const availableFallbackNodes = selectedFallbackProvider?.nodes || []
  const fallbackSearch = fallbackForm.search.trim().toLowerCase()
  const filteredFallbackNodes = availableFallbackNodes.filter((node) => {
    if (!fallbackSearch) return true
    return [node.name, node.server, node.type].filter(Boolean).some((value) => String(value).toLowerCase().includes(fallbackSearch))
  })

  /* ════════════════════  Group handlers  ════════════════════ */

  const openAddLanding = () => {
    setLandingForm({ ...emptyLandingForm })
    setShowAddLanding(true)
  }

  const openEditLanding = (landing: LandingProxyView) => {
    setEditingLandingName(landing.name)
    setLandingForm({
      name: landing.name,
      type: landing.type || 'socks5',
      server: landing.server || '',
      port: String(landing.port || ''),
      username: landing.username || '',
      password: landing.password || '',
      tls: Boolean(landing.tls),
      sni: landing.sni || '',
      skip_cert_verify: Boolean(landing.skipCertVerify),
      enabled: landing.enabled !== false,
    })
    setShowEditLanding(true)
  }

  const handleAddLanding = async () => {
    if (!landingForm.name.trim() || !landingForm.server.trim() || !landingForm.port) {
      toast.error('最终出口节点名称、地址和端口都需要填写。')
      return
    }
    setLandingSubmitting(true)
    try {
      const data = await api.addLandingProxy({
        name: landingForm.name.trim(),
        type: landingForm.type,
        server: landingForm.server.trim(),
        port: Number(landingForm.port),
        username: landingForm.username.trim() || undefined,
        password: landingForm.password.trim() || undefined,
        tls: landingForm.tls,
        sni: landingForm.sni.trim() || undefined,
        skip_cert_verify: landingForm.skip_cert_verify,
        enabled: landingForm.enabled,
      })
      if (!data.ok) throw new Error('添加失败')
      mutateAll()
      setShowAddLanding(false)
      setLandingForm({ ...emptyLandingForm })
      toast.success('最终出口节点已创建。')
    } catch (err) {
      toastApiError(err, '添加失败')
    } finally {
      setLandingSubmitting(false)
    }
  }

  const handleEditLanding = async () => {
    if (!landingForm.server.trim() || !landingForm.port) {
      toast.error('地址和端口都需要填写。')
      return
    }
    setLandingSubmitting(true)
    try {
      const data = await api.updateLandingProxy(editingLandingName, {
        type: landingForm.type,
        server: landingForm.server.trim(),
        port: Number(landingForm.port),
        username: landingForm.username.trim(),
        password: landingForm.password.trim(),
        tls: landingForm.tls,
        sni: landingForm.sni.trim(),
        skip_cert_verify: landingForm.skip_cert_verify,
        enabled: landingForm.enabled,
      })
      if (!data.ok) throw new Error('修改失败')
      mutateAll()
      setShowEditLanding(false)
      setLandingForm({ ...emptyLandingForm })
      toast.success('最终出口节点已更新。')
    } catch (err) {
      toastApiError(err, '修改失败')
    } finally {
      setLandingSubmitting(false)
    }
  }

  const handleDeleteLanding = async () => {
    if (!pendingDeleteLandingName) return
    setDeletingLanding(true)
    try {
      const data = await api.deleteLandingProxy(pendingDeleteLandingName)
      if (!data.ok) throw new Error('删除失败')
      mutateAll()
      toast.success(`已删除最终出口节点“${pendingDeleteLandingName}”。`)
      setPendingDeleteLandingName(null)
    } catch (err) {
      toastApiError(err, '删除失败')
    } finally {
      setDeletingLanding(false)
    }
  }

  const handleAddGroup = async () => {
    if (!groupForm.name.trim() || !groupForm.provider) { toast.error('名称和订阅源不能为空'); return }
    setGroupSubmitting(true)
    try {
      const data = await api.addEgressGroup({
        name: groupForm.name.trim(),
        provider: groupForm.provider,
        mode: groupForm.mode,
        filter: groupForm.filter,
        landing_proxy: groupForm.landing_proxy || undefined,
      })
      if (!data.ok) throw new Error('添加失败')
      mutateAll()
      setShowAddGroup(false)
      setGroupForm({ ...emptyGroupForm })
      toast.success('出口线路已创建。')
    } catch (err) { toastApiError(err, '添加失败') } finally { setGroupSubmitting(false) }
  }

  const resetFallbackForm = () => {
    setFallbackForm({ ...emptyFallbackGroupForm })
    setEditingFallbackGroupName('')
  }

  const handleFallbackProviderChange = (providerName: string) => {
    const provider = providerList.find((item) => item.name === providerName)
    const availableNames = new Set((provider?.nodes || []).map((node) => node.name))
    setFallbackForm((current) => ({
      ...current,
      provider: providerName,
      search: '',
      proxies: current.proxies.filter((name) => availableNames.has(name)),
    }))
  }

  const toggleFallbackProxy = (proxyName: string) => {
    setFallbackForm((current) => {
      const exists = current.proxies.includes(proxyName)
      return {
        ...current,
        proxies: exists ? current.proxies.filter((item) => item !== proxyName) : [...current.proxies, proxyName],
      }
    })
  }

  const moveFallbackProxy = (index: number, direction: -1 | 1) => {
    setFallbackForm((current) => {
      const targetIndex = index + direction
      if (targetIndex < 0 || targetIndex >= current.proxies.length) return current
      const next = [...current.proxies]
      const [item] = next.splice(index, 1)
      next.splice(targetIndex, 0, item)
      return { ...current, proxies: next }
    })
  }

  const handleAddFallbackGroup = async () => {
    if (!fallbackForm.name.trim() || !fallbackForm.provider) { toast.error('名称和订阅源不能为空'); return }
    if (fallbackForm.proxies.length === 0) { toast.error('顺序容灾组至少需要选择一个节点'); return }
    const interval = Number(fallbackForm.interval || 60)
    if (!Number.isFinite(interval) || interval <= 0) { toast.error('检测频率必须是大于 0 的数字'); return }
    setFallbackSubmitting(true)
    try {
      const data = await api.addEgressGroup({
        name: fallbackForm.name.trim(),
        provider: fallbackForm.provider,
        mode: 'fallback',
        proxies: fallbackForm.proxies,
        landing_proxy: fallbackForm.landing_proxy || undefined,
        interval,
      })
      if (!data.ok) throw new Error('创建失败')
      mutateAll()
      setShowAddFallbackGroup(false)
      resetFallbackForm()
      toast.success('顺序容灾组已创建。')
    } catch (err) { toastApiError(err, '创建失败') } finally { setFallbackSubmitting(false) }
  }

  const openEditGroup = (g: GroupView) => {
    if (g.mode === 'fallback') {
      const configGroup = configData?.config.egress_groups.find((item) => item.name === g.name)
      setEditingFallbackGroupName(g.name)
      setFallbackForm({
        name: g.name,
        provider: g.provider,
        proxies: configGroup?.proxies || g.proxyOrder || [],
        search: '',
        landing_proxy: g.landingProxy || '',
        interval: String(g.healthCheckInterval || 60),
      })
      setShowEditFallbackGroup(true)
      return
    }
    setEditingGroupName(g.name)
    setGroupForm({ name: g.name, provider: g.provider, mode: g.mode, filter: g.filter || '', landing_proxy: g.landingProxy || '' })
    setShowEditGroup(true)
  }

  const handleEditGroup = async () => {
    if (!groupForm.provider) { toast.error('订阅源不能为空'); return }
    setGroupSubmitting(true)
    try {
      const data = await api.updateEgressGroup(editingGroupName, {
        provider: groupForm.provider,
        mode: groupForm.mode,
        filter: groupForm.filter,
        landing_proxy: groupForm.landing_proxy,
      })
      if (!data.ok) throw new Error('修改失败')
      mutateAll()
      setShowEditGroup(false)
      toast.success('出口线路已更新。')
    } catch (err) { toastApiError(err, '修改失败') } finally { setGroupSubmitting(false) }
  }

  const handleEditFallbackGroup = async () => {
    if (!fallbackForm.provider) { toast.error('订阅源不能为空'); return }
    if (fallbackForm.proxies.length === 0) { toast.error('顺序容灾组至少需要选择一个节点'); return }
    const interval = Number(fallbackForm.interval || 60)
    if (!Number.isFinite(interval) || interval <= 0) { toast.error('检测频率必须是大于 0 的数字'); return }
    setFallbackSubmitting(true)
    try {
      const data = await api.updateEgressGroup(editingFallbackGroupName, {
        provider: fallbackForm.provider,
        mode: 'fallback',
        proxies: fallbackForm.proxies,
        landing_proxy: fallbackForm.landing_proxy || '',
        interval,
      })
      if (!data.ok) throw new Error('修改失败')
      mutateAll()
      setShowEditFallbackGroup(false)
      resetFallbackForm()
      toast.success('顺序容灾组已更新。')
    } catch (err) { toastApiError(err, '修改失败') } finally { setFallbackSubmitting(false) }
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
    } catch (err) { toastApiError(err, '删除失败') } finally { setDeletingGroup(false) }
  }

  const handleSelectGroup = async (groupName: string, proxyName: string) => {
    try {
      await api.selectGroup(groupName, { proxyName })
      mutateGroups()
      toast.success(`线路“${groupName}”当前改为使用“${proxyName}”。`)
    } catch (err) { toastApiError(err, '切换失败') }
  }

  const handleHealthcheck = async (groupName: string) => {
    try {
      await api.healthcheckGroup(groupName)
      mutateGroups()
      toast.success(`线路“${groupName}”已经重新检测可用节点。`)
    } catch (err) { toastApiError(err, '健康检查失败') }
  }

  /* ════════════════════  Render  ════════════════════ */

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Routes"
        title="把节点和最终出口整理成一条可复用路径"
        description="先确定最终出口，再整理出口线路。本地代理只需要选择线路即可。"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={openAddLanding}>
              <Plus className="h-4 w-4" />
              添加最终出口节点
            </Button>
            <Button onClick={() => { setGroupForm({ ...emptyGroupForm }); setShowAddGroup(true) }} disabled={subscriptionNames.length === 0}>
              <FolderPlus className="h-4 w-4" />
              添加出口线路
            </Button>
            <Button variant="outline" onClick={() => { resetFallbackForm(); setShowAddFallbackGroup(true) }} disabled={subscriptionNames.length === 0}>
              <ShieldCheck className="h-4 w-4" />
              添加顺序容灾组
            </Button>
          </div>
        }
      />

      <NoticeCard
        icon={<Route className="h-4 w-4" />}
        title="先确定最终出口"
        description="想固定出站 IP，就先添加最终出口节点；不添加则直接出公网。"
      />

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">最终出口节点</h2>
            <p className="text-sm text-zinc-500">这里是最终出口。出口线路选了它后，会先走订阅节点，再从这里出公网。</p>
          </div>
          <Button variant="outline" onClick={openAddLanding}>
            <Plus className="h-4 w-4" />
            添加最终出口节点
          </Button>
        </div>

        {!landingProxies ? (
          <p className="text-sm text-zinc-500">加载中...</p>
        ) : landingProxies.length === 0 ? (
          <EmptyStateCard
            icon={<Route className="h-5 w-5" />}
            title="还没有最终出口节点"
            description="想固定出站 IP，就先添加一个最终出口节点；不添加则直接出公网。"
            action={
              <Button variant="outline" onClick={openAddLanding}>
                <Plus className="h-4 w-4" />
                添加第一个最终出口节点
              </Button>
            }
          />
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            {landingProxies.map((landing) => (
              <Card key={landing.name} className={landing.enabled ? 'shadow-sm' : 'border-amber-200 bg-amber-50/30'}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        {landing.name}
                        {!landing.enabled && <span className="text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded font-normal">已停用</span>}
                      </CardTitle>
                      <CardDescription className="mt-1">{landing.server}:{landing.port}</CardDescription>
                    </div>
                    <div className="px-2 py-1 text-xs rounded-full bg-zinc-100 border text-zinc-600 font-medium uppercase">
                      {landing.type}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex flex-wrap gap-2 text-xs text-zinc-500">
                    <span className="bg-zinc-100 px-2 py-1 rounded">已绑定线路 {landing.routeCount}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded">{landing.username ? '需要账号密码' : '无需账号密码'}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded">{landing.tls ? 'TLS 已开启' : '纯明文连接'}</span>
                  </div>

                  {landing.inUseBy.length > 0 ? (
                    <div className="rounded-lg border bg-zinc-50 p-3 text-sm space-y-2">
                      <p className="text-zinc-500">正在使用这个最终出口的出口线路</p>
                      <div className="flex flex-wrap gap-2">
                        {landing.inUseBy.map((groupName) => (
                          <span key={groupName} className="rounded-full border bg-white px-2 py-1 text-xs text-zinc-700">
                            {groupName}
                          </span>
                        ))}
                      </div>
                    </div>
                  ) : (
                    <div className="rounded-lg border bg-zinc-50 p-3 text-sm text-zinc-500">
                      这个最终出口节点还没有被任何出口线路使用。
                    </div>
                  )}

                  {!landing.enabled && (
                    <div className="flex items-start gap-2 text-xs text-amber-700 bg-amber-50 p-2 rounded">
                      <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                      <p>停用后，相关出口线路不会继续转发到这里，本地代理会进入降级状态。</p>
                    </div>
                  )}

                  <div className="flex items-center gap-2 pt-2 border-t">
                    <Button variant="outline" size="sm" onClick={() => openEditLanding(landing)}><Pencil className="h-3.5 w-3.5 mr-1" /> 编辑最终出口节点</Button>
                    <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => setPendingDeleteLandingName(landing.name)}><Trash2 className="h-3.5 w-3.5 mr-1" /> 删除最终出口节点</Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </section>

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
            description="因为还没导入订阅。先去“订阅与节点”填入订阅链接。"
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
              description="先新建一条线路，把节点按地区或用途整理好。本地代理只需要选线路，不用翻节点列表。"
              action={
                <Button onClick={() => { setGroupForm({ ...emptyGroupForm }); setShowAddGroup(true) }} disabled={subscriptionNames.length === 0}>
                  <FolderPlus className="h-4 w-4" />
                  添加第一条线路
                </Button>
              }
            />
          ) : (
            groups.map((g: GroupView) => (
              <Card key={g.name} className={g.providerMissing || g.providerDisabled || g.landingMissing || g.landingDisabled || g.candidateCount === 0 ? 'border-red-100 bg-gradient-to-b from-white to-red-50/30 shadow-[0_8px_30px_-6px_rgba(239,68,68,0.15)] relative z-10 transition-shadow' : 'shadow-sm'}>
                <CardHeader className="pb-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        {g.name}
                        {g.providerMissing && <span className="text-xs bg-red-100 text-red-700 px-1.5 py-0.5 rounded font-normal">订阅已删除</span>}
                        {g.providerDisabled && <span className="text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded font-normal">订阅已停用</span>}
                        {g.landingMissing && <span className="text-xs bg-red-100 text-red-700 px-1.5 py-0.5 rounded font-normal">出口已删除</span>}
                        {g.landingDisabled && <span className="text-xs bg-amber-100 text-amber-700 px-1.5 py-0.5 rounded font-normal">出口已停用</span>}
                      </CardTitle>
                      <CardDescription className="mt-1">数据来源：{g.provider}</CardDescription>
                    </div>
                    <div className="px-2 py-1 text-xs rounded-full bg-zinc-100 border text-zinc-600 font-medium">{formatGroupMode(g.mode)}</div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-2 text-xs text-zinc-500 mb-3 flex-wrap">
                    <span className="bg-zinc-100 px-2 py-1 rounded">{g.mode === 'fallback' ? `顺序节点 ${g.candidateCount}` : `可用节点 ${g.candidateCount}`}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded truncate max-w-[180px]" title={g.current || '还没有选中的节点'}>当前节点 {g.current || '还没有选中的节点'}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded truncate max-w-[260px]" title={buildRouteSummary(g)}>实际路径 {buildRouteSummary(g)}</span>
                    <span className="bg-zinc-100 px-2 py-1 rounded">{g.landingProxy ? `最终出口 ${g.landingProxy}` : '最终出口 直接出公网'}</span>
                  </div>
                  {g.mode === 'fallback' ? (
                    <div className="mb-3 rounded-lg border bg-zinc-50 p-3">
                      <p className="text-xs text-zinc-500">顺序容灾顺位</p>
                      <div className="mt-2 flex flex-wrap gap-2">
                        {(g.proxyOrder || []).length > 0 ? (g.proxyOrder || []).map((name, index) => (
                          <span key={name} className="rounded-full border bg-white px-2 py-1 text-xs text-zinc-700">
                            #{index + 1} {name}
                          </span>
                        )) : <span className="text-xs text-zinc-400">还没有配置顺位</span>}
                      </div>
                    </div>
                  ) : g.filter ? <p className="text-xs text-zinc-500 font-mono mb-3 bg-zinc-50 p-2 rounded">筛选规则：{g.filter}</p> : null}

                  {(g.providerMissing || g.providerDisabled || g.landingMissing || g.landingDisabled) && (
                    <div className="flex items-start gap-2 text-xs text-amber-600 bg-amber-50 p-2 rounded mb-3">
                      <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                      <p>{buildGroupWarning(g)}</p>
                    </div>
                  )}

                  {!g.providerMissing && !g.providerDisabled && !g.landingMissing && !g.landingDisabled && (
                    <div className="flex items-center gap-2 mb-3">
                      <Select 
                        value={g.candidateCount > 0 ? g.currentValue : undefined} 
                        onValueChange={(val: string) => handleSelectGroup(g.name, val)}
                        disabled={g.candidateCount === 0 || g.mode === 'auto' || g.mode === 'fallback'}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder={g.candidateCount === 0 ? "没有节点匹配这条线路" : g.mode === 'auto' ? "系统会自动选择更合适的节点" : g.mode === 'fallback' ? "顺序容灾组按顺位自动切换" : "选择一个当前要使用的节点"} />
                        </SelectTrigger>
                        <SelectContent>
                          {g.candidates?.map((c) => <SelectItem key={c.id} value={c.id}>{c.name} ({c.type})</SelectItem>)}
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
            <DialogDescription>为一组节点取个更好理解的名字，后面本地代理就能直接选这条线路。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>线路名称</Label>
              <Input placeholder="例如：香港日常、日本流媒体、自动选择" value={groupForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setGroupForm({ ...groupForm, name: e.target.value })} />
              <p className="text-xs leading-5 text-zinc-500">建议按用途或地区命名，后面配置本地代理时更容易选对。</p>
            </div>
            <div className="space-y-2">
              <Label>数据来源</Label>
              <Select value={groupForm.provider} onValueChange={(val: string) => setGroupForm({ ...groupForm, provider: val })}>
                <SelectTrigger><SelectValue placeholder="选择订阅源" /></SelectTrigger>
                <SelectContent>{subscriptionNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent>
              </Select>
              <p className="text-xs leading-5 text-zinc-500">这条线路会从你选中的订阅里筛选节点。</p>
            </div>
            <div className="space-y-2">
              <Label>最终出口</Label>
              <Select value={landingSelectValue(groupForm.landing_proxy)} onValueChange={(val: string) => setGroupForm({ ...groupForm, landing_proxy: val === NO_LANDING_VALUE ? '' : val })}>
                <SelectTrigger><SelectValue placeholder="直接出公网，或选择一个最终出口" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value={NO_LANDING_VALUE}>直接出公网</SelectItem>
                  {(landingProxies || []).map((landing) => (
                    <SelectItem key={landing.name} value={landing.name}>
                      {landing.name}{landing.enabled ? '' : '（已停用）'}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs leading-5 text-zinc-500">选了最终出口后，流量会先走订阅节点，再从最终出口出公网。</p>
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
            <div className="space-y-2">
              <Label>最终出口</Label>
              <Select value={landingSelectValue(groupForm.landing_proxy)} onValueChange={(val: string) => setGroupForm({ ...groupForm, landing_proxy: val === NO_LANDING_VALUE ? '' : val })}>
                <SelectTrigger><SelectValue placeholder="直接出公网，或选择一个最终出口" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value={NO_LANDING_VALUE}>直接出公网</SelectItem>
                  {(landingProxies || []).map((landing) => (
                    <SelectItem key={landing.name} value={landing.name}>
                      {landing.name}{landing.enabled ? '' : '（已停用）'}
                    </SelectItem>
                  ))}
                </SelectContent>
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

      <Dialog open={showAddFallbackGroup} onOpenChange={setShowAddFallbackGroup}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>新建顺序容灾组</DialogTitle>
            <DialogDescription>按顺位挑出一组固定节点。排在前面的节点不可用时，流量会自动切到后面的候补节点。</DialogDescription>
          </DialogHeader>
          <FallbackGroupEditor
            fallbackForm={fallbackForm}
            setFallbackForm={setFallbackForm}
            subscriptionNames={subscriptionNames}
            landingProxies={landingProxies || []}
            filteredFallbackNodes={filteredFallbackNodes}
            onProviderChange={handleFallbackProviderChange}
            onToggleProxy={toggleFallbackProxy}
            onMoveProxy={moveFallbackProxy}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddFallbackGroup(false)}>取消</Button>
            <Button onClick={handleAddFallbackGroup} disabled={fallbackSubmitting}>{fallbackSubmitting ? '创建中...' : '创建顺序容灾组'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showEditFallbackGroup} onOpenChange={setShowEditFallbackGroup}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>编辑顺序容灾组：{editingFallbackGroupName}</DialogTitle>
            <DialogDescription>你可以调整订阅源、顺位列表、健康检查频率和最终出口。</DialogDescription>
          </DialogHeader>
          <FallbackGroupEditor
            fallbackForm={fallbackForm}
            setFallbackForm={setFallbackForm}
            subscriptionNames={subscriptionNames}
            landingProxies={landingProxies || []}
            filteredFallbackNodes={filteredFallbackNodes}
            onProviderChange={handleFallbackProviderChange}
            onToggleProxy={toggleFallbackProxy}
            onMoveProxy={moveFallbackProxy}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditFallbackGroup(false)}>取消</Button>
            <Button onClick={handleEditFallbackGroup} disabled={fallbackSubmitting}>{fallbackSubmitting ? '保存中...' : '保存顺序容灾组'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showAddLanding} onOpenChange={setShowAddLanding}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加一个最终出口节点</DialogTitle>
            <DialogDescription>配置一个固定的最终出口。出口线路选了它后，会先走订阅节点，再从这里出公网。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>节点名称</Label>
                <Input placeholder="例如：日本固定出口、新加坡固定出口" value={landingForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, name: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>协议</Label>
                <Select value={landingForm.type} onValueChange={(val: string) => setLandingForm({ ...landingForm, type: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="socks5">SOCKS5</SelectItem>
                    <SelectItem value="http">HTTP</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>地址</Label>
                <Input placeholder="landing.example.com" value={landingForm.server} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, server: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>端口</Label>
                <Input type="number" placeholder="443" value={landingForm.port} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, port: e.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>用户名</Label>
                <Input placeholder="可选" value={landingForm.username} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, username: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>密码</Label>
                <Input placeholder="可选" value={landingForm.password} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, password: e.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>SNI</Label>
                <Input placeholder="只有 TLS 场景需要填写" value={landingForm.sni} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, sni: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label className="opacity-0">状态</Label>
                <div className="rounded-lg border bg-zinc-50 px-3 py-2 text-sm text-zinc-600">
                  这个最终出口节点创建后，就可以在出口线路里直接选用了。
                </div>
              </div>
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                <Switch checked={landingForm.enabled} onCheckedChange={(checked: boolean) => setLandingForm({ ...landingForm, enabled: checked })} />
                <Label>启用这个最终出口节点</Label>
              </div>
              <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                <Switch checked={landingForm.tls} onCheckedChange={(checked: boolean) => setLandingForm({ ...landingForm, tls: checked })} />
                <Label>通过 TLS 连接</Label>
              </div>
              <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                <Switch checked={landingForm.skip_cert_verify} onCheckedChange={(checked: boolean) => setLandingForm({ ...landingForm, skip_cert_verify: checked })} />
                <Label>跳过证书校验</Label>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddLanding(false)}>取消</Button>
            <Button onClick={handleAddLanding} disabled={landingSubmitting}>{landingSubmitting ? '创建中...' : '创建最终出口节点'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showEditLanding} onOpenChange={setShowEditLanding}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑最终出口节点：{editingLandingName}</DialogTitle>
            <DialogDescription>修改会影响所有使用这个最终出口的出口线路。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>协议</Label>
                <Select value={landingForm.type} onValueChange={(val: string) => setLandingForm({ ...landingForm, type: val })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="socks5">SOCKS5</SelectItem>
                    <SelectItem value="http">HTTP</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>端口</Label>
                <Input type="number" value={landingForm.port} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, port: e.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>地址</Label>
                <Input value={landingForm.server} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, server: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>SNI</Label>
                <Input value={landingForm.sni} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, sni: e.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>用户名</Label>
                <Input value={landingForm.username} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, username: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>密码</Label>
                <Input value={landingForm.password} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLandingForm({ ...landingForm, password: e.target.value })} />
              </div>
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                <Switch checked={landingForm.enabled} onCheckedChange={(checked: boolean) => setLandingForm({ ...landingForm, enabled: checked })} />
                <Label>启用</Label>
              </div>
              <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                <Switch checked={landingForm.tls} onCheckedChange={(checked: boolean) => setLandingForm({ ...landingForm, tls: checked })} />
                <Label>TLS</Label>
              </div>
              <div className="flex items-center gap-3 rounded-lg border px-3 py-2">
                <Switch checked={landingForm.skip_cert_verify} onCheckedChange={(checked: boolean) => setLandingForm({ ...landingForm, skip_cert_verify: checked })} />
                <Label>跳过证书校验</Label>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditLanding(false)}>取消</Button>
            <Button onClick={handleEditLanding} disabled={landingSubmitting}>{landingSubmitting ? '保存中...' : '保存最终出口节点'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(pendingDeleteGroupName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteGroupName(null)
        }}
        title="删除这条出口线路后会发生什么？"
        description={pendingDeleteGroupName ? `“${pendingDeleteGroupName}”删除后，相关本地代理不会被一起删除，但会暂时失去可用线路，需要你重新选择。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingGroup}
        onConfirm={handleDeleteGroup}
      />

      <ConfirmDialog
        open={Boolean(pendingDeleteLandingName)}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteLandingName(null)
        }}
        title="删除这个最终出口节点后会发生什么？"
        description={pendingDeleteLandingName ? `“${pendingDeleteLandingName}”删除后，已经绑定它的出口线路会失去最终出口，需要你重新指定。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingLanding}
        onConfirm={handleDeleteLanding}
      />
    </div>
  )
}

function FallbackGroupEditor({
  fallbackForm,
  setFallbackForm,
  subscriptionNames,
  landingProxies,
  filteredFallbackNodes,
  onProviderChange,
  onToggleProxy,
  onMoveProxy,
}: {
  fallbackForm: FallbackGroupForm
  setFallbackForm: Dispatch<SetStateAction<FallbackGroupForm>>
  subscriptionNames: string[]
  landingProxies: LandingProxyView[]
  filteredFallbackNodes: ProviderListItem['nodes']
  onProviderChange: (providerName: string) => void
  onToggleProxy: (proxyName: string) => void
  onMoveProxy: (index: number, direction: -1 | 1) => void
}) {
  return (
    <div className="space-y-4 py-2">
      <div className="space-y-2">
        <Label>线路名称</Label>
        <Input placeholder="例如：香港顺序容灾、日本主备切换" value={fallbackForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFallbackForm({ ...fallbackForm, name: e.target.value })} />
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>数据来源</Label>
          <Select value={fallbackForm.provider} onValueChange={onProviderChange}>
            <SelectTrigger><SelectValue placeholder="选择订阅源" /></SelectTrigger>
            <SelectContent>{subscriptionNames.map((name) => <SelectItem key={name} value={name}>{name}</SelectItem>)}</SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>检测频率（秒）</Label>
          <Input type="number" value={fallbackForm.interval} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFallbackForm({ ...fallbackForm, interval: e.target.value })} />
        </div>
      </div>
      <div className="space-y-2">
        <Label>最终出口</Label>
        <Select value={landingSelectValue(fallbackForm.landing_proxy)} onValueChange={(val: string) => setFallbackForm({ ...fallbackForm, landing_proxy: val === NO_LANDING_VALUE ? '' : val })}>
          <SelectTrigger><SelectValue placeholder="直接出公网，或选择一个最终出口" /></SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_LANDING_VALUE}>直接出公网</SelectItem>
            {landingProxies.map((landing) => <SelectItem key={landing.name} value={landing.name}>{landing.name}{landing.enabled ? '' : '（已停用）'}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label>搜索节点</Label>
        <Input placeholder="按名称、地址或协议筛选可加入顺位的节点" value={fallbackForm.search} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFallbackForm({ ...fallbackForm, search: e.target.value })} />
      </div>
      <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
        <div className="space-y-2">
          <Label>可加入顺位的节点</Label>
          <div className="max-h-72 space-y-2 overflow-y-auto rounded-lg border bg-zinc-50 p-3">
            {filteredFallbackNodes.length === 0 ? (
              <p className="text-sm text-zinc-500">当前订阅还没有可选节点。先刷新订阅，或换一个已经拿到节点缓存的数据源。</p>
            ) : filteredFallbackNodes.map((node) => {
              const selected = fallbackForm.proxies.includes(node.name)
              return (
                <button
                  key={node.id}
                  type="button"
                  onClick={() => onToggleProxy(node.name)}
                  className={`flex w-full items-start justify-between rounded-lg border px-3 py-2 text-left transition-colors ${selected ? 'border-emerald-200 bg-emerald-50' : 'border-zinc-200 bg-white hover:bg-zinc-50'}`}
                >
                  <div>
                    <div className="text-sm font-medium text-zinc-900">{node.name}</div>
                    <div className="text-xs text-zinc-500">{node.server}:{node.port} · {node.type}</div>
                  </div>
                  <div className={`rounded-full px-2 py-1 text-xs ${selected ? 'bg-emerald-100 text-emerald-700' : 'bg-zinc-100 text-zinc-500'}`}>
                    {selected ? '已加入' : '加入顺位'}
                  </div>
                </button>
              )
            })}
          </div>
        </div>
        <div className="space-y-2">
          <Label>当前顺位</Label>
          <div className="max-h-72 space-y-2 overflow-y-auto rounded-lg border bg-zinc-50 p-3">
            {fallbackForm.proxies.length === 0 ? (
              <p className="text-sm text-zinc-500">还没有选择任何节点。左侧点击节点后，这里会按顺位展示主用和候补节点。</p>
            ) : fallbackForm.proxies.map((name, index) => (
              <div key={`${name}-${index}`} className="rounded-lg border bg-white p-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-medium text-zinc-900">#{index + 1} {name}</div>
                    <div className="text-xs text-zinc-500">{index === 0 ? '主用节点' : '候补节点'}</div>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button variant="outline" size="icon" onClick={() => onMoveProxy(index, -1)} disabled={index === 0}>
                      <ArrowUp className="h-3.5 w-3.5" />
                    </Button>
                    <Button variant="outline" size="icon" onClick={() => onMoveProxy(index, 1)} disabled={index === fallbackForm.proxies.length - 1}>
                      <ArrowDown className="h-3.5 w-3.5" />
                    </Button>
                    <Button variant="outline" size="icon" onClick={() => onToggleProxy(name)}>
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
