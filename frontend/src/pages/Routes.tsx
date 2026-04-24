import { useState } from 'react'
import useSWR from 'swr'
import { Copy, Plus, Route, Server, ShieldCheck, Trash2 } from 'lucide-react'
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
import type { BaotaConfigResponse, LocalNodeCheckResponse, LocalNodeConnectionResponse, NodeSource } from '@/services/product-types'

const protocolPorts: Record<string, number> = { http: 39011, socks: 39012, trojan: 39013, hysteria2: 39014 }
const protocolLabels: Record<string, string> = { http: 'HTTP', socks: 'SOCKS', trojan: 'Trojan', hysteria2: 'Hysteria2' }

function sourceTypeLabel(source: NodeSource) {
  if (source.type === 'subscription') return '订阅'
  if (source.type === 'proxy') return '单个代理'
  if (source.type === 'axis') return '其他 AXIS'
  if (source.type === 'local_node') return `本机节点 · ${protocolLabels[source.protocol || ''] || source.protocol || '代理'}`
  return source.type
}

async function copyText(text: string, message: string) {
  await navigator.clipboard.writeText(text)
  toast.success(message)
}

export default function Routes() {
  const { data: sources, mutate: mutateSources } = useSWR(apiKeys.nodeSources, api.getNodeSources)
  const { data: routes, mutate: mutateRoutes } = useSWR(apiKeys.routes, api.getRoutes)
  const [sourceName, setSourceName] = useState('')
  const [sourceUrl, setSourceUrl] = useState('')
  const [routeName, setRouteName] = useState('')
  const [routeSource, setRouteSource] = useState('')
  const [localName, setLocalName] = useState('')
  const [localProtocol, setLocalProtocol] = useState('http')
  const [localHost, setLocalHost] = useState('')
  const [localPort, setLocalPort] = useState(protocolPorts.http)
  const [localRoute, setLocalRoute] = useState('')
  const [localUsername, setLocalUsername] = useState('axis')
  const [localPassword, setLocalPassword] = useState('')
  const [checkResult, setCheckResult] = useState<LocalNodeCheckResponse | null>(null)
  const [connectionResult, setConnectionResult] = useState<LocalNodeConnectionResponse | null>(null)
  const [baotaResult, setBaotaResult] = useState<BaotaConfigResponse | null>(null)

  const addSource = async () => {
    try {
      await api.addNodeSource({ name: sourceName.trim(), type: 'subscription', enabled: true, protocol: 'mihomo-http', subscription: { url: sourceUrl.trim() } })
      setSourceName('')
      setSourceUrl('')
      await mutateSources()
      toast.success('节点来源已添加')
    } catch (err) {
      toastApiError(err, '添加节点来源失败')
    }
  }

  const addRoute = async () => {
    try {
      await api.addRoute({ name: routeName.trim(), enabled: true, strategy: 'manual', entry: { source: routeSource } })
      setRouteName('')
      setRouteSource('')
      await mutateRoutes()
      toast.success('线路已创建')
    } catch (err) {
      toastApiError(err, '创建线路失败')
    }
  }

  const addLocalNode = async () => {
    try {
      await api.addNodeSource({
        name: localName.trim(),
        type: 'local_node',
        enabled: true,
        protocol: localProtocol,
        local_node: {
          access_mode: 'bt_reverse_proxy',
          listen: '127.0.0.1',
          port: localPort,
          external_host: localHost.trim(),
          external_port: localProtocol === 'trojan' ? 443 : localPort,
          route: localRoute,
          users: [{ username: localUsername.trim(), password: localPassword.trim() }],
        },
      })
      setLocalName('')
      setLocalPassword('')
      setCheckResult(null)
      setConnectionResult(null)
      setBaotaResult(null)
      await mutateSources()
      toast.success('本机节点已添加')
    } catch (err) {
      toastApiError(err, '添加本机节点失败')
    }
  }

  const inspectLocalNode = async (name: string) => {
    try {
      const [check, connection, baota] = await Promise.all([api.checkLocalNode(name), api.getLocalNodeConnection(name), api.getLocalNodeBaotaConfig(name)])
      setCheckResult(check)
      setConnectionResult(connection)
      setBaotaResult(baota)
      toast.success('检查完成')
    } catch (err) {
      toastApiError(err, '检查失败')
    }
  }

  const localNodes = (sources || []).filter((source) => source.type === 'local_node')
  const canAddLocalNode = localName.trim() && localHost.trim() && localRoute && localUsername.trim() && localPassword.trim() && localPort > 0

  return (
    <div className="space-y-6">
      <PageHeader eyebrow="Routes" title="整理节点来源和线路" description="把订阅、单个代理、其他 AXIS 或本机节点统一加入，再组合成可使用和发布的线路。" />
      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base"><Server className="h-4 w-4" />添加订阅</CardTitle>
            <CardDescription>把已有订阅加入 AXIS，之后可以组合成线路。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2"><Label>名称</Label><Input value={sourceName} onChange={(event) => setSourceName(event.target.value)} placeholder="例如：我的订阅" /></div>
            <div className="space-y-2"><Label>订阅链接</Label><Input value={sourceUrl} onChange={(event) => setSourceUrl(event.target.value)} placeholder="https://..." /></div>
            <Button onClick={addSource} disabled={!sourceName.trim() || !sourceUrl.trim()}><Plus className="h-4 w-4" />添加订阅</Button>
          </CardContent>
        </Card>

        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base"><Route className="h-4 w-4" />创建线路</CardTitle>
            <CardDescription>选择一个节点来源，创建这台设备可以使用或对外发布的线路。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2"><Label>线路名称</Label><Input value={routeName} onChange={(event) => setRouteName(event.target.value)} placeholder="例如：日常使用" /></div>
            <div className="space-y-2">
              <Label>节点来源</Label>
              <Select value={routeSource} onValueChange={setRouteSource}>
                <SelectTrigger><SelectValue placeholder="选择节点来源" /></SelectTrigger>
                <SelectContent>{(sources || []).map((source) => <SelectItem key={source.name} value={source.name}>{source.name}</SelectItem>)}</SelectContent>
              </Select>
            </div>
            <Button onClick={addRoute} disabled={!routeName.trim() || !routeSource}><Plus className="h-4 w-4" />创建线路</Button>
          </CardContent>
        </Card>
      </div>

      <Card className="border-zinc-200 shadow-sm">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base"><ShieldCheck className="h-4 w-4" />创建本机节点</CardTitle>
          <CardDescription>把这台设备变成一个可连接的节点，外部连接只使用域名。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 lg:grid-cols-3">
          <div className="space-y-2"><Label>名称</Label><Input value={localName} onChange={(event) => setLocalName(event.target.value)} placeholder="例如：家里服务器" /></div>
          <div className="space-y-2">
            <Label>连接方式</Label>
            <Select value={localProtocol} onValueChange={(value) => { setLocalProtocol(value); setLocalPort(protocolPorts[value] || 39011) }}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{Object.entries(protocolLabels).map(([value, label]) => <SelectItem key={value} value={value}>{label}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div className="space-y-2"><Label>访问域名</Label><Input value={localHost} onChange={(event) => setLocalHost(event.target.value)} placeholder="axis.example.com" /></div>
          <div className="space-y-2"><Label>本机服务端口</Label><Input type="number" value={localPort} onChange={(event) => setLocalPort(Number(event.target.value))} /></div>
          <div className="space-y-2">
            <Label>使用哪条线路</Label>
            <Select value={localRoute} onValueChange={setLocalRoute}>
              <SelectTrigger><SelectValue placeholder="选择线路" /></SelectTrigger>
              <SelectContent>{(routes || []).map((route) => <SelectItem key={route.name} value={route.name}>{route.name}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div className="space-y-2"><Label>账号</Label><Input value={localUsername} onChange={(event) => setLocalUsername(event.target.value)} /></div>
          <div className="space-y-2"><Label>密码</Label><Input value={localPassword} onChange={(event) => setLocalPassword(event.target.value)} placeholder="给连接设备使用" /></div>
          <div className="flex items-end lg:col-span-2"><Button onClick={addLocalNode} disabled={!canAddLocalNode}><Plus className="h-4 w-4" />添加本机节点</Button></div>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader><CardTitle className="text-base">节点来源</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {sources && sources.length > 0 ? sources.map((source) => (
              <div key={source.name} className="flex items-center justify-between gap-3 rounded-2xl border border-zinc-200 px-4 py-3">
                <div><div className="font-medium text-zinc-950">{source.name}</div><div className="text-sm text-zinc-500">{sourceTypeLabel(source)} · {source.enabled ? '可用' : '已关闭'}</div></div>
                <div className="flex gap-1">
                  {source.type === 'local_node' && <Button variant="outline" size="sm" onClick={() => inspectLocalNode(source.name)}>检查</Button>}
                  <Button variant="ghost" size="icon" onClick={async () => { await api.deleteNodeSource(source.name); await mutateSources(); toast.success('节点来源已删除') }}><Trash2 className="h-4 w-4" /></Button>
                </div>
              </div>
            )) : <EmptyStateCard title="还没有节点来源" description="添加订阅后，就可以创建线路。" />}
          </CardContent>
        </Card>
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader><CardTitle className="text-base">线路</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {routes && routes.length > 0 ? routes.map((route) => (
              <div key={route.name} className="flex items-center justify-between rounded-2xl border border-zinc-200 px-4 py-3">
                <div><div className="font-medium text-zinc-950">{route.name}</div><div className="text-sm text-zinc-500">来源：{route.entry?.source || '未选择'} · {route.enabled ? '可用' : '已关闭'}</div></div>
                <Button variant="ghost" size="icon" onClick={async () => { await api.deleteRoute(route.name); await mutateRoutes(); toast.success('线路已删除') }}><Trash2 className="h-4 w-4" /></Button>
              </div>
            )) : <EmptyStateCard title="还没有线路" description="选择节点来源后，创建第一条线路。" />}
          </CardContent>
        </Card>
      </div>

      {localNodes.length > 0 && (checkResult || connectionResult || baotaResult) && (
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader><CardTitle className="text-base">本机节点连接信息</CardTitle><CardDescription>复制给其他设备时只使用域名，不使用服务器 IP。</CardDescription></CardHeader>
          <CardContent className="space-y-4">
            {checkResult && <div className="grid gap-2 md:grid-cols-2">{checkResult.checks.map((item) => <div key={item.name} className="rounded-2xl border border-zinc-200 p-3"><div className="font-medium text-zinc-950">{item.name}：{item.ok ? '正常' : '需要处理'}</div><div className="text-sm text-zinc-500">{item.message}{item.action ? `，${item.action}` : ''}</div></div>)}</div>}
            {connectionResult && <div className="space-y-2">{connectionResult.connections.map((item) => <div key={`${item.username}-${item.url}`} className="flex items-center justify-between gap-3 rounded-2xl bg-zinc-50 px-4 py-3 text-sm"><span className="break-all text-zinc-700">{item.url}</span><Button variant="outline" size="sm" onClick={() => copyText(item.url, '连接信息已复制')}><Copy className="h-4 w-4" />复制</Button></div>)}</div>}
            {baotaResult && <div className="space-y-2"><Label>宝塔需要转发到：{baotaResult.target}</Label><pre className="overflow-auto rounded-2xl bg-zinc-950 p-4 text-xs text-zinc-50">{baotaResult.snippet}</pre><Button variant="outline" onClick={() => copyText(baotaResult.snippet, '宝塔配置已复制')}><Copy className="h-4 w-4" />复制宝塔配置</Button></div>}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
