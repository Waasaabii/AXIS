import { Link, useNavigate } from 'react-router-dom'
import useSWR from 'swr'
import { ArrowRight, CheckCircle2, KeyRound, Radio, Route, Server, ShieldCheck } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { NoticeCard } from '@/components/NoticeCard'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'

export default function Setup() {
  const navigate = useNavigate()
  const { data: session } = useSWR(apiKeys.session, api.getSession)
  const { data: setup } = useSWR(apiKeys.setupState, api.getSetupState)
  const setupModel = setup as typeof setup & { hasNodeSources?: boolean; hasRoutes?: boolean; hasUsage?: boolean }

  if (setup && !setup.required) {
    return (
      <main className="mx-auto flex min-h-[calc(100vh-3rem)] max-w-5xl items-center px-4 py-6 sm:px-6 lg:px-8">
        <div className="w-full space-y-4">
          <PageHeader eyebrow="Setup" title="首次初始化已经完成" description="节点来源、线路和使用方式已经准备好。现在可以进入首页。" />
          <NoticeCard tone="success" icon={<ShieldCheck className="h-4 w-4" />} title="可以开始使用" description="进入首页查看当前状态，或继续切换这台设备使用的线路。" action={<Button onClick={() => navigate(session?.authenticated ? '/home' : '/login')}>{session?.authenticated ? '进入首页' : '去登录'}</Button>} />
        </div>
      </main>
    )
  }

  return (
    <main className="mx-auto grid min-h-[calc(100vh-3rem)] max-w-6xl content-center gap-4 px-4 py-4 sm:px-6 lg:px-8">
      <PageHeader eyebrow="Setup" title="先完成这几步，再开始使用" description="按顺序补齐管理员密码、节点来源、线路和这台设备的使用方式。" />
      <div className="grid gap-4 lg:grid-cols-[0.9fr_1.1fr] lg:items-start">
        <section className="space-y-3">
          <NoticeCard className="border-dashed" icon={<ShieldCheck className="h-4 w-4" />} title="按提示补齐缺口即可" description="缺什么就先处理什么。准备好后，使用页会显示代理地址和当前线路。" />
          {setup?.reasons && setup.reasons.length > 0 ? <MissingCard reasons={setup.reasons} /> : null}
        </section>

        <section className="grid gap-3 sm:grid-cols-2">
          <StepCard icon={<KeyRound className="h-4 w-4" />} index="01" title="设置管理员密码" description={setup?.needsPasswordReset ? '先保护控制台。' : '管理员密码已设置。'} to="/setup" active={Boolean(setup?.needsPasswordReset)} />
          <StepCard icon={<Server className="h-4 w-4" />} index="02" title="添加节点来源" description="添加订阅、单个代理、其他 AXIS 或本机节点。" to="/egress" active={!setupModel?.hasNodeSources} />
          <StepCard icon={<Route className="h-4 w-4" />} index="03" title="创建线路" description="把节点来源组合成可使用或发布的线路。" to="/egress" active={!setupModel?.hasRoutes} />
          <StepCard icon={<Radio className="h-4 w-4" />} index="04" title="选择使用方式" description="选择当前线路，开启本机代理或虚拟网口。" to="/home" active={!setupModel?.hasUsage} />
        </section>
      </div>
    </main>
  )
}

function MissingCard({ reasons }: { reasons: string[] }) {
  return (
    <Card className="border-zinc-200 bg-white/90 shadow-sm">
      <CardHeader className="px-4 py-3">
        <CardTitle className="text-sm">当前还缺什么</CardTitle>
        <CardDescription className="text-xs">处理完这些项目后，就可以正常使用。</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-2 px-4 pb-4 text-sm text-zinc-600">
        {reasons.slice(0, 4).map((reason) => <div key={reason} className="rounded-xl bg-zinc-50 px-3 py-2 leading-5">{reason}</div>)}
      </CardContent>
    </Card>
  )
}

function StepCard({ icon, index, title, description, to, active }: { icon: React.ReactNode; index: string; title: string; description: string; to: string; active: boolean }) {
  return (
    <Card className={active ? 'border-amber-200 bg-amber-50/85 shadow-sm' : 'border-zinc-200 bg-white/90 shadow-sm'}>
      <CardHeader className="space-y-3 px-4 py-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-white text-zinc-700 shadow-sm">{active ? icon : <CheckCircle2 className="h-4 w-4 text-emerald-600" />}</div>
            <span className="text-xs font-semibold tracking-[0.2em] text-zinc-400">{index}</span>
          </div>
          <Button asChild variant={active ? 'default' : 'outline'} size="sm" className="h-8">
            <Link to={to}>{active ? '去处理' : '查看'}<ArrowRight className="h-3.5 w-3.5" /></Link>
          </Button>
        </div>
        <div className="space-y-1">
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription className="min-h-10 text-sm leading-5">{description}</CardDescription>
        </div>
      </CardHeader>
    </Card>
  )
}
