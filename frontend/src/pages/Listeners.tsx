import { useState } from 'react'
import useSWR from 'swr'
import { api, ApiError, type ListenerView, type ListenerUser } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { useClipboard } from '@/hooks/useClipboard'
import { generatePassword, generateUsername } from '@/lib/random'
import { toast } from 'sonner'
import { Network, Radio, Users, ShieldAlert, Plus, Trash2, Pencil, AlertTriangle, Eye, EyeOff, Copy, Shuffle } from 'lucide-react'

/* ── PasswordCell component ── */
function PasswordCell({ value }: { value: string }) {
  const [visible, setVisible] = useState(false)
  const { copy } = useClipboard()

  const handleCopy = () => void copy(value, '密码已复制')

  return (
    <span className="inline-flex items-center gap-1">
      <span
        className="font-mono text-xs select-all cursor-pointer"
        onClick={() => setVisible(!visible)}
        title={visible ? '点击隐藏' : '点击显示'}
      >{visible ? value : '••••••••'}</span>
      <button type="button" onClick={() => setVisible(!visible)} className="p-0.5 rounded hover:bg-zinc-200 text-zinc-400 hover:text-zinc-600 transition-colors" title={visible ? '隐藏' : '显示'}>
        {visible ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
      </button>
      <button type="button" onClick={handleCopy} className="p-0.5 rounded hover:bg-zinc-200 text-zinc-400 hover:text-zinc-600 transition-colors" title="复制密码">
        <Copy className="h-3.5 w-3.5" />
      </button>
    </span>
  )
}

interface ListenerForm { name: string; type: string; listen: string; port: string; udp: boolean; egress_group: string; username: string; password: string }
const emptyListenerForm: ListenerForm = { name: '', type: 'socks', listen: '0.0.0.0', port: '', udp: true, egress_group: '', username: '', password: '' }

export default function Listeners() {
  const { data: listeners, mutate: mutateListeners } = useSWR('/api/listeners', api.getListeners)
  const { data: groups, mutate: mutateGroups } = useSWR('/api/groups', api.getGroups)
  const { copy } = useClipboard()

  const groupNames = groups?.map((group) => group.name) || []

  const mutateAll = () => { mutateListeners(); mutateGroups() }

  /* ── Listener state ── */
  const [showAddListener, setShowAddListener] = useState(false)
  const [showEditListener, setShowEditListener] = useState(false)
  const [listenerForm, setListenerForm] = useState<ListenerForm>({ ...emptyListenerForm })
  const [editingListenerName, setEditingListenerName] = useState('')
  const [listenerSubmitting, setListenerSubmitting] = useState(false)
  const [pendingDeleteListenerName, setPendingDeleteListenerName] = useState<string | null>(null)
  const [deletingListener, setDeletingListener] = useState(false)

  /* ════════════════════  Listener handlers  ════════════════════ */

  const openAddListenerDialog = () => {
    setListenerForm({
      ...emptyListenerForm,
      username: generateUsername(),
      password: generatePassword(),
    })
    setShowAddListener(true)
  }

  const handleAddListener = async () => {
    if (!listenerForm.name.trim() || !listenerForm.port || !listenerForm.egress_group) { toast.error('入口名称、端口和出口线路都需要填写。'); return }
    setListenerSubmitting(true)
    try {
      const payload = { name: listenerForm.name.trim(), type: listenerForm.type, listen: listenerForm.listen || '0.0.0.0', port: Number(listenerForm.port), udp: listenerForm.udp, egress_group: listenerForm.egress_group, users: [] as ListenerUser[] }
      if (listenerForm.username.trim() && listenerForm.password.trim()) payload.users = [{ username: listenerForm.username.trim(), password: listenerForm.password.trim() }]
      const data = await api.addListener(payload)
      if (!data.ok) throw new Error('添加失败')
      mutateAll()
      setShowAddListener(false)
      setListenerForm({ ...emptyListenerForm })
      toast.success('本地入口已创建。')
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '添加失败') } finally { setListenerSubmitting(false) }
  }

  const openEditListener = (l: ListenerView) => {
    const firstUser = l.users?.[0]
    setEditingListenerName(l.name)
    setListenerForm({
      name: l.name,
      type: l.type || 'socks',
      listen: l.listen || '0.0.0.0',
      port: String(l.port),
      udp: l.udp !== false,
      egress_group: l.egressGroup || '',
      username: firstUser?.username || '',
      password: firstUser?.password || '',
    })
    setShowEditListener(true)
  }

  const handleEditListener = async () => {
    if (!listenerForm.port || !listenerForm.egress_group) { toast.error('端口和出口线路都需要填写。'); return }
    setListenerSubmitting(true)
    try {
      const payload = { type: listenerForm.type, listen: listenerForm.listen || '0.0.0.0', port: Number(listenerForm.port), udp: listenerForm.udp, egress_group: listenerForm.egress_group, users: [] as ListenerUser[] }
      if (listenerForm.username.trim() && listenerForm.password.trim()) payload.users = [{ username: listenerForm.username.trim(), password: listenerForm.password.trim() }]
      const data = await api.updateListener(editingListenerName, payload)
      if (!data.ok) throw new Error('修改失败')
      mutateAll()
      setShowEditListener(false)
      setListenerForm({ ...emptyListenerForm })
      toast.success('本地入口已更新。')
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '修改失败') } finally { setListenerSubmitting(false) }
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
    } catch (err) { toast.error(err instanceof ApiError ? err.message : '删除失败') } finally { setDeletingListener(false) }
  }

  /* ════════════════════  Render  ════════════════════ */

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Local Access"
        title="给浏览器和设备准备一个可连接的本地入口"
        description="本地入口就是你最终要在系统、浏览器或其他设备里填写的代理地址。先选好出口线路，再决定这个入口是否需要用户名和密码。"
        actions={
          <Button onClick={openAddListenerDialog} disabled={groupNames.length === 0}>
            <Plus className="h-4 w-4" />
            添加本地入口
          </Button>
        }
      />

      <NoticeCard
        icon={<Radio className="h-4 w-4" />}
        title="什么时候需要多个本地入口"
        description="如果你想给不同设备、不同场景或不同权限单独分配账号和线路，就值得创建多个本地入口。比如“电脑默认”“电视流媒体”“临时分享”这种分法会更清晰。"
      />

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">本地入口列表</h2>
        </div>

        {groupNames.length === 0 && (
          <NoticeCard
            tone="warning"
            icon={<AlertTriangle className="h-4 w-4" />}
            title="还不能创建本地入口"
            description="因为系统里还没有可用的出口线路。先去“出口线路”页整理好一条线路，这里才能继续往下配。"
          />
        )}

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {!listeners ? (
            <p className="text-sm text-zinc-500">加载中...</p>
          ) : listeners.length === 0 ? (
            <EmptyStateCard
              className="md:col-span-2 xl:col-span-3"
              icon={<Radio className="h-5 w-5" />}
              title="还没有任何本地入口"
              description="先创建一个入口，让浏览器、系统代理或其他设备真正能连到 AXIS。创建时只需要决定协议、端口和要绑定的出口线路。"
              action={
                <Button onClick={openAddListenerDialog} disabled={groupNames.length === 0}>
                  <Plus className="h-4 w-4" />
                  添加第一个本地入口
                </Button>
              }
            />
          ) : (
            listeners.map((l: ListenerView) => (
              <Card key={l.name} className={`flex flex-col ${l.groupMissing || l.providerMissing ? 'border-red-200 bg-red-50/30' : l.providerDisabled ? 'border-amber-200 bg-amber-50/30' : ''}`}>
                <CardHeader className="pb-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        <Network className="h-4 w-4 text-zinc-400" />{l.name}
                      </CardTitle>
                      <CardDescription className="mt-1 font-mono">{l.listen}:{l.port}</CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      {l.status !== 'configured' && (
                        <div className={`px-2 py-1 text-xs rounded font-medium border ${l.status === 'orphaned' ? 'bg-red-50 text-red-700 border-red-200' : 'bg-amber-50 text-amber-700 border-amber-200'}`}>
                          {l.status === 'orphaned' ? '孤立' : '降级'}
                        </div>
                      )}
                      <div className="px-2 py-1 text-xs rounded bg-blue-50 text-blue-700 font-medium border border-blue-100">{l.type}</div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="flex-1 flex flex-col justify-between">
                  <div className="space-y-4">
                    <div className="flex flex-wrap gap-2 text-xs">
                      <span className="inline-flex items-center gap-1 bg-zinc-100 px-2 py-1 rounded text-zinc-600 border"><Users className="h-3 w-3" /> 账号 {l.userCount}</span>
                      <span className="inline-flex items-center bg-zinc-100 px-2 py-1 rounded text-zinc-600 border">UDP {l.udp ? '开' : '关'}</span>
                    </div>
                    <div className="bg-zinc-50 rounded-lg p-3 text-sm border space-y-2">
                      <div className="flex justify-between"><span className="text-zinc-500">使用线路</span><span className="font-medium text-zinc-900">{l.egressGroup}</span></div>
                      <div className="flex justify-between"><span className="text-zinc-500">当前出口</span><span className="text-zinc-900 truncate max-w-[140px]" title={l.currentProxy || '还没有选中的节点'}>{l.currentProxy || '还没有选中的节点'}</span></div>
                    </div>

                    {l.users && l.users.length > 0 && (
                      <div className="bg-zinc-50 rounded-lg p-3 text-sm border space-y-2">
                        {l.users.map((u: ListenerUser, idx: number) => (
                          <div key={idx} className="space-y-1.5">
                            <div className="flex justify-between items-center">
                              <span className="text-zinc-500">用户名</span>
                              <div className="flex items-center gap-1">
                                <span className="font-mono text-xs text-zinc-900 select-all">{u.username}</span>
                                <button type="button" onClick={() => void copy(u.username, '用户名已复制')} className="p-0.5 rounded hover:bg-zinc-200 text-zinc-400 hover:text-zinc-600 transition-colors" title="复制用户名">
                                  <Copy className="h-3.5 w-3.5" />
                                </button>
                              </div>
                            </div>
                            <div className="flex justify-between items-center">
                              <span className="text-zinc-500">密码</span>
                              <PasswordCell value={u.password} />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>

                  {(l.groupMissing || l.providerMissing || l.providerDisabled) && (
                    <div className="mt-4 flex items-start gap-2 text-xs text-amber-600 bg-amber-50 p-2 rounded">
                      <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                      <p>{l.groupMissing ? '这条入口原先绑定的出口线路已经不存在了，请重新选择线路。' : l.providerMissing ? '这条入口依赖的订阅已经不存在了。' : '这条入口依赖的订阅目前被停用了。'}</p>
                    </div>
                  )}
                  {l.userCount === 0 && !l.groupMissing && (
                    <div className="mt-4 flex items-start gap-2 text-xs text-amber-600 bg-amber-50 p-2 rounded">
                      <ShieldAlert className="h-4 w-4 shrink-0" /><p>这个入口没有账号密码，知道地址的人都可以直接连接。</p>
                    </div>
                  )}

                  <div className="flex items-center gap-2 mt-4 pt-3 border-t">
                    <Button variant="outline" size="sm" onClick={() => openEditListener(l)}><Pencil className="h-3.5 w-3.5 mr-1" /> 编辑入口</Button>
                    <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => setPendingDeleteListenerName(l.name)}><Trash2 className="h-3.5 w-3.5 mr-1" /> 删除入口</Button>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </section>

      <Dialog open={showAddListener} onOpenChange={setShowAddListener}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加一个本地入口</DialogTitle>
            <DialogDescription>配置浏览器、系统或其他设备要连接的本地代理地址，并绑定到一条出口线路。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>入口名称</Label>
              <Input placeholder="例如：电脑默认、电视流媒体、分享给朋友" value={listenerForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, name: e.target.value })} />
              <p className="text-xs leading-5 text-zinc-500">建议按设备或用途命名，后面更容易区分。</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>协议</Label><Select value={listenerForm.type} onValueChange={(val: string) => setListenerForm({ ...listenerForm, type: val })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="socks">SOCKS5</SelectItem><SelectItem value="http">HTTP</SelectItem><SelectItem value="mixed">Mixed</SelectItem></SelectContent></Select><p className="text-xs leading-5 text-zinc-500">如果没有特殊需求，SOCKS5 或 Mixed 都可以。</p></div>
              <div className="space-y-2"><Label>端口</Label><Input type="number" placeholder="10801" value={listenerForm.port} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, port: e.target.value })} /><p className="text-xs leading-5 text-zinc-500">这是设备真正要填写的端口号。</p></div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>监听地址</Label><Input placeholder="0.0.0.0" value={listenerForm.listen} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, listen: e.target.value })} /><p className="text-xs leading-5 text-zinc-500">默认留 `0.0.0.0` 即可，表示其他设备也能连接。</p></div>
              <div className="space-y-2"><Label>出口线路</Label><Select value={listenerForm.egress_group} onValueChange={(val: string) => setListenerForm({ ...listenerForm, egress_group: val })}><SelectTrigger><SelectValue placeholder="选择一条出口线路" /></SelectTrigger><SelectContent>{groupNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent></Select><p className="text-xs leading-5 text-zinc-500">这个入口连上后，就会走你选中的线路。</p></div>
            </div>
            <div className="flex items-center gap-3"><Switch checked={listenerForm.udp} onCheckedChange={(checked: boolean) => setListenerForm({ ...listenerForm, udp: checked })} /><Label>允许 UDP</Label></div>
            <div className="border-t pt-4">
              <p className="text-sm text-zinc-500 mb-3">访问账号（如果留空，这个入口会变成无密码访问）</p>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>用户名</Label>
                  <div className="flex gap-1">
                    <Input placeholder="username" value={listenerForm.username} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, username: e.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, username: generateUsername() })} title="随机用户名"><Shuffle className="h-4 w-4" /></Button>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>密码</Label>
                  <div className="flex gap-1">
                    <Input placeholder="password" value={listenerForm.password} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, password: e.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, password: generatePassword() })} title="随机密码"><Shuffle className="h-4 w-4" /></Button>
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
            <DialogDescription>你可以修改协议、端口、线路或访问账号。清空账号字段即可移除认证。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>协议</Label><Select value={listenerForm.type} onValueChange={(val: string) => setListenerForm({ ...listenerForm, type: val })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="socks">SOCKS5</SelectItem><SelectItem value="http">HTTP</SelectItem><SelectItem value="mixed">Mixed</SelectItem></SelectContent></Select></div>
              <div className="space-y-2"><Label>端口</Label><Input type="number" value={listenerForm.port} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, port: e.target.value })} /></div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>监听地址</Label><Input placeholder="0.0.0.0" value={listenerForm.listen} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, listen: e.target.value })} /></div>
              <div className="space-y-2"><Label>出口线路</Label><Select value={listenerForm.egress_group} onValueChange={(val: string) => setListenerForm({ ...listenerForm, egress_group: val })}><SelectTrigger><SelectValue placeholder="选择一条出口线路" /></SelectTrigger><SelectContent>{groupNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent></Select></div>
            </div>
            <div className="flex items-center gap-3"><Switch checked={listenerForm.udp} onCheckedChange={(checked: boolean) => setListenerForm({ ...listenerForm, udp: checked })} /><Label>允许 UDP</Label></div>
            <div className="border-t pt-4">
              <p className="text-sm text-zinc-500 mb-3">访问账号（清空两个字段即可移除认证）</p>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>用户名</Label>
                  <div className="flex gap-1">
                    <Input placeholder="username" value={listenerForm.username} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, username: e.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, username: generateUsername() })} title="随机用户名"><Shuffle className="h-4 w-4" /></Button>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>密码</Label>
                  <div className="flex gap-1">
                    <Input placeholder="password" value={listenerForm.password} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, password: e.target.value })} />
                    <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={() => setListenerForm({ ...listenerForm, password: generatePassword() })} title="随机密码"><Shuffle className="h-4 w-4" /></Button>
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
        description={pendingDeleteListenerName ? `“${pendingDeleteListenerName}”删除后，设备就不能再通过这个地址连接 AXIS。已经绑定的线路不会被一起删除。` : ''}
        confirmLabel="确认删除"
        destructive
        confirming={deletingListener}
        onConfirm={handleDeleteListener}
      />
    </div>
  )
}
