import { Link } from 'react-router-dom'
import useSWR from 'swr'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { ArrowRight, CheckCircle2, Link2, Network, Settings2, ShieldCheck, TriangleAlert } from 'lucide-react'

import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api, ApiError } from '@/services/api'

function formatMode(mode?: string) {
  return mode === 'render-only' ? '先保存配置，再手动应用' : '保存后可直接接管代理核心'
}

function buildNextSteps(setupRequired: boolean, warningCount: number) {
  if (setupRequired) {
    return [
      { title: '完成首次初始化', description: '先准备密码、订阅和本地入口，系统才会真正可用。', to: '/setup' },
      { title: '导入你的订阅', description: '把服务商给的订阅链接导入进来，先让节点进系统。', to: '/subscriptions' },
      { title: '创建本地入口', description: '配置一个可供浏览器或设备连接的本地代理端口。', to: '/listeners' },
    ]
  }
  if (warningCount > 0) {
    return [
      { title: '处理运行提醒', description: '先看运行状态页，定位当前还没准备好的部分。', to: '/status' },
      { title: '检查出口线路', description: '确认常用线路都有可选节点，没有空线路。', to: '/interfaces' },
      { title: '查看最近操作记录', description: '如果刚改过配置，可以从记录里追溯发生了什么。', to: '/events' },
    ]
  }
  return [
    { title: '继续整理出口线路', description: '把机场节点按地区或用途分组，后续绑定更省心。', to: '/interfaces' },
    { title: '准备新的本地入口', description: '给不同设备或场景分配独立入口，管理更清晰。', to: '/listeners' },
    { title: '检查 Mihomo 版本', description: '如果你要切换核心版本，可以去系统与核心页处理。', to: '/system' },
  ]
}

export default function Dashboard() {
  const { data: status, error } = useSWR('/api/status', api.getStatus)
  const { data: setupState } = useSWR('/api/setup-state', api.getSetupState)

  if (error) {
    return <div className="text-red-500">{error instanceof ApiError ? error.message : '加载失败'}</div>
  }

  if (!status) {
    return <div className="text-sm text-zinc-500">正在整理当前系统状态...</div>
  }

  const warningCount = status.warnings.length
  const setupRequired = Boolean(setupState?.required)
  const nextSteps = buildNextSteps(setupRequired, warningCount)

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Overview"
        title="先看系统现在能不能用"
        description="这里不会堆技术细节，只回答三件事：系统是否就绪、哪里还缺配置、你接下来最该点哪里。"
      />

      <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
        <NoticeCard
          tone={setupRequired || warningCount > 0 ? 'warning' : 'success'}
          icon={setupRequired || warningCount > 0 ? <TriangleAlert className="h-4 w-4" /> : <CheckCircle2 className="h-4 w-4" />}
          title={setupRequired ? '还差几步才能开始稳定使用' : warningCount > 0 ? '系统能运行，但还有提醒建议先处理' : '系统已经进入可用状态'}
          description={
            setupRequired
              ? '你现在可以继续浏览页面，但建议先完成初始化任务，避免后面每一步都被配置缺口打断。'
              : warningCount > 0
                ? '核心功能已经在线，但还有环境或运行提醒。先处理这些问题，后面会更省心。'
                : '订阅、线路、本地入口和运行环境目前都没有明显阻塞项，可以继续做更细的线路整理。'
          }
          action={
            <div className="flex flex-wrap gap-3">
              <Button asChild size="sm">
                <Link to={setupRequired ? '/setup' : warningCount > 0 ? '/status' : '/interfaces'}>
                  {setupRequired ? '继续初始化' : warningCount > 0 ? '查看运行状态' : '整理出口线路'}
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
            <CardDescription>这会影响你保存配置后，系统是否会立即接管代理核心。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="text-2xl font-semibold tracking-tight text-zinc-950">{formatMode(status.app.mode)}</div>
            <p className="text-sm leading-6 text-zinc-600">
              {status.app.renderOnly ? '当前更适合先确认配置内容，等你准备好再手动让它生效。' : '保存后系统会尽量直接应用到代理核心，适合日常管理。'}
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
          title="出口线路与本地入口"
          value={`${status.counts.groups} / ${status.counts.listeners}`}
          description="已整理好的出口线路 / 可供设备连接的本地入口"
          icon={<Network className="h-4 w-4" />}
        />
        <OverviewMetric
          title="代理核心状态"
          value={status.runtime.controllerReachable ? '连接正常' : '还没连上'}
          description={status.runtime.controllerReachable ? 'AXIS 当前能和 Mihomo 正常通信。' : '建议去运行状态页检查控制器地址和密钥。'}
          icon={<ShieldCheck className="h-4 w-4" />}
        />
        <OverviewMetric
          title="待处理提醒"
          value={String(warningCount)}
          description={warningCount === 0 ? '目前没有需要你立刻处理的提醒。' : '建议优先处理这些提醒，避免后续出现配置偏差。'}
          icon={<Settings2 className="h-4 w-4" />}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_1fr]">
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">你接下来最该做的三件事</CardTitle>
            <CardDescription>按这个顺序处理，最不容易走弯路。</CardDescription>
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
            <CardDescription>这里只保留你此刻真正会关心的信息。</CardDescription>
          </CardHeader>
          <CardContent>
            <dl className="space-y-4 text-sm">
              <SummaryRow label="服务名称" value={status.app.name} />
              <SummaryRow label="启动时间" value={format(new Date(status.app.startedAt), 'PP HH:mm:ss', { locale: zhCN })} />
              <SummaryRow label="当前工作方式" value={formatMode(status.runtime.mode)} />
              <SummaryRow label="最近一次应用结果" value={status.runtime.lastApplyMessage || '还没有应用记录'} />
              <SummaryRow label="本地入口是否已准备好" value={status.counts.listeners > 0 ? '已准备' : '还没有'} />
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
