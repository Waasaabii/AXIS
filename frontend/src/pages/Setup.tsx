import { Link, useNavigate } from 'react-router-dom'
import useSWR from 'swr'
import { KeyRound, Radio, Route, Server, ShieldCheck } from 'lucide-react'
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
      <div className="space-y-6">
        <PageHeader eyebrow="Setup" title="首次初始化已经完成" description="节点来源、线路和使用方式已经准备好。现在可以进入使用页。" />
        <NoticeCard tone="success" icon={<ShieldCheck className="h-4 w-4" />} title="可以开始使用" description="进入使用页复制代理地址，或继续切换这台设备使用的线路。" action={<Button onClick={() => navigate(session?.authenticated ? '/usage' : '/login')}>{session?.authenticated ? '进入使用页' : '去登录'}</Button>} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader eyebrow="Setup" title="先完成这几步，再开始使用" description="按顺序补齐管理员密码、节点来源、线路和这台设备的使用方式。" />
      <NoticeCard icon={<ShieldCheck className="h-4 w-4" />} title="按提示补齐缺口即可" description="缺什么就先处理什么。准备好后，使用页会显示代理地址和当前线路。" />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StepCard icon={<KeyRound className="h-5 w-5" />} title="设置管理员密码" description={setup?.needsPasswordReset ? '先保护控制台，避免别人直接进入。' : '管理员密码已设置。'} to="/setup" active={Boolean(setup?.needsPasswordReset)} />
        <StepCard icon={<Server className="h-5 w-5" />} title="添加节点来源" description="添加订阅、单个代理、其他 AXIS 或本机节点。" to="/routes" active={!setupModel?.hasNodeSources} />
        <StepCard icon={<Route className="h-5 w-5" />} title="创建线路" description="把节点来源组合成这台设备可以使用或发布的线路。" to="/routes" active={!setupModel?.hasRoutes} />
        <StepCard icon={<Radio className="h-5 w-5" />} title="选择使用方式" description="选择当前线路，开启本机代理或虚拟网口。" to="/usage" active={!setupModel?.hasUsage} />
      </div>
      {setup?.reasons && setup.reasons.length > 0 ? <Card className="border-zinc-200 shadow-sm"><CardHeader><CardTitle className="text-base">当前还缺什么</CardTitle><CardDescription>处理完这些项目后，就可以正常使用。</CardDescription></CardHeader><CardContent className="space-y-2 text-sm text-zinc-600">{setup.reasons.map((reason) => <div key={reason} className="rounded-xl bg-zinc-50 px-3 py-2">{reason}</div>)}</CardContent></Card> : null}
    </div>
  )
}

function StepCard({ icon, title, description, to, active }: { icon: React.ReactNode; title: string; description: string; to: string; active: boolean }) {
  return (
    <Card className={active ? 'border-amber-200 bg-amber-50/80 shadow-sm' : 'border-zinc-200 shadow-sm'}>
      <CardHeader>
        <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-full bg-white text-zinc-700 shadow-sm">{icon}</div>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription className="leading-6">{description}</CardDescription>
      </CardHeader>
      <CardContent><Button asChild variant={active ? 'default' : 'outline'} size="sm"><Link to={to}>{active ? '去处理' : '查看'}</Link></Button></CardContent>
    </Card>
  )
}
