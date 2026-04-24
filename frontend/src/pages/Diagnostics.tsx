import useSWR from 'swr'
import { Activity, CheckCircle2, FileText, TriangleAlert } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { NoticeCard } from '@/components/NoticeCard'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'

export default function Runtime() {
  const { data: status } = useSWR(apiKeys.status, api.getStatus, { refreshInterval: 5000 })
  const { data: events } = useSWR(apiKeys.events, api.getEvents, { refreshInterval: 5000 })
  const warnings = (status?.warnings || []) as string[]
  const counts = (status?.counts || {}) as Record<string, number | undefined>

  return (
    <div className="space-y-6">
      <PageHeader eyebrow="Diagnostics" title="检查连接问题和最近记录" description="这里集中展示代理核心、虚拟网口、发布项和最近失败原因，原始记录只作为排查依据。" />
      <NoticeCard
        tone={warnings.length > 0 ? 'warning' : 'success'}
        icon={warnings.length > 0 ? <TriangleAlert className="h-4 w-4" /> : <CheckCircle2 className="h-4 w-4" />}
        title={warnings.length > 0 ? '还有需要处理的提醒' : '当前没有明显阻塞'}
        description={warnings.length > 0 ? warnings.join(' ') : '如果连接异常，可以先查看最近记录。'}
      />
      <div className="grid gap-4 md:grid-cols-3">
        <Metric title="节点来源" value={String(counts.nodeSources ?? counts.providers ?? 0)} />
        <Metric title="线路" value={String(counts.routes ?? counts.groups ?? 0)} />
        <Metric title="发布" value={String(counts.publications ?? counts.listeners ?? 0)} />
      </div>
      <Card className="border-zinc-200 shadow-sm">
        <CardHeader><CardTitle className="flex items-center gap-2 text-base"><FileText className="h-4 w-4" />最近记录</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {(events || []).slice(0, 20).map((event) => <div key={event.id} className="rounded-2xl border border-zinc-200 px-4 py-3 text-sm"><div className="font-medium text-zinc-950">{event.message}</div><div className="text-zinc-500">{event.at}</div></div>)}
          {(!events || events.length === 0) ? <p className="text-sm text-zinc-500">暂时还没有记录。</p> : null}
        </CardContent>
      </Card>
    </div>
  )
}

function Metric({ title, value }: { title: string; value: string }) {
  return <Card className="border-zinc-200 shadow-sm"><CardHeader><CardTitle className="flex items-center gap-2 text-sm text-zinc-600"><Activity className="h-4 w-4" />{title}</CardTitle></CardHeader><CardContent><div className="text-3xl font-semibold text-zinc-950">{value}</div></CardContent></Card>
}
