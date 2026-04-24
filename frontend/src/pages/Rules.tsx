import { useState } from 'react'
import useSWR from 'swr'
import { Brain, CheckCircle2, ListChecks, Sparkles, TriangleAlert } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { NoticeCard } from '@/components/NoticeCard'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { toastApiError } from '@/lib/toast-api-error'

export default function Rules() {
  const { data: routes } = useSWR(apiKeys.routes, api.getRoutes)
  const { data: capabilities } = useSWR(apiKeys.coreCapabilities, api.getCoreCapabilities)
  const [goal, setGoal] = useState('ChatGPT、开发社区走最快的出口，普通网页能直连就直连，节点异常时自动换备用出口')
  const [proposal, setProposal] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  const generateProposal = async () => {
    setLoading(true)
    try {
      const result = await api.createLLMProposal({ kind: 'rule', goal, stream: true, context: { routes: (routes || []).map((item) => item.name) } })
      setProposal(result)
    } catch (err) {
      toastApiError(err, '生成规则草案失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader eyebrow="Rules" title="让 AXIS 自动选择怎么访问" description="把一句话变成可测试的规则草案。测试通过并确认后，AXIS 才会使用它。" />
      <NoticeCard tone="neutral" icon={<Brain className="h-4 w-4" />} title="规则不会每次访问都问大模型" description="大模型只负责生成草案，AXIS 会把草案变成固定规则并先测试。" />

      <Card className="border-zinc-200 shadow-sm">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base"><Sparkles className="h-4 w-4" />生成规则草案</CardTitle>
          <CardDescription>描述你希望 AXIS 怎么处理直连、代理和不同出口。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Input value={goal} onChange={(event) => setGoal(event.target.value)} placeholder="例如：开发网站走美国出口，国内网站直连" />
          <Button onClick={generateProposal} disabled={!goal.trim() || loading}>{loading ? '正在生成...' : '生成可测试草案'}</Button>
        </CardContent>
      </Card>

      {proposal ? (
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader><CardTitle className="flex items-center gap-2 text-base"><ListChecks className="h-4 w-4" />草案进度</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {proposal.steps?.map((step: any) => <div key={step.title} className="rounded-2xl border border-zinc-200 px-4 py-3"><div className="flex items-center gap-2 font-medium text-zinc-950"><CheckCircle2 className="h-4 w-4 text-emerald-600" />{step.title}</div><div className="mt-1 text-sm text-zinc-500">{step.message}</div></div>)}
            <pre className="overflow-auto rounded-2xl bg-zinc-950 p-4 text-xs text-zinc-50">{JSON.stringify(proposal.proposal, null, 2)}</pre>
            <NoticeCard tone="warning" icon={<TriangleAlert className="h-4 w-4" />} title="还不能直接采用" description="下一步需要用 AXIS 的连接测试验证，再由你确认后写入配置。" />
          </CardContent>
        </Card>
      ) : null}

      <Card className="border-zinc-200 shadow-sm">
        <CardHeader><CardTitle className="text-base">当前支持层级</CardTitle><CardDescription>协议不是简单支持或不支持，AXIS 会按导入、保真、发布、使用、创建分别判断。</CardDescription></CardHeader>
        <CardContent className="grid gap-3 lg:grid-cols-2">
          {(capabilities?.protocols || []).map((item: any) => <div key={item.protocol} className="rounded-2xl border border-zinc-200 p-4"><div className="font-semibold text-zinc-950">{item.protocol}</div><div className="mt-2 space-y-1 text-sm text-zinc-500"><div>导入：{item.import}</div><div>保真：{item.preserve}</div><div>发布：{item.publish}</div><div>创建：{item.create}</div></div></div>)}
        </CardContent>
      </Card>
    </div>
  )
}
