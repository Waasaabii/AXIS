import { useEffect, useMemo, useState } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Activity, Home, LogOut, Menu, RefreshCw, Route, Settings, Share2, Sparkles, Wrench } from 'lucide-react'
import { Button } from '@/components/ui/button'
import useSWR from 'swr'
import { api, ApiError, type SessionStatusResponse, type SetupStateResponse } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { NoticeCard } from '@/components/NoticeCard'
import { toastApiError } from '@/lib/toast-api-error'

const navItems = [
  { name: '首页', path: '/home', icon: Home, description: '查看现在能不能用，并快速开始。' },
  { name: '出口', path: '/egress', icon: Route, description: '添加节点来源，创建这台设备的节点。' },
  { name: '规则', path: '/rules', icon: Sparkles, description: '让 AXIS 自动选择直连或出口。' },
  { name: '发布', path: '/publications', icon: Share2, description: '把线路提供给其他设备或 AXIS。' },
  { name: '诊断', path: '/diagnostics', icon: Activity, description: '查看问题、检查结果和最近记录。' },
  { name: '系统', path: '/system', icon: Settings, description: '密码、高级配置、核心版本。' },
]

export default function AppLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const [navOpen, setNavOpen] = useState(false)
  const { data: session, error: sessionError } = useSWR<SessionStatusResponse, ApiError>(apiKeys.session, api.getSession, { onErrorRetry: (error: ApiError) => { if (error.status === 401) return } })
  const { data: setupState } = useSWR<SetupStateResponse, ApiError>(session?.authenticated ? apiKeys.setupState : null, api.getSetupState)

  useEffect(() => {
    if (sessionError?.status === 401 || (session && !session.authenticated)) navigate('/launch', { replace: true })
  }, [session, sessionError, navigate])

  useEffect(() => {
    if (!session?.authenticated || !setupState) return
    if (setupState.needsPasswordReset && (location.pathname === '/' || location.pathname === '/home')) {
      navigate('/setup', { replace: true })
      return
    }
    if (!setupState.required && location.pathname === '/setup') navigate('/home', { replace: true })
  }, [session, setupState, location.pathname, navigate])

  const currentPage = useMemo(() => location.pathname === '/setup' ? { name: '首次初始化', description: '先完成最少准备项，再开始使用。' } : navItems.find((item) => item.path === location.pathname) ?? navItems[0], [location.pathname])

  const handleLogout = async () => {
    try { await api.logout(); navigate('/launch', { replace: true }) } catch { toast.error('退出登录失败') }
  }

  const handleReloadRuntime = async () => {
    try { const result = await api.reloadRuntime(); toast.success(result.message || '最新配置已应用') } catch (err) { toastApiError(err, '应用配置失败') }
  }

  if (!session?.authenticated) return <div className="flex h-screen items-center justify-center">加载中...</div>

  const navContent = (
    <>
      {setupState?.required ? <Link to="/setup" className="mb-3 flex items-start gap-3 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800"><Wrench className="mt-0.5 h-4 w-4 shrink-0" /><div><div className="font-medium">首次初始化</div><div className="text-xs leading-5 text-amber-700">先补齐使用前需要的准备。</div></div></Link> : null}
      {navItems.map((item) => {
        const isActive = location.pathname === item.path
        return <Link key={item.path} to={item.path} onClick={() => setNavOpen(false)} className={`flex items-start gap-3 rounded-2xl px-4 py-3 transition-colors ${isActive ? 'bg-zinc-950 text-white shadow-sm' : 'text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950'}`}><item.icon className={`mt-0.5 h-4 w-4 shrink-0 ${isActive ? 'text-white' : 'text-zinc-400'}`} /><div><div className="text-sm font-medium">{item.name}</div><div className={`text-xs leading-5 ${isActive ? 'text-zinc-300' : 'text-zinc-500'}`}>{item.description}</div></div></Link>
      })}
    </>
  )

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.96),_rgba(244,244,245,0.94)_38%,_rgba(244,244,245,0.9)_100%)]">
      <div className="mx-auto flex min-h-screen max-w-[1600px]">
        <aside className="hidden w-72 shrink-0 border-r border-zinc-200/80 bg-white/80 px-5 py-6 backdrop-blur xl:flex xl:flex-col">
          <div className="space-y-1 px-3 pb-6"><div className="text-[11px] font-semibold uppercase tracking-[0.28em] text-zinc-400">AXIS Console</div><div className="text-lg font-semibold tracking-tight text-zinc-950">控制台</div><p className="text-sm leading-6 text-zinc-500">使用、出口、规则和发布都在这里完成。</p></div>
          <nav className="space-y-1">{navContent}</nav>
        </aside>
        <main className="min-w-0 flex-1">
          <header className="sticky top-0 z-20 border-b border-zinc-200/70 bg-white/80 backdrop-blur-xl">
            <div className="mx-auto flex min-h-20 max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6 lg:px-8">
              <div className="flex min-w-0 items-center gap-3"><Button variant="outline" size="icon" className="xl:hidden" onClick={() => setNavOpen(true)}><Menu className="h-4 w-4" /></Button><div className="min-w-0"><div className="truncate text-sm font-medium text-zinc-950">{currentPage.name}</div><div className="truncate text-sm text-zinc-500">{currentPage.description}</div></div></div>
              <div className="flex items-center gap-2"><Button variant="outline" size="sm" onClick={handleReloadRuntime}><RefreshCw className="mr-2 h-4 w-4" />让当前更改生效</Button><Button variant="ghost" size="sm" className="text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900" onClick={handleLogout}><LogOut className="mr-2 h-4 w-4" />退出</Button></div>
            </div>
          </header>
          <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8">
            {setupState?.required && location.pathname !== '/setup' ? <NoticeCard tone="warning" icon={<Wrench className="h-4 w-4" />} title="还没完成首次初始化" description="可以先浏览页面，但建议先补齐节点来源、线路和使用方式。" action={<Button asChild size="sm"><Link to="/setup">继续完成初始化</Link></Button>} /> : null}
            <Outlet />
          </div>
        </main>
      </div>
      <Dialog open={navOpen} onOpenChange={setNavOpen}><DialogContent className="max-w-sm rounded-3xl p-0"><DialogHeader className="border-b border-zinc-200 px-5 py-5"><DialogTitle className="text-left text-lg">AXIS</DialogTitle><DialogDescription className="text-left">选择你现在要处理的任务。</DialogDescription></DialogHeader><div className="space-y-1 px-3 py-3">{navContent}</div></DialogContent></Dialog>
    </div>
  )
}
