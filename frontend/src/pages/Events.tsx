import useSWR from 'swr'
import { fetcher } from '@/services/api'
import { Card, CardContent } from '@/components/ui/card'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

export default function Events() {
  const { data: events } = useSWR('/api/events', fetcher, { refreshInterval: 5000 })

  const getBadgeStyle = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error': return 'bg-red-50 text-red-700 border-red-200'
      case 'warn': return 'bg-amber-50 text-amber-700 border-amber-200'
      default: return 'bg-blue-50 text-blue-700 border-blue-200'
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">事件日志</h1>
        <p className="text-zinc-500">查看控制面的重要变更与告警事件。</p>
      </div>

      <Card className="shadow-sm border-zinc-200">
        <CardContent className="p-0">
          {!events ? (
            <div className="p-8 text-center text-sm text-zinc-500">加载中...</div>
          ) : events.length === 0 ? (
            <div className="p-8 text-center text-sm text-zinc-500">暂无事件记录。</div>
          ) : (
            <div className="divide-y divide-zinc-100">
              {events.map((event: any, idx: number) => (
                <div key={idx} className="flex items-start gap-4 p-4 hover:bg-zinc-50 transition-colors">
                  <div className="flex-1 space-y-1 text-sm">
                    <div className="flex items-center justify-between">
                      <h4 className="font-medium text-zinc-900">{event.message}</h4>
                      <span className="text-xs text-zinc-400 shrink-0">
                        {format(new Date(event.at), 'MM-dd HH:mm:ss', { locale: zhCN })}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`text-[10px] px-1.5 py-0.5 rounded border uppercase font-medium ${getBadgeStyle(event.level)}`}>
                        {event.level}
                      </span>
                      <span className="text-xs text-zinc-500 font-mono bg-zinc-100 px-1.5 py-0.5 rounded">
                        {event.scope}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
