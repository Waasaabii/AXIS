import { useState } from 'react'
import useSWR from 'swr'
import { Copy, Plus, Radio, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/PageHeader'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { toastApiError } from '@/lib/toast-api-error'

export default function Publications() {
  const { data: routes } = useSWR(apiKeys.routes, api.getRoutes)
  const { data: publications, mutate } = useSWR(apiKeys.publications, api.getPublications)
  const [name, setName] = useState('')
  const [route, setRoute] = useState('')
  const [port, setPort] = useState('10801')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const addPublication = async () => {
    try {
      await api.addPublication({ name: name.trim(), type: 'http_proxy', enabled: true, route, listen: '0.0.0.0', port: Number(port), auth: { username: username.trim(), password } })
      setName('')
      setRoute('')
      setUsername('')
      setPassword('')
      await mutate()
      toast.success('发布已创建')
    } catch (err) {
      toastApiError(err, '创建发布失败')
    }
  }

  const copyPublication = async (item: { listen?: string; port?: number; auth?: { username?: string; password?: string } }) => {
    const user = item.auth?.username && item.auth.password ? `${encodeURIComponent(item.auth.username)}:${encodeURIComponent(item.auth.password)}@` : ''
    await navigator.clipboard.writeText(`http://${user}${item.listen || '127.0.0.1'}:${item.port}`)
    toast.success('连接信息已复制')
  }

  return (
    <div className="space-y-6">
      <PageHeader eyebrow="Publish" title="把线路发布给其他地方使用" description="把已有线路生成 HTTP 代理、订阅链接或 AXIS 连接。对外使用默认需要账号和密码。" />
      <Card className="border-zinc-200 shadow-sm">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base"><Radio className="h-4 w-4" />创建 HTTP 发布</CardTitle>
          <CardDescription>选择一条线路，生成给浏览器、应用或其他设备使用的连接。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <div className="space-y-2"><Label>名称</Label><Input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：给手机用" /></div>
          <div className="space-y-2">
            <Label>线路</Label>
            <Select value={route} onValueChange={setRoute}><SelectTrigger><SelectValue placeholder="选择线路" /></SelectTrigger><SelectContent>{(routes || []).map((item) => <SelectItem key={item.name} value={item.name}>{item.name}</SelectItem>)}</SelectContent></Select>
          </div>
          <div className="space-y-2"><Label>端口</Label><Input value={port} onChange={(event) => setPort(event.target.value)} /></div>
          <div className="space-y-2"><Label>账号</Label><Input value={username} onChange={(event) => setUsername(event.target.value)} /></div>
          <div className="space-y-2"><Label>密码</Label><Input value={password} onChange={(event) => setPassword(event.target.value)} type="password" /></div>
          <div className="md:col-span-2 xl:col-span-5"><Button onClick={addPublication} disabled={!name.trim() || !route || !username.trim() || !password}><Plus className="h-4 w-4" />发布线路</Button></div>
        </CardContent>
      </Card>

      <Card className="border-zinc-200 shadow-sm">
        <CardHeader><CardTitle className="text-base">已发布的连接</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {publications && publications.length > 0 ? publications.map((item) => (
            <div key={item.name} className="flex items-center justify-between rounded-2xl border border-zinc-200 px-4 py-3">
              <div><div className="font-medium text-zinc-950">{item.name}</div><div className="text-sm text-zinc-500">线路：{item.route} · 端口：{item.port || '按需生成'} · {item.enabled ? '可用' : '已关闭'}</div></div>
              <div className="flex gap-1"><Button variant="ghost" size="icon" onClick={() => copyPublication(item)}><Copy className="h-4 w-4" /></Button><Button variant="ghost" size="icon" onClick={async () => { await api.deletePublication(item.name); await mutate(); toast.success('发布已删除') }}><Trash2 className="h-4 w-4" /></Button></div>
            </div>
          )) : <EmptyStateCard title="还没有发布连接" description="创建后，其他设备或应用就可以通过账号和密码连接。" />}
        </CardContent>
      </Card>
    </div>
  )
}
