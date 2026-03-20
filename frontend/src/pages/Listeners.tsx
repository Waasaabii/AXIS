import { useState } from 'react'
import useSWR from 'swr'
import { fetcher } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { toast } from 'sonner'
import { Network, Users, ShieldAlert, Plus, Trash2, Pencil, AlertTriangle, Eye, EyeOff, Copy, Shuffle } from 'lucide-react'

/* ── helpers ── */
function randomString(length: number): string {
  const chars = 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789'
  let result = ''
  const array = new Uint32Array(length)
  crypto.getRandomValues(array)
  for (let i = 0; i < length; i++) result += chars[array[i] % chars.length]
  return result
}

function generateUsername(): string { return 'user' + randomString(6) }
function generatePassword(): string { return randomString(16) }

/* ── PasswordCell component ── */
function PasswordCell({ value }: { value: string }) {
  const [visible, setVisible] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(value).then(() => toast.success('密码已复制'))
  }

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
  const { data: listeners, mutate: mutateListeners } = useSWR('/api/listeners', fetcher)
  const { data: groups, mutate: mutateGroups } = useSWR('/api/groups', fetcher)

  const groupNames: string[] = groups?.map((g: any) => g.name) || []

  const mutateAll = () => { mutateListeners(); mutateGroups() }

  /* ── Listener state ── */
  const [showAddListener, setShowAddListener] = useState(false)
  const [showEditListener, setShowEditListener] = useState(false)
  const [listenerForm, setListenerForm] = useState<ListenerForm>({ ...emptyListenerForm })
  const [editingListenerName, setEditingListenerName] = useState('')
  const [listenerSubmitting, setListenerSubmitting] = useState(false)

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
    if (!listenerForm.name.trim() || !listenerForm.port || !listenerForm.egress_group) { toast.error('名称、端口和出口组不能为空'); return }
    setListenerSubmitting(true)
    try {
      const payload: any = { name: listenerForm.name.trim(), type: listenerForm.type, listen: listenerForm.listen || '0.0.0.0', port: Number(listenerForm.port), udp: listenerForm.udp, egress_group: listenerForm.egress_group, users: [] }
      if (listenerForm.username.trim() && listenerForm.password.trim()) payload.users = [{ username: listenerForm.username.trim(), password: listenerForm.password.trim() }]
      const res = await fetch('/api/listeners', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
      const data = await res.json()
      if (!data.ok) throw new Error(data.error || '添加失败')
      mutateAll()
      setShowAddListener(false)
      setListenerForm({ ...emptyListenerForm })
      toast.success('端口已添加')
    } catch (err: any) { toast.error(err.message || '添加失败') } finally { setListenerSubmitting(false) }
  }

  const openEditListener = (l: any) => {
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
    if (!listenerForm.port || !listenerForm.egress_group) { toast.error('端口和出口组不能为空'); return }
    setListenerSubmitting(true)
    try {
      const payload: any = { type: listenerForm.type, listen: listenerForm.listen || '0.0.0.0', port: Number(listenerForm.port), udp: listenerForm.udp, egress_group: listenerForm.egress_group }
      if (listenerForm.username.trim() && listenerForm.password.trim()) payload.users = [{ username: listenerForm.username.trim(), password: listenerForm.password.trim() }]
      else payload.users = []
      const res = await fetch(`/api/listeners/${encodeURIComponent(editingListenerName)}/update`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
      const data = await res.json()
      if (!data.ok) throw new Error(data.error || '修改失败')
      mutateAll()
      setShowEditListener(false)
      setListenerForm({ ...emptyListenerForm })
      toast.success('端口已更新')
    } catch (err: any) { toast.error(err.message || '修改失败') } finally { setListenerSubmitting(false) }
  }

  const handleDeleteListener = async (name: string) => {
    if (!confirm(`确认删除端口 "${name}"？`)) return
    try {
      const res = await fetch(`/api/listeners/${encodeURIComponent(name)}`, { method: 'DELETE' })
      const data = await res.json()
      if (!data.ok) throw new Error(data.error || '删除失败')
      mutateAll()
      toast.success(`端口 ${name} 已删除`)
    } catch (err: any) { toast.error(err.message || '删除失败') }
  }

  /* ════════════════════  Render  ════════════════════ */

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">入口管理</h1>
        <p className="text-zinc-500">配置对外监听端口。</p>
      </div>

      {/* ──────── Listeners Section ──────── */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">监听端口</h2>
          <Button size="sm" onClick={openAddListenerDialog} disabled={groupNames.length === 0}>
            <Plus className="h-4 w-4 mr-1" /> 添加端口
          </Button>
        </div>

        {groupNames.length === 0 && (
          <div className="border rounded-lg p-4 bg-amber-50 text-amber-700 text-sm">
            请先创建出口组后，再添加监听端口。
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {!listeners ? (
            <p className="text-sm text-zinc-500">加载中...</p>
          ) : listeners.length === 0 ? (
            <p className="text-sm text-zinc-500 border rounded-lg p-8 text-center bg-zinc-50 md:col-span-2 xl:col-span-3">
              还没有配置任何端口。
            </p>
          ) : (
            listeners.map((l: any) => (
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
                      <span className="inline-flex items-center gap-1 bg-zinc-100 px-2 py-1 rounded text-zinc-600 border"><Users className="h-3 w-3" /> 认证 {l.userCount}</span>
                      <span className="inline-flex items-center bg-zinc-100 px-2 py-1 rounded text-zinc-600 border">UDP {l.udp ? '开' : '关'}</span>
                    </div>
                    <div className="bg-zinc-50 rounded-lg p-3 text-sm border space-y-2">
                      <div className="flex justify-between"><span className="text-zinc-500">出口组</span><span className="font-medium text-zinc-900">{l.egressGroup}</span></div>
                      <div className="flex justify-between"><span className="text-zinc-500">当前代理</span><span className="text-zinc-900 truncate max-w-[140px]" title={l.currentProxy || '未选择'}>{l.currentProxy || '未选择'}</span></div>
                    </div>

                    {/* ── Credentials display ── */}
                    {l.users && l.users.length > 0 && (
                      <div className="bg-zinc-50 rounded-lg p-3 text-sm border space-y-2">
                        {l.users.map((u: any, idx: number) => (
                          <div key={idx} className="space-y-1.5">
                            <div className="flex justify-between items-center">
                              <span className="text-zinc-500">用户名</span>
                              <span className="font-mono text-xs text-zinc-900 select-all">{u.username}</span>
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
                      <p>{l.groupMissing ? '出口组已删除，请编辑重新绑定。' : l.providerMissing ? '上游订阅已删除。' : '上游订阅已停用。'}</p>
                    </div>
                  )}
                  {l.userCount === 0 && !l.groupMissing && (
                    <div className="mt-4 flex items-start gap-2 text-xs text-amber-600 bg-amber-50 p-2 rounded">
                      <ShieldAlert className="h-4 w-4 shrink-0" /><p>未配置认证，端口允许无密码访问。</p>
                    </div>
                  )}

                  <div className="flex items-center gap-2 mt-4 pt-3 border-t">
                    <Button variant="outline" size="sm" onClick={() => openEditListener(l)}><Pencil className="h-3.5 w-3.5 mr-1" /> 编辑</Button>
                    <Button variant="outline" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => handleDeleteListener(l.name)}><Trash2 className="h-3.5 w-3.5 mr-1" /> 删除</Button>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </section>

      {/* ──────── Add Listener Dialog ──────── */}
      <Dialog open={showAddListener} onOpenChange={setShowAddListener}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加监听端口</DialogTitle>
            <DialogDescription>配置代理监听端口，绑定到出口组。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>端口名称</Label>
              <Input placeholder="例如: sg-socks" value={listenerForm.name} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, name: e.target.value })} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>协议</Label><Select value={listenerForm.type} onValueChange={(val: string) => setListenerForm({ ...listenerForm, type: val })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="socks">SOCKS5</SelectItem><SelectItem value="http">HTTP</SelectItem><SelectItem value="mixed">Mixed</SelectItem></SelectContent></Select></div>
              <div className="space-y-2"><Label>端口</Label><Input type="number" placeholder="10801" value={listenerForm.port} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, port: e.target.value })} /></div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>监听地址</Label><Input placeholder="0.0.0.0" value={listenerForm.listen} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, listen: e.target.value })} /></div>
              <div className="space-y-2"><Label>出口组</Label><Select value={listenerForm.egress_group} onValueChange={(val: string) => setListenerForm({ ...listenerForm, egress_group: val })}><SelectTrigger><SelectValue placeholder="选择出口组" /></SelectTrigger><SelectContent>{groupNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent></Select></div>
            </div>
            <div className="flex items-center gap-3"><Switch checked={listenerForm.udp} onCheckedChange={(checked: boolean) => setListenerForm({ ...listenerForm, udp: checked })} /><Label>启用 UDP</Label></div>
            <div className="border-t pt-4">
              <p className="text-sm text-zinc-500 mb-3">认证用户 (留空不设置)</p>
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
            <Button onClick={handleAddListener} disabled={listenerSubmitting}>{listenerSubmitting ? '添加中...' : '添加'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ──────── Edit Listener Dialog ──────── */}
      <Dialog open={showEditListener} onOpenChange={setShowEditListener}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑端口: {editingListenerName}</DialogTitle>
            <DialogDescription>修改端口配置。填写凭据可更新认证。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>协议</Label><Select value={listenerForm.type} onValueChange={(val: string) => setListenerForm({ ...listenerForm, type: val })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="socks">SOCKS5</SelectItem><SelectItem value="http">HTTP</SelectItem><SelectItem value="mixed">Mixed</SelectItem></SelectContent></Select></div>
              <div className="space-y-2"><Label>端口</Label><Input type="number" value={listenerForm.port} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, port: e.target.value })} /></div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2"><Label>监听地址</Label><Input placeholder="0.0.0.0" value={listenerForm.listen} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setListenerForm({ ...listenerForm, listen: e.target.value })} /></div>
              <div className="space-y-2"><Label>出口组</Label><Select value={listenerForm.egress_group} onValueChange={(val: string) => setListenerForm({ ...listenerForm, egress_group: val })}><SelectTrigger><SelectValue placeholder="选择出口组" /></SelectTrigger><SelectContent>{groupNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}</SelectContent></Select></div>
            </div>
            <div className="flex items-center gap-3"><Switch checked={listenerForm.udp} onCheckedChange={(checked: boolean) => setListenerForm({ ...listenerForm, udp: checked })} /><Label>启用 UDP</Label></div>
            <div className="border-t pt-4">
              <p className="text-sm text-zinc-500 mb-3">认证用户 (清空两个字段即移除认证)</p>
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
            <Button onClick={handleEditListener} disabled={listenerSubmitting}>{listenerSubmitting ? '保存中...' : '保存'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
