import { Link } from 'react-router-dom'
import useSWR from 'swr'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { ArrowRight, CheckCircle2, Link2, Network, Settings2, ShieldCheck, TriangleAlert } from 'lucide-react'

import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api, ApiError, type StatusResponse } from '@/services/api'
import { apiKeys } from '@/services/api-keys'

function formatMode(mode?: string) {
  return mode === 'render-only' ? '只保存配置' : '保存后自动生效'
}

type DashboardRuntimeState = StatusResponse['runtime']

function runtimeActionPath(action?: string) {
  switch (action) {
    case 'manage-core':
      return '/system'
    case 'check-controller':
    case 'view-runtime':
      return '/status'
    default:
      return '/interfaces'
  }
}

function runtimeActionLabel(action?: string) {
  switch (action) {
    case 'manage-core':
      return '去维护代理核心'
    case 'check-controller':
      return '重新检查连接'
    case 'view-runtime':
      return '查看运行状态'
    default:
      return '整理出口线路'
  }
}

function formatRuntimeState(runtime: DashboardRuntimeState) {
  switch (runtime.state) {
    case 'ready':
      return '连接正常'
    case 'core-missing':
      return '未准备好'
    case 'controller-unreachable':
      return '还没连上'
    case 'apply-failed':
      return '应用失败'
    case 'render-only':
      return '只保存配置'
    default:
      return runtime.controllerReachable ? '连接正常' : '还没连上'
  }
}

function buildNextSteps(setupRequired: boolean, runtime: DashboardRuntimeState, warningCount: number) {
  if (setupRequired) {
    return [
      { title: '完成首次初始化', description: '先把密码、订阅和本地代理配好，系统才真正可用。', to: '/setup' },
      { title: '导入订阅', description: '填入订阅链接，让节点进系统。', to: '/subscriptions' },
      { title: '配置本地代理', description: '生成一个设备可连接的代理地址。', to: '/listeners' },
    ]
  }
  if (runtime.state === 'core-missing') {
    return [
      { title: '维护代理核心', description: '安装、激活或重新指定代理核心程序。', to: '/system' },
      { title: '查看运行状态', description: '确认当前缺什么，以及下一步怎么处理。', to: '/status' },
      { title: '查看最近记录', description: '回溯最近一次检测结果。', to: '/events' },
    ]
  }
  if (runtime.state === 'controller-unreachable') {
    return [
      { title: '重新检查连接', description: '确认代理核心已经启动，并重新检测连接。', to: '/status' },
      { title: '维护代理核心', description: '需要切换版本或路径时，在系统页处理。', to: '/system' },
      { title: '查看最近记录', description: '回溯最近一次连接失败原因。', to: '/events' },
    ]
  }
  if (warningCount > 0) {
    return [
      { title: '处理运行提醒', description: '先看运行状态，定位缺口。', to: '/status' },
      { title: '检查出口线路', description: '确认常用线路都有可用节点。', to: '/interfaces' },
      { title: '查看最近记录', description: '回溯最近变更和原因。', to: '/events' },
    ]
  }
  return [
    { title: '继续整理出口线路', description: '按地区或用途整理节点。', to: '/interfaces' },
    { title: '再建一个本地代理', description: '给不同设备单独一个地址。', to: '/listeners' },
    { title: '检查核心版本', description: '需要切换时去系统页处理。', to: '/system' },
  ]
}

