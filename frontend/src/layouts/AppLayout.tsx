import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Activity, Link as LinkIcon, Network, Radio, Settings, FileText, LogOut, RefreshCw, Wrench } from 'lucide-react';
import { Button } from '@/components/ui/button';
import useSWR from 'swr';
import { api, ApiError, type SessionStatusResponse, type SetupStateResponse } from '@/services/api';
import { useEffect } from 'react';
import { toast } from 'sonner';

const navItems = [
  { name: '总览', path: '/dashboard', icon: LayoutDashboard },
  { name: '状态', path: '/status', icon: Activity },
  { name: '订阅管理', path: '/subscriptions', icon: LinkIcon },
  { name: '出口管理', path: '/interfaces', icon: Network },
  { name: '入口管理', path: '/listeners', icon: Radio },
  { name: '系统配置', path: '/system', icon: Settings },
  { name: '事件日志', path: '/events', icon: FileText },
];

export default function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();

  const { data: session, error: sessionError } = useSWR<SessionStatusResponse, ApiError>('/api/session', api.getSession, {
    onErrorRetry: (error: ApiError) => {
      if (error.status === 401) return;
    }
  });
  const { data: setupState } = useSWR<SetupStateResponse, ApiError>(session?.authenticated ? '/api/setup-state' : null, api.getSetupState);

  useEffect(() => {
    if (sessionError?.status === 401 || (session && !session.authenticated)) {
      navigate('/login');
    }
  }, [session, sessionError, navigate]);

  useEffect(() => {
    if (!session?.authenticated || !setupState) {
      return;
    }
    if (setupState.required && (location.pathname === '/' || location.pathname === '/dashboard')) {
      navigate('/setup', { replace: true });
      return;
    }
    if (!setupState.required && location.pathname === '/setup') {
      navigate('/dashboard', { replace: true });
    }
  }, [session, setupState, location.pathname, navigate]);

  const handleLogout = async () => {
    try {
      await api.logout();
      navigate('/login', { replace: true });
    } catch {
      toast.error('退出登录失败');
    }
  };

  const handleReloadRuntime = async () => {
    try {
      const result = await api.reloadRuntime();
      toast.success(result.message || '最新配置已应用');
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '应用配置失败');
    }
  };

  if (!session?.authenticated) {
    return <div className="flex h-screen items-center justify-center">加载中...</div>;
  }

  return (
    <div className="flex min-h-screen bg-zinc-50/50">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 w-64 border-r bg-white pb-10">
        <div className="flex h-16 items-center px-6 border-b">
          <span className="text-sm font-bold tracking-widest text-zinc-800 uppercase">A X I S</span>
        </div>
        <div className="px-4 py-6">
          <h2 className="mb-2 px-2 text-lg font-semibold tracking-tight">控制台</h2>
          {setupState?.required && (
            <Link
              to="/setup"
              className={`mb-3 flex items-center gap-3 rounded-lg border px-3 py-2 text-sm font-medium transition-colors ${
                location.pathname === '/setup'
                  ? 'border-amber-300 bg-amber-50 text-amber-700'
                  : 'border-amber-200 bg-amber-50/70 text-amber-700 hover:bg-amber-100'
              }`}
            >
              <Wrench className="h-4 w-4" />
              首次初始化
            </Link>
          )}
          <nav className="space-y-1">
            {navItems.map((item) => {
              const isActive = location.pathname === item.path;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-zinc-100 text-zinc-900'
                      : 'text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900'
                  }`}
                >
                  <item.icon className="h-4 w-4" />
                  {item.name}
                </Link>
              );
            })}
          </nav>
        </div>
      </aside>

      {/* Main Content */}
      <main className="pl-64 flex-1">
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b bg-white/80 px-8 backdrop-blur-sm">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-zinc-500">
              {location.pathname === '/setup' ? '首次初始化' : navItems.find((i) => i.path === location.pathname)?.name}
            </span>
          </div>
          <div className="flex items-center gap-4">
            <Button variant="outline" size="sm" onClick={handleReloadRuntime}>
              <RefreshCw className="mr-2 h-4 w-4" />
              应用当前配置
            </Button>
            <Button variant="ghost" size="sm" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={handleLogout}>
              <LogOut className="mr-2 h-4 w-4" />
              退出登录
            </Button>
          </div>
        </header>
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
