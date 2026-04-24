import useSWR from 'swr'
import { CheckCircle2, Copy, Network, Power, Radio, Route, TriangleAlert } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/PageHeader'
import { NoticeCard } from '@/components/NoticeCard'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { toastApiError } from '@/lib/toast-api-error'

function proxyAddress(usage: { localProxy?: { listen?: string; port?: number } }) {
  const localProxy = usage.localProxy
  if (!localProxy?.port) return '还没有代理地址'
  return `${localProxy.listen || '127.0.0.1'}:${localProxy.port}`
}

export default function Usage() {
  const { data: usage, mutate: mutateUsage } = useSWR(apiKeys.usage, api.getUsage, { refreshInterval: 5000 })
  const { data: routes } = useSWR(apiKeys.routes, api.getRoutes)

  const handleCopy = async () => {
    if (!usage?.localProxy?.port) return
    await navigator.clipboard.writeText(proxyAddress(usage))
    toast.success('代理地址已复制')
  }

  const updateUsage = async (body: Record<string, unknown>) => {
    try {
      await api.updateUsage(body)
      await mutateUsage()
      toast.success('使用设置已保存')
    } catch (err) {
      toastApiError(err, '保存使用设置失败')
    }
  }

  const ready = usage?.ready

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Usage"
        title="让这台设备开始使用代理"
        description="查看当前是否可用，复制代理地址，切换线路，或开启虚拟网口。"
      />

      <NoticeCard
        tone={ready ? 'success' : 'warning'}
        icon={ready ? <CheckCircle2 className="h-4 w-4" /> : <TriangleAlert className="h-4 w-4" />}
        title={ready ? '这台设备已经可以使用代理' : '还差几步才能开始使用'}
        description={ready ? '可以复制本机代理地址，或继续切换这台设备使用的线路。' : (usage?.missingSteps || ['请先添加节点来源并创建线路。']).join(' ')}
      />

      {!usage ? (
        <p className="text-sm text-zinc-500">正在读取使用状态...</p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
          <Card className="border-zinc-200 shadow-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base"><Radio className="h-4 w-4" />本机代理</CardTitle>
              <CardDescription>复制下面的地址，填到浏览器、系统或应用里。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="rounded-2xl border border-zinc-200 bg-zinc-50 p-4">
                <div className="text-xs text-zinc-500">代理地址</div>
                <div className="mt-1 font-mono text-lg font-semibold text-zinc-950">{proxyAddress(usage)}</div>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button onClick={handleCopy} disabled={!usage.localProxy?.port}>
                  <Copy className="h-4 w-4" />复制代理地址
                </Button>
                <Button variant="outline" onClick={() => updateUsage({ localProxy: { ...usage.localProxy, enabled: !usage.localProxy.enabled } })}>
                  <Power className="h-4 w-4" />{usage.localProxy.enabled ? '关闭本机代理' : '开启本机代理'}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card className="border-zinc-200 shadow-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base"><Route className="h-4 w-4" />当前线路</CardTitle>
              <CardDescription>选择这台设备访问网络时要使用的线路。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {routes && routes.length > 0 ? (
                <Select value={usage.selectedRoute || ''} onValueChange={(value) => updateUsage({ selectedRoute: value })}>
                  <SelectTrigger><SelectValue placeholder="选择一条线路" /></SelectTrigger>
                  <SelectContent>
                    {routes.map((route) => <SelectItem key={route.name} value={route.name}>{route.name}</SelectItem>)}
                  </SelectContent>
                </Select>
              ) : (
                <EmptyStateCard title="还没有线路" description="先去线路页添加节点来源并创建一条线路。" icon={<Network className="h-5 w-5" />} />
              )}
              <div className="flex items-center justify-between rounded-2xl border border-zinc-200 px-4 py-3">
                <div>
                  <div className="text-sm font-medium text-zinc-950">虚拟网口</div>
                  <div className="text-sm text-zinc-500">开启后，这台设备的网络可以由 AXIS 接管。</div>
                </div>
                <Switch checked={usage.virtualInterface.enabled} onCheckedChange={(enabled) => updateUsage({ virtualInterface: { ...usage.virtualInterface, enabled } })} />
              </div>
              {usage.virtualInterface.message ? <p className="text-sm text-zinc-500">{usage.virtualInterface.message}</p> : null}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
