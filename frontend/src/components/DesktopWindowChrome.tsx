import useSWR from 'swr'
import { Minus, Square, X } from 'lucide-react'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { hasWailsRuntime } from '@/services/wails-provider'
import { Quit, WindowMinimise, WindowToggleMaximise } from '../../wailsjs/runtime/runtime'

type DesktopPlatform = 'darwin' | 'windows' | 'linux' | string

interface DesktopWindowChromeProps {
  title: string
  description?: string
}

export function DesktopWindowChrome({ title }: DesktopWindowChromeProps) {
  const { data: host } = useSWR(apiKeys.hostStatus, api.getHostStatus, { refreshInterval: 10000 })
  if (!hasWailsRuntime() || !host?.desktopMode) return null
  return <DesktopTitlebar platform={host.platform || 'linux'} pageTitle={title} />
}

function DesktopTitlebar({ platform, pageTitle }: { platform: DesktopPlatform; pageTitle: string }) {
  const isMac = platform === 'darwin'
  const isWindows = platform === 'windows'
  return (
    <div
      className={cn(
        'sticky top-0 z-30 grid h-12 select-none items-center border-b border-zinc-200/70 bg-white/88 backdrop-blur-2xl',
        isMac && 'grid-cols-[88px_auto_1fr] pl-4 pr-4',
        isWindows && 'grid-cols-[auto_1fr_auto] pl-4',
        !isMac && !isWindows && 'grid-cols-[auto_1fr] px-4',
      )}
      style={{ '--wails-draggable': 'drag' } as React.CSSProperties}
    >
      {isMac ? <TrafficLightSpace /> : null}
      <AppTitle pageTitle={pageTitle} />
      <DragRegion />
      {isWindows ? <WindowControls /> : null}
    </div>
  )
}

function TrafficLightSpace() {
  return <div className="h-full" aria-hidden="true" />
}

function AppTitle({ pageTitle }: { pageTitle: string }) {
  return (
    <div className="flex h-full min-w-[220px] max-w-[360px] items-center gap-3 pr-6" style={{ '--wails-draggable': 'drag' } as React.CSSProperties}>
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-zinc-950 text-[11px] font-semibold tracking-tight text-white shadow-sm">AX</div>
      <div className="min-w-0 leading-none">
        <div className="truncate text-[13px] font-semibold text-zinc-950">AXIS</div>
        <div className="mt-1 truncate text-[11px] text-zinc-500">{pageTitle}</div>
      </div>
    </div>
  )
}

function DragRegion() {
  return <div className="h-full min-w-16" style={{ '--wails-draggable': 'drag' } as React.CSSProperties} aria-hidden="true" />
}

function WindowControls() {
  return (
    <div className="flex h-full shrink-0" style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}>
      <WindowButton label="最小化" onClick={() => WindowMinimise()}><Minus className="h-4 w-4" /></WindowButton>
      <WindowButton label="最大化" onClick={() => WindowToggleMaximise()}><Square className="h-3.5 w-3.5" /></WindowButton>
      <WindowButton label="关闭" danger onClick={() => Quit()}><X className="h-4 w-4" /></WindowButton>
    </div>
  )
}

function WindowButton({ label, danger, onClick, children }: { label: string; danger?: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <Button
      type="button"
      variant="ghost"
      aria-label={label}
      title={label}
      onClick={onClick}
      className={cn('h-full w-12 rounded-none text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950', danger && 'hover:bg-red-500 hover:text-white')}
    >
      {children}
    </Button>
  )
}
