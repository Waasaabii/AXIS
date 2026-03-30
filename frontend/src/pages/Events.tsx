import useSWR from 'swr'
import { api, type EventEntry } from '@/services/api'
import { EmptyStateCard } from '@/components/EmptyStateCard'
import { PageHeader } from '@/components/PageHeader'
import { Card, CardContent } from '@/components/ui/card'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { FileText } from 'lucide-react'

const levelLabelMap: Record<string, string> = {
  error: '问题',
  warn: '提醒',
  info: '信息',
}

const scopeLabelMap: Record<string, string> = {
  auth: '登录',
  config: '配置',
  controller: '代理核心连接',
  group: '出口线路',
  healthcheck: '健康检查',
  provider: '订阅',
  'provider-test': '订阅测试',
  reload: '配置应用',
  render: '配置生成',
  runtime: '代理核心',
  'runtime-apply': '配置下发',
  setup: '首次初始化',
}

function sanitizeEventMessage(message: string) {
  return message
    .replace(/^未找到 mihomo 二进制: /, '未找到代理核心程序：')
    .replace(/^已生成 runtime\/mihomo\.yaml$/, '最新代理核心配置已生成')
    .replace('当前为 render-only 模式，仅完成配置渲染', '当前只保存配置，暂不会自动应用到代理核心。')
    .replace('当前为 render-only 模式，未接入运行态 controller。', '当前只保存配置，还没有接管代理核心。')
    .replace(/^controller 返回 (\d+)$/, '连接代理核心失败（状态码 $1）')
    .replace(/^controller 可达$/, '代理核心连接正常')
    .replace(/出口组/g, '出口线路')
    .replace(/入口监听/g, '本地入口')
}

export default function Events() {
  const { data: events } = useSWR('/api/events', api.getEvents, { refreshInterval: 5000 })

  const getBadgeStyle = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error': return 'bg-red-50 text-red-700 border-red-200'
      case 'warn': return 'bg-amber-50 text-amber-700 border-amber-200'
      default: return 'bg-blue-50 text-blue-700 border-blue-200'
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="History"
        title="看看最近到底发生了什么"
        description="这里记录的是 AXIS 最近的重要变化、提醒和失败原因。改完配置以后，如果结果和预期不一样，先来这里看。"
      />

      <Card className="shadow-sm border-zinc-200">
        <CardContent className="p-0">
          {!events ? (
            <div className="p-8 text-center text-sm text-zinc-500">加载中...</div>
          ) : events.length === 0 ? (
            <div className="p-6">
              <EmptyStateCard
                icon={<FileText className="h-5 w-5" />}
                title="暂时还没有操作记录"
                description="这通常表示你刚启动系统，或者最近还没有执行订阅刷新、配置保存、入口变更这类操作。"
              />
            </div>
          ) : (
            <div className="divide-y divide-zinc-100">
              {events.map((event: EventEntry, idx: number) => (
                <div key={idx} className="flex items-start gap-4 p-4 hover:bg-zinc-50 transition-colors">
                  <div className="flex-1 space-y-1 text-sm">
                    <div className="flex items-center justify-between">
                      <h4 className="font-medium text-zinc-900">{sanitizeEventMessage(event.message)}</h4>
                      <span className="text-xs text-zinc-400 shrink-0">
                        {format(new Date(event.at), 'MM-dd HH:mm:ss', { locale: zhCN })}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`text-[10px] px-1.5 py-0.5 rounded border uppercase font-medium ${getBadgeStyle(event.level)}`}>
                        {levelLabelMap[event.level] || event.level}
                      </span>
                      <span className="text-xs text-zinc-500 bg-zinc-100 px-1.5 py-0.5 rounded">
                        {scopeLabelMap[event.scope] || event.scope}
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