export default function Dashboard() {
  const { data: status, error } = useSWR(apiKeys.status, api.getStatus)
  const { data: setupState } = useSWR(apiKeys.setupState, api.getSetupState)

  if (error) {
    return <div className="text-red-500">{error instanceof ApiError ? error.message : '加载失败'}</div>
  }

  if (!status) {
    return <div className="text-sm text-zinc-500">正在整理当前系统状态...</div>
  }

  const warningCount = status.warnings.length
  const setupRequired = Boolean(setupState?.required)
  const nextSteps = buildNextSteps(setupRequired, status.runtime, warningCount)
  const runtimeAction = runtimeActionPath(status.runtime.action)
  const runtimeMessage = status.runtime.message || status.warnings[0] || '当前没有明显阻塞项，可以继续整理线路。'

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Overview"
        title="先看系统现在能不能用"
        description="这里只告诉你三件事：能不能用、还缺什么、下一步去哪。"
      />

      <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
        <NoticeCard
          tone={setupRequired || warningCount > 0 ? 'warning' : 'success'}
          icon={setupRequired || warningCount > 0 ? <TriangleAlert className="h-4 w-4" /> : <CheckCircle2 className="h-4 w-4" />}
          title={setupRequired ? '还差几步才能开始稳定使用' : warningCount > 0 ? '代理核心还需要处理' : '系统已经进入可用状态'}
          description={
            setupRequired
              ? '建议先完成初始化，避免后面每一步都被缺口打断。'
              : warningCount > 0
                ? runtimeMessage
                : '目前没有明显阻塞项，可以继续整理线路。'
          }
          action={
            <div className="flex flex-wrap gap-3">
              <Button asChild size="sm">
                <Link to={setupRequired ? '/setup' : warningCount > 0 ? runtimeAction : '/interfaces'}>
                  {setupRequired ? '继续初始化' : warningCount > 0 ? runtimeActionLabel(status.runtime.action) : '整理出口线路'}
                </Link>
              </Button>
              <Button asChild size="sm" variant="outline">
                <Link to="/events">查看最近记录</Link>
              </Button>
            </div>
          }
        />

        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">当前工作方式</CardTitle>
            <CardDescription>这会影响你保存配置后是否会自动生效。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-2xl font-semibold tracking-tight text-zinc-950">{formatMode(status.app.mode)}</div>
            <p className="text-sm leading-6 text-zinc-600">
              {status.app.renderOnly ? '当前只会保存配置，不会自动生效。' : '保存后会尽量自动应用。'}
            </p>
            <div className="rounded-2xl border border-zinc-200 bg-zinc-50 px-4 py-3 text-sm text-zinc-600">
              最近一次生成配置：
              <span className="ml-2 font-medium text-zinc-900">
                {status.runtime.lastRenderAt ? format(new Date(status.runtime.lastRenderAt), 'PP HH:mm:ss', { locale: zhCN }) : '还没有生成记录'}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <OverviewMetric
          title="订阅与节点"
          value={`${status.counts.providers} / ${status.counts.nodes}`}
          description="已接入的订阅数量 / 当前识别到的节点总数"
          icon={<Link2 className="h-4 w-4" />}
        />
        <OverviewMetric
          title="出口线路与本地代理"
          value={`${status.counts.groups} / ${status.counts.listeners}`}
          description="已整理的出口线路 / 已配置的本地代理"
          icon={<Network className="h-4 w-4" />}
        />
        <OverviewMetric
          title="代理核心状态"
          value={formatRuntimeState(status.runtime)}
          description={status.runtime.message || (status.runtime.controllerReachable ? '代理核心已连接。' : '建议去运行状态页检查地址和密钥。')}
          icon={<ShieldCheck className="h-4 w-4" />}
        />
        <OverviewMetric
          title="待处理提醒"
          value={String(warningCount)}
          description={warningCount === 0 ? '当前没有需要立刻处理的提醒。' : '建议先处理这些提醒。'}
          icon={<Settings2 className="h-4 w-4" />}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_1fr]">
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">下一步建议</CardTitle>
            <CardDescription>按这个顺序来，少走弯路。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {nextSteps.map((item, index) => (
              <Link
                key={item.to}
                to={item.to}
                className="flex items-center justify-between rounded-2xl border border-zinc-200 bg-white px-4 py-4 transition-colors hover:border-zinc-300 hover:bg-zinc-50"
              >
                <div className="min-w-0 space-y-1">
                  <div className="flex items-center gap-2 text-sm font-medium text-zinc-950">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-zinc-100 text-xs text-zinc-700">
                      {index + 1}
                    </span>
                    {item.title}
                  </div>
                  <p className="text-sm leading-6 text-zinc-600">{item.description}</p>
                </div>
                <ArrowRight className="ml-4 h-4 w-4 shrink-0 text-zinc-400" />
              </Link>
            ))}
          </CardContent>
        </Card>

        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">当前情况</CardTitle>
            <CardDescription>只展示关键状态。</CardDescription>
          </CardHeader>
          <CardContent>
            <dl className="space-y-4 text-sm">
              <SummaryRow label="服务名称" value={status.app.name} />
              <SummaryRow label="启动时间" value={format(new Date(status.app.startedAt), 'PP HH:mm:ss', { locale: zhCN })} />
              <SummaryRow label="当前工作方式" value={formatMode(status.runtime.mode)} />
              <SummaryRow label="代理核心状态" value={status.runtime.message || formatRuntimeState(status.runtime)} />
              <SummaryRow label="最近一次应用结果" value={status.runtime.lastApplyMessage || '还没有应用记录'} />
              <SummaryRow label="本地代理是否已配置" value={status.counts.listeners > 0 ? '已配置' : '还没有'} />
            </dl>
          </CardContent>
        </Card>
      </div>

      {warningCount > 0 ? (
        <Card className="border-amber-200 bg-amber-50/70 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">现在最值得先处理的提醒</CardTitle>
            <CardDescription>这些提醒不会阻止你继续操作，但通常会影响后面的可用性和体验。</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3 text-sm text-amber-900">
              {status.warnings.map((warning: string, index: number) => (
                <li key={index} className="rounded-2xl border border-amber-200 bg-white/70 px-4 py-3">
                  {warning}
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function OverviewMetric({
  title,
  value,
  description,
  icon,
}: {
  title: string
  value: string
  description: string
  icon: React.ReactNode
}) {
  return (
    <Card className="border-zinc-200 shadow-sm">
      <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0 pb-3">
        <div className="space-y-1">
          <CardTitle className="text-sm font-medium text-zinc-700">{title}</CardTitle>
          <CardDescription className="text-sm leading-6 text-zinc-500">{description}</CardDescription>
        </div>
        <div className="flex h-9 w-9 items-center justify-center rounded-full border border-zinc-200 bg-zinc-50 text-zinc-500">
          {icon}
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold tracking-tight text-zinc-950">{value}</div>
      </CardContent>
    </Card>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-zinc-100 pb-3 last:border-b-0 last:pb-0">
      <dt className="text-zinc-500">{label}</dt>
      <dd className="max-w-[60%] text-right font-medium text-zinc-900">{value}</dd>
    </div>
  )
}
