import { useEffect, useMemo } from 'react'
import useSWR from 'swr'
import { ArrowRight, CheckCircle2, Cpu, Download, LockKeyhole, MonitorSmartphone, RefreshCw, TriangleAlert } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api, ApiError, type BootstrapStatus } from '@/services/api'

function nextPath(nextStep?: string) {
  switch (nextStep) {
    case 'login':
      return '/login'
    case 'setup':
      return '/setup'
    case 'dashboard':
      return '/dashboard'
    default:
      return '/launch'
  }
}

function nextLabel(nextStep?: string) {
  switch (nextStep) {
    case 'login':
      return '进入登录'
    case 'setup':
      return '继续初始化'
    case 'dashboard':
      return '进入控制台'
    default:
      return '留在启动页'
  }
}

export default function Launch() {
  const navigate = useNavigate()
  const { data, error, isLoading, mutate } = useSWR<BootstrapStatus, ApiError>('/api/bootstrap/status', api.getBootstrapStatus, {
    refreshInterval: 3000,
  })

  const targetPath = useMemo(() => nextPath(data?.nextStep), [data?.nextStep])

  useEffect(() => {
    if (!data || data.nextStep === 'launch') {
      return
    }

    const timer = window.setTimeout(() => {
      navigate(targetPath, { replace: true })
    }, 900)

    return () => window.clearTimeout(timer)
  }, [data, navigate, targetPath])

  const cards = data
    ? [
        {
          title: '主服务',
          value: data.mainService.state === 'running' ? '已就绪' : data.mainService.state === 'attention' ? '需留意' : data.mainService.state === 'degraded' ? '降级中' : '未就绪',
          description: data.mainService.message,
          icon: <Cpu className="h-4 w-4" />,
        },
        {
          title: '宿主模式',
          value: data.host.desktopMode ? '桌面宿主' : 'Web 服务',
          description: data.host.desktopMode ? '当前通过 Wails 桌面壳运行。' : '当前通过 HTTP 服务模式运行。',
          icon: <MonitorSmartphone className="h-4 w-4" />,
        },
        {
          title: '登录状态',
          value: data.auth.authenticated ? '已登录' : '未登录',
          description: data.auth.authenticated ? '会话已经建立，可以直接进入下一步。' : data.setup.needsPasswordReset ? '当前还没完成首次初始化，下一步会先进入 Setup 创建管理员账号。' : '还没有有效会话，下一步会先进入登录页。',
          icon: <LockKeyhole className="h-4 w-4" />,
        },
        {
          title: '更新服务',
          value: data.updater.updateAvailable ? '发现新版本' : data.updater.running ? '已在线' : '未启用',
          description: data.updater.message || '当前未返回更新状态说明。',
          icon: <Download className="h-4 w-4" />,
        },
      ]
    : []

  return (
    <div className="relative min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.98),_rgba(244,244,245,0.95)_38%,_rgba(228,228,231,0.84)_100%)] px-4 py-8">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(120deg,rgba(24,24,27,0.02),transparent_35%,rgba(24,24,27,0.03))]" />

      <div className="relative mx-auto flex min-h-[calc(100vh-4rem)] max-w-6xl items-center">
        <div className="grid w-full gap-6 lg:grid-cols-[1.05fr_0.95fr]">
          <section className="space-y-6 rounded-[2rem] border border-white/80 bg-white/85 p-8 shadow-[0_30px_100px_-52px_rgba(24,24,27,0.4)] backdrop-blur">
            <div className="space-y-4">
              <div className="text-[11px] font-semibold uppercase tracking-[0.32em] text-zinc-400">AXIS Launch</div>
              <div className="space-y-3">
                <h1 className="max-w-2xl text-4xl font-semibold tracking-tight text-zinc-950 sm:text-5xl">
                  先判断系统现在处在哪一步，再决定把你送去哪里。
                </h1>
                <p className="max-w-2xl text-sm leading-7 text-zinc-600">
                  这里不是空欢迎页。AXIS 会先检查主服务、登录态、初始化状态和桌面更新服务，再把你送到最合适的下一步。
                </p>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              {cards.map((card) => (
                <Card key={card.title} className="border-zinc-200/80 bg-white/80">
                  <CardHeader className="space-y-3 pb-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-zinc-50 text-zinc-600">
                      {card.icon}
                    </div>
                    <div className="space-y-1">
                      <CardDescription className="text-xs uppercase tracking-[0.2em] text-zinc-400">{card.title}</CardDescription>
                      <CardTitle className="text-xl">{card.value}</CardTitle>
                    </div>
                  </CardHeader>
                  <CardContent className="pt-0 text-sm leading-6 text-zinc-600">
                    {card.description}
                  </CardContent>
                </Card>
              ))}
            </div>
          </section>

          <section className="space-y-5 rounded-[2rem] border border-zinc-200/80 bg-zinc-950 p-8 text-zinc-50 shadow-[0_30px_100px_-52px_rgba(24,24,27,0.55)]">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-zinc-300">
                {data?.mainService.ready ? <CheckCircle2 className="h-3.5 w-3.5" /> : <TriangleAlert className="h-3.5 w-3.5" />}
                启动决策
              </div>
              <h2 className="text-2xl font-semibold tracking-tight">当前下一步</h2>
              <p className="text-sm leading-7 text-zinc-300">
                {error
                  ? error.message
                  : data?.blockingReason
                    ? data.blockingReason
                    : data?.nextStep === 'setup' && !data.auth.authenticated
                      ? '当前还没创建管理员账号和密码，会先进入 Setup 完成首次初始化。'
                    : data?.nextStep === 'login'
                      ? '先建立管理员会话，再继续后面的初始化或管理动作。'
                    : data?.nextStep === 'setup'
                        ? '当前还没创建管理员账号和密码，会先进入 Setup 完成首次初始化。'
                        : data?.nextStep === 'dashboard'
                          ? data?.setup.required
                            ? '管理员账号已经准备好，可以先进入控制台；订阅、线路和入口仍会在控制台内继续提示你补齐。'
                            : '登录态和初始化检查都通过了，可以直接进入控制台。'
                          : '正在读取启动状态。'}
              </p>
            </div>

            <div className="space-y-3 rounded-3xl border border-white/10 bg-white/[0.03] p-5">
              <StatusRow
                label="初始化"
                value={data?.setup.required ? '未完成' : '已完成'}
                detail={data?.setup.required ? data.setup.reasons.join(' ') : '基础准备项已通过。'}
              />
              <StatusRow
                label="更新服务"
                value={data?.updater.running ? (data.updater.updateAvailable ? '有新版本' : '在线') : '离线'}
                detail={data?.updater.message || '当前没有可展示的更新服务状态。'}
              />
              <StatusRow
                label="下一跳"
                value={nextLabel(data?.nextStep)}
                detail={targetPath === '/launch' ? '当前不会跳转。' : `即将跳转到 ${targetPath}`}
              />
            </div>

            <div className="flex flex-wrap gap-3">
              <Button className="min-w-[160px]" onClick={() => navigate(targetPath, { replace: true })} disabled={isLoading || targetPath === '/launch'}>
                {isLoading ? '检查中...' : nextLabel(data?.nextStep)}
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
              <Button variant="outline" className="border-white/15 bg-transparent text-zinc-100 hover:bg-white/10 hover:text-white" onClick={() => mutate()}>
                <RefreshCw className="mr-2 h-4 w-4" />
                重新检测
              </Button>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}

function StatusRow({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <div className="rounded-2xl border border-white/8 bg-white/[0.02] px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <div className="text-xs uppercase tracking-[0.24em] text-zinc-500">{label}</div>
        <div className="text-sm font-medium text-zinc-100">{value}</div>
      </div>
      <p className="mt-2 text-sm leading-6 text-zinc-400">{detail}</p>
    </div>
  )
}
