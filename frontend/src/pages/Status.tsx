import useSWR from 'swr'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { toast } from 'sonner'
import { CheckCircle2, AlertTriangle, XCircle, RefreshCw } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { toastApiError } from '@/lib/toast-api-error'

function formatMode(mode?: string, renderOnly?: boolean) {
  const currentMode = mode || (renderOnly ? 'render-only' : 'managed')
  return currentMode === 'render-only' ? '当前只保存配置' : '当前会自动接管代理核心'
}

export default function Status() {
  const { data: controller, mutate: mutateController } = useSWR(apiKeys.controller, api.getController)
  const { data: preflight, mutate: mutatePreflight } = useSWR(apiKeys.runtimePreflight, api.getRuntimePreflight)

  const handleProbe = async () => {
    try {
      await api.probeController()
      mutateController()
      toast.success('检测完成')
    } catch (err) {
      toastApiError(err, '检测失败')
    }
  }

  const handleRefreshPreflight = () => {
    mutatePreflight()
    toast.success('检查结果已刷新')
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Runtime"
        title="先看系统能不能跑起来"
        description="这里主要看两件事：代理核心是否连上，运行环境是否就绪。"
      />

      <NoticeCard
        icon={<AlertTriangle className="h-4 w-4" />}
        title="什么时候需要看这里"
        description="节点刷不出、本地代理连不上、改了不生效，先看这里。"
      />

      <div className="grid gap-6 md:grid-cols-2">
        {/* Controller Status */}
        <Card className="flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="space-y-1">
              <CardTitle>代理核心连接</CardTitle>
              <CardDescription>确认代理核心是否已连上</CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={handleProbe}>
              重新检测
            </Button>
          </CardHeader>
          <CardContent className="flex-1">
            {!controller ? (
              <p className="text-sm text-zinc-500">加载中...</p>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <div className={`h-2.5 w-2.5 rounded-full ${controller.reachable ? 'bg-green-500' : 'bg-red-500'}`} />
                    <span className="font-medium">{controller.reachable ? '连接正常' : '暂时还没连上'}</span>
                  <span className="text-sm text-zinc-500 ml-2">({formatMode(controller.mode, controller.renderOnly)})</span>
                </div>
                <div className="bg-zinc-50 rounded-lg p-4 text-sm overflow-auto max-h-[300px]">
                  <div className="flex py-1 border-b border-zinc-100">
                    <span className="text-zinc-500 w-28 shrink-0">控制器地址</span>
                    <span className="text-zinc-900 break-all font-mono">{controller.baseUrl || '未配置'}</span>
                  </div>
                  <div className="flex py-1 border-b border-zinc-100">
                    <span className="text-zinc-500 w-28 shrink-0">当前状态</span>
                    <span className="text-zinc-900 break-all">{controller.message || '尚未检测'}</span>
                  </div>
                  <div className="flex py-1 border-b border-zinc-100">
                    <span className="text-zinc-500 w-28 shrink-0">连接密钥</span>
                    <span className="text-zinc-900">{controller.secretConfigured ? '已设置' : '未设置'}</span>
                  </div>
                  <div className="flex py-1 border-b border-zinc-100">
                    <span className="text-zinc-500 w-28 shrink-0">代理核心版本</span>
                    <span className="text-zinc-900">{controller.version || '尚未获取'}</span>
                  </div>
                  <div className="flex py-1">
                    <span className="text-zinc-500 w-28 shrink-0">最近检测时间</span>
                    <span className="text-zinc-900">{controller.checkedAt ? format(new Date(controller.checkedAt), 'PP HH:mm:ss', { locale: zhCN }) : '尚未检测'}</span>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Preflight Summary */}
        <Card className="flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <div className="space-y-1">
              <CardTitle>使用前检查</CardTitle>
              <CardDescription>检查目录、配置是否就绪</CardDescription>
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
                    {preflight.ready ? '可以正常使用' : '还有问题需要处理'}
                  </span>
                  <span className="text-sm text-zinc-500 ml-auto">
                    通过 {preflight.summary.passed} / 提醒 {preflight.summary.warnings} / 问题 {preflight.summary.errors}
                  </span>
                </div>

                <div className="space-y-3 max-h-[350px] overflow-y-auto pr-2">
                  {preflight.checks.map((check, idx: number) => (
                    <div key={idx} className="flex gap-3 p-3 rounded-lg border bg-card">
                      <div className="shrink-0 mt-0.5">
                        {check.level === 'error' ? (
                          <XCircle className="h-4 w-4 text-red-500" />
                        ) : check.level === 'warn' ? (
                          <AlertTriangle className="h-4 w-4 text-amber-500" />
                        ) : check.level === 'info' ? (
                          <RefreshCw className="h-4 w-4 text-blue-500" />
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
                    <h4 className="text-sm font-medium mb-2">建议你先处理这些项</h4>
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
