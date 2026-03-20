import useSWR from 'swr'
import { fetcher } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Settings2, Link2, Network, ShieldCheck } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

export default function Dashboard() {
  const { data: status, error } = useSWR('/api/status', fetcher)

  if (error) return <div className="text-red-500">加载失败</div>
  if (!status) return <div>加载中...</div>

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">系统总览</h1>
        <p className="text-zinc-500">查看当前控制面与运行态的基础信息。</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">控制面模式</CardTitle>
            <Settings2 className="h-4 w-4 text-zinc-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.app.mode}</div>
            <p className="text-xs text-zinc-500 mt-1">
              {status.app.renderOnly ? 'render-only' : 'managed'}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">订阅与节点</CardTitle>
            <Link2 className="h-4 w-4 text-zinc-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.counts.providers} / {status.counts.nodes}</div>
            <p className="text-xs text-zinc-500 mt-1">
              当前 provider 与节点规模
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">出口组与接口</CardTitle>
            <Network className="h-4 w-4 text-zinc-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.counts.groups} / {status.counts.listeners}</div>
            <p className="text-xs text-zinc-500 mt-1">
              逻辑出口与对外入口
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">运行态</CardTitle>
            <ShieldCheck className={`h-4 w-4 ${status.runtime.controllerReachable ? 'text-green-500' : 'text-yellow-500'}`} />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.runtime.controllerReachable ? '可达' : '待检查'}</div>
            <p className="text-xs text-zinc-500 mt-1 truncate">
              渲染: {status.runtime.lastRenderAt ? format(new Date(status.runtime.lastRenderAt), 'PP HH:mm:ss', { locale: zhCN }) : '未渲染'}
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>运行摘要</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="space-y-4 text-sm">
              <div className="flex justify-between border-b pb-2">
                <dt className="text-zinc-500">控制台名称</dt>
                <dd className="font-medium">{status.app.name}</dd>
              </div>
              <div className="flex justify-between border-b pb-2">
                <dt className="text-zinc-500">启动时间</dt>
                <dd className="font-medium">{format(new Date(status.app.startedAt), 'PP HH:mm:ss', { locale: zhCN })}</dd>
              </div>
              <div className="flex justify-between border-b pb-2">
                <dt className="text-zinc-500">运行态模式</dt>
                <dd className="font-medium">{status.runtime.mode}</dd>
              </div>
              <div className="flex justify-between pb-2">
                <dt className="text-zinc-500">最近下发</dt>
                <dd className="font-medium max-w-[200px] truncate" title={status.runtime.lastApplyMessage || "尚未执行"}>
                  {status.runtime.lastApplyMessage || "尚未执行"}
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>重点提醒</CardTitle>
          </CardHeader>
          <CardContent>
            {status.warnings.length === 0 ? (
              <p className="text-sm text-zinc-500">当前没有额外告警。</p>
            ) : (
              <ul className="list-disc pl-4 space-y-2 text-sm text-amber-600">
                {status.warnings.map((warning: string, i: number) => (
                  <li key={i}>{warning}</li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
