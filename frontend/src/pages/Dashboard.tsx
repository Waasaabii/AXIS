import useSWR from 'swr'
import { api, ApiError } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Settings2, Link2, Network, ShieldCheck } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

function formatMode(mode?: string) {
  return mode === 'render-only' ? '仅保存配置' : '自动接管'
}

export default function Dashboard() {
  const { data: status, error } = useSWR('/api/status', api.getStatus)

  if (error) return <div className="text-red-500">{error instanceof ApiError ? error.message : '加载失败'}</div>
  if (!status) return <div>加载中...</div>

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">系统总览</h1>
        <p className="text-zinc-500">查看当前服务状态、订阅规模和最近一次配置应用结果。</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">工作方式</CardTitle>
            <Settings2 className="h-4 w-4 text-zinc-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatMode(status.app.mode)}</div>
            <p className="text-xs text-zinc-500 mt-1">
              {status.app.renderOnly ? '会生成配置文件，但不会自动接管代理核心' : '会把配置直接应用到代理核心'}
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
              已添加的订阅数量 / 已识别的节点数量
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">分组与入口</CardTitle>
            <Network className="h-4 w-4 text-zinc-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.counts.groups} / {status.counts.listeners}</div>
            <p className="text-xs text-zinc-500 mt-1">
              代理分组数量 / 代理入口数量
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">代理核心状态</CardTitle>
            <ShieldCheck className={`h-4 w-4 ${status.runtime.controllerReachable ? 'text-green-500' : 'text-yellow-500'}`} />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.runtime.controllerReachable ? '连接正常' : '尚未连接'}</div>
            <p className="text-xs text-zinc-500 mt-1 truncate">
              最近生成配置: {status.runtime.lastRenderAt ? format(new Date(status.runtime.lastRenderAt), 'PP HH:mm:ss', { locale: zhCN }) : '尚未生成'}
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>当前情况</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="space-y-4 text-sm">
              <div className="flex justify-between border-b pb-2">
                <dt className="text-zinc-500">服务名称</dt>
                <dd className="font-medium">{status.app.name}</dd>
              </div>
              <div className="flex justify-between border-b pb-2">
                <dt className="text-zinc-500">启动时间</dt>
                <dd className="font-medium">{format(new Date(status.app.startedAt), 'PP HH:mm:ss', { locale: zhCN })}</dd>
              </div>
              <div className="flex justify-between border-b pb-2">
                <dt className="text-zinc-500">当前工作方式</dt>
                <dd className="font-medium">{formatMode(status.runtime.mode)}</dd>
              </div>
              <div className="flex justify-between pb-2">
                <dt className="text-zinc-500">最近一次应用结果</dt>
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
