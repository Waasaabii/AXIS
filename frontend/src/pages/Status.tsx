import useSWR from 'swr'
import { fetcher } from '@/services/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { CheckCircle2, AlertTriangle, XCircle, RefreshCw } from 'lucide-react'

export default function Status() {
  const { data: controller, mutate: mutateController } = useSWR('/api/controller', fetcher)
  const { data: preflight, mutate: mutatePreflight } = useSWR('/api/runtime-preflight', fetcher)

  const handleProbe = async () => {
    try {
      await fetch('/api/controller/probe', { method: 'POST' })
      mutateController()
      toast.success('探测完成')
    } catch (err) {
      toast.error('探测失败')
    }
  }

  const handleRefreshPreflight = () => {
    mutatePreflight()
    toast.success('运行预检已刷新')
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">状态与预检</h1>
        <p className="text-zinc-500">查看 controller 状态和系统运行预检结果。</p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Controller Status */}
        <Card className="flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="space-y-1">
              <CardTitle>Controller 状态</CardTitle>
              <CardDescription>Mihomo 外部控制接口连通性</CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={handleProbe}>
              探测
            </Button>
          </CardHeader>
          <CardContent className="flex-1">
            {!controller ? (
              <p className="text-sm text-zinc-500">加载中...</p>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <div className={`h-2.5 w-2.5 rounded-full ${controller.reachable ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="font-medium">{controller.reachable ? '可达' : '不可达'}</span>
                  <span className="text-sm text-zinc-500 ml-2">({controller.mode || (controller.renderOnly ? 'render-only' : 'managed')})</span>
                </div>
                <div className="bg-zinc-50 rounded-lg p-4 text-sm font-mono overflow-auto max-h-[300px]">
                  {Object.entries(controller).map(([k, v]) => (
                    <div key={k} className="flex py-1 border-b border-zinc-100 last:border-0">
                      <span className="text-zinc-500 w-32 shrink-0">{k}</span>
                      <span className="text-zinc-900 break-all">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Preflight Summary */}
        <Card className="flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="space-y-1">
              <CardTitle>运行预检</CardTitle>
              <CardDescription>系统环境与配置合规性检查</CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={handleRefreshPreflight}>
              <RefreshCw className="h-4 w-4 mr-2" />
              刷新
            </Button>
          </CardHeader>
          <CardContent className="flex-1">
            {!preflight ? (
              <p className="text-sm text-zinc-500">加载中...</p>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center gap-2 pb-4 border-b">
                  {preflight.ready ? (
                    <CheckCircle2 className="h-5 w-5 text-green-500" />
                  ) : (
                    <AlertTriangle className="h-5 w-5 text-amber-500" />
                  )}
                  <span className="font-medium text-lg">
                    {preflight.ready ? '可进入联调' : '仍有阻塞项'}
                  </span>
                  <span className="text-sm text-zinc-500 ml-auto">
                    成功 {preflight.summary.passed} / 警告 {preflight.summary.warnings} / 错误 {preflight.summary.errors}
                  </span>
                </div>

                <div className="space-y-3 max-h-[350px] overflow-y-auto pr-2">
                  {preflight.checks.map((check: any, idx: number) => (
                    <div key={idx} className="flex gap-3 p-3 rounded-lg border bg-card">
                      <div className="shrink-0 mt-0.5">
                        {check.level === 'error' ? (
                          <XCircle className="h-4 w-4 text-red-500" />
                        ) : check.level === 'warn' ? (
                          <AlertTriangle className="h-4 w-4 text-amber-500" />
                        ) : (
                          <CheckCircle2 className="h-4 w-4 text-green-500" />
                        )}
                      </div>
                      <div>
                        <h4 className="text-sm font-medium">{check.title}</h4>
                        <p className="text-xs text-zinc-500 mt-1">{check.summary}</p>
                      </div>
                    </div>
                  ))}
                </div>

                {preflight.recommendations?.length > 0 && (
                  <div className="mt-4 pt-4 border-t">
                    <h4 className="text-sm font-medium mb-2">建议动作</h4>
                    <ul className="text-sm text-zinc-600 space-y-1 list-disc pl-4">
                      {preflight.recommendations.map((rec: string, i: number) => (
                        <li key={i}>{rec}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
