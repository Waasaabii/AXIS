import { Outlet, useLocation } from 'react-router-dom'
import { DesktopWindowChrome } from '@/components/DesktopWindowChrome'

const pageTitles: Record<string, string> = {
  '/launch': '启动检查',
  '/login': '登录',
  '/setup': '首次初始化',
  '/home': '首页',
  '/egress': '出口',
  '/rules': '规则',
  '/publications': '发布',
  '/diagnostics': '诊断',
  '/system': '系统',
}

export default function DesktopShell() {
  const location = useLocation()
  const title = pageTitles[location.pathname] || 'AXIS'
  return (
    <div className="min-h-screen">
      <DesktopWindowChrome title={title} />
      <Outlet />
    </div>
  )
}
