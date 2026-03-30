import { useState, type ReactNode } from "react"
import { Link, useNavigate } from "react-router-dom"
import useSWR from "swr"
import { AlertTriangle, CheckCircle2, ChevronRight, KeyRound, Link2, Network, Radio, Sparkles } from "lucide-react"
import { toast } from "sonner"

import { EmptyStateCard } from "@/components/EmptyStateCard"
import { NoticeCard } from "@/components/NoticeCard"
import { PageHeader } from "@/components/PageHeader"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api, ApiError, type SetupStateResponse } from "@/services/api"

const actionLinks = {
  password: "/system",
  subscriptions: "/subscriptions",
  "egress-groups": "/interfaces",
  listeners: "/listeners",
} as const

const checkTitleMap: Partial<Record<SetupStateResponse["checks"][number]["key"], string>> = {
  password: "管理员密码",
  subscriptions: "订阅与节点",
  "egress-groups": "出口线路",
  listeners: "本地入口",
}

function sanitizeCheckSummary(summary: string) {
  return summary
    .replace(/订阅源/g, "订阅")
    .replace(/出口组/g, "出口线路")
    .replace(/代理分组/g, "出口线路")
    .replace(/入口监听/g, "本地入口")
    .replace(/代理入口/g, "本地入口")
}

export default function Setup() {
  const navigate = useNavigate()
  const { data: setupState, mutate: mutateSetup } = useSWR("/api/setup-state", api.getSetupState)
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [subscriptionName, setSubscriptionName] = useState("airport-main")
  const [subscriptionURL, setSubscriptionURL] = useState("")
  const [isSavingPassword, setIsSavingPassword] = useState(false)
  const [isSavingSubscription, setIsSavingSubscription] = useState(false)

  const refreshSetup = async () => {
    await mutateSetup()
  }

  const handleSavePassword = async () => {
    if (password !== confirmPassword) {
      toast.error("两次输入的密码不一致，请重新确认。")
      return
    }
    if (password.length < 6) {
      toast.error("密码至少需要 6 位。")
      return
    }

    setIsSavingPassword(true)
    try {
      await api.updatePassword({ password })
      toast.success("管理员密码已更新。")
      setPassword("")
      setConfirmPassword("")
      await refreshSetup()
    } catch (error) {
      toast.error(error instanceof ApiError ? error.message : "密码更新失败，请稍后再试。")
    } finally {
      setIsSavingPassword(false)
    }
  }

  const handleAddSubscription = async () => {
    if (!subscriptionName.trim() || !subscriptionURL.trim()) {
      toast.error("订阅名称和链接都需要填写。")
      return
    }

    setIsSavingSubscription(true)
    try {
      await api.addSubscription({
        name: subscriptionName.trim(),
        url: subscriptionURL.trim(),
        type: "mihomo-http",
        interval: 3600,
      })
      toast.success("订阅已经添加，系统会开始拉取节点。")
      setSubscriptionURL("")
      await refreshSetup()
    } catch (error) {
      toast.error(error instanceof ApiError ? error.message : "添加订阅失败，请检查链接后重试。")
    } finally {
      setIsSavingSubscription(false)
    }
  }

  if (!setupState) {
    return <div className="flex min-h-[50vh] items-center justify-center text-sm text-zinc-500">正在检查初始化状态...</div>
  }

  if (!setupState.required) {
    return (
      <EmptyStateCard
        className="mx-auto max-w-3xl"
        icon={<CheckCircle2 className="h-5 w-5" />}
        title="首次初始化已经完成"
        description="密码、订阅和基础入口都已经准备好。你可以直接进入总览继续使用，或者回到各页面做更细的整理。"
        action={
          <Button onClick={() => navigate("/dashboard")}>
            进入总览
          </Button>
        }
      />
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Getting Started"
        title="先完成这几步，再开始稳定使用"
        description="这里不会让你读一堆配置名词。你只需要完成密码、订阅和入口这几项最小准备，系统就能正常进入工作状态。"
      />

      <NoticeCard
        tone="warning"
        icon={<Sparkles className="h-4 w-4" />}
        title="初始化的目标只有一个"
        description="把系统从“能打开页面”变成“真的可以连上用”。下面每张卡片都只告诉你缺什么、为什么缺、点哪里继续。"
      />

      <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">当前还没完成的准备项</CardTitle>
            <CardDescription>{setupState.reasons.join(" ")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {setupState.checks.map((check) => (
              <SetupCheckCard key={check.key} check={check} />
            ))}
          </CardContent>
        </Card>

        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">推荐顺序</CardTitle>
            <CardDescription>按这个顺序处理，最不容易来回跳页面。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <QuickStep
              index={1}
              title="先改管理员密码"
              description="如果还在用默认密码，先替换掉，避免控制台被直接登录。"
            />
            <QuickStep
              index={2}
              title="再导入你的订阅"
              description="有了真实订阅，系统才会出现你自己的可用节点。"
            />
            <QuickStep
              index={3}
              title="最后创建本地入口"
              description="让浏览器、系统或其他设备真正能连到 AXIS。"
            />
          </CardContent>
        </Card>
      </div>

      {setupState.needsPasswordReset && (
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <KeyRound className="h-5 w-5 text-zinc-500" />
              先把管理员密码换掉
            </CardTitle>
            <CardDescription>这是最先该完成的一步。设置好之后，后面再继续导入订阅和入口配置。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <FieldBlock
                label="新密码"
                hint="建议至少 6 位。"
                htmlFor="setup-password"
                input={
                  <Input id="setup-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
                }
              />
              <FieldBlock
                label="确认密码"
                hint="再输入一次，避免手误。"
                htmlFor="setup-password-confirm"
                input={
                  <Input id="setup-password-confirm" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
                }
              />
            </div>
            <Button onClick={handleSavePassword} disabled={isSavingPassword}>
              {isSavingPassword ? "保存中..." : "保存管理员密码"}
            </Button>
          </CardContent>
        </Card>
      )}

      {!setupState.hasRealSubscriptions && (
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Link2 className="h-5 w-5 text-zinc-500" />
              添加第一个可用订阅
            </CardTitle>
            <CardDescription>把服务商给你的订阅链接填进来，替换当前示例内容。完成这一步后，AXIS 才会有真实可用的节点。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-[220px_1fr]">
              <FieldBlock
                label="订阅名称"
                hint="这是给你自己看的名字，后面创建线路时会用到。"
                htmlFor="setup-subscription-name"
                input={
                  <Input id="setup-subscription-name" value={subscriptionName} onChange={(event) => setSubscriptionName(event.target.value)} />
                }
              />
              <FieldBlock
                label="订阅链接"
                hint="通常由服务商提供，形如 https://..."
                htmlFor="setup-subscription-url"
                input={
                  <Input id="setup-subscription-url" value={subscriptionURL} onChange={(event) => setSubscriptionURL(event.target.value)} placeholder="https://..." />
                }
              />
            </div>
            <div className="flex flex-wrap gap-3">
              <Button onClick={handleAddSubscription} disabled={isSavingSubscription}>
                {isSavingSubscription ? "添加中..." : "添加订阅"}
              </Button>
              <Button asChild variant="outline">
                <Link to="/subscriptions">去订阅与节点页</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card className="border-zinc-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-base">如果你想分步骤完成</CardTitle>
          <CardDescription>也可以直接进入下面的页面逐项设置。每个页面都保留了更详细的说明。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <QuickLinkCard title="订阅与节点" description="导入你的订阅链接，让可用节点进入系统。" icon={<Link2 className="h-5 w-5 text-zinc-500" />} to="/subscriptions" />
          <QuickLinkCard title="出口线路" description="把节点整理成常用线路，例如香港、日本或自动选择。" icon={<Network className="h-5 w-5 text-zinc-500" />} to="/interfaces" />
          <QuickLinkCard title="本地入口" description="创建浏览器和设备真正要连接的本地代理入口。" icon={<Radio className="h-5 w-5 text-zinc-500" />} to="/listeners" />
        </CardContent>
      </Card>
    </div>
  )
}

function SetupCheckCard({ check }: { check: SetupStateResponse["checks"][number] }) {
  const target = actionLinks[check.key as keyof typeof actionLinks]

  return (
    <div className={`rounded-2xl border px-4 py-4 ${check.ready ? "border-emerald-200 bg-emerald-50/70" : "border-zinc-200 bg-white"}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-sm font-medium text-zinc-950">
            {check.ready ? <CheckCircle2 className="h-4 w-4 text-emerald-600" /> : <AlertTriangle className="h-4 w-4 text-amber-500" />}
            {checkTitleMap[check.key] || check.title}
          </div>
          <p className="text-sm leading-6 text-zinc-600">{sanitizeCheckSummary(check.summary)}</p>
        </div>
        {!check.ready && check.action && target ? (
          <Button asChild variant="outline" size="sm" className="shrink-0">
            <Link to={target}>
              {check.action}
              <ChevronRight className="ml-1 h-4 w-4" />
            </Link>
          </Button>
        ) : null}
      </div>
    </div>
  )
}

function QuickStep({ index, title, description }: { index: number; title: string; description: string }) {
  return (
    <div className="rounded-2xl border border-zinc-200 bg-zinc-50/70 px-4 py-4">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-zinc-950">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white text-xs text-zinc-700 shadow-sm">{index}</span>
        {title}
      </div>
      <p className="text-sm leading-6 text-zinc-600">{description}</p>
    </div>
  )
}

function QuickLinkCard({ title, description, icon, to }: { title: string; description: string; icon: ReactNode; to: string }) {
  return (
    <Link
      to={to}
      className="group rounded-2xl border border-zinc-200 bg-white px-4 py-4 transition-colors hover:border-zinc-300 hover:bg-zinc-50"
    >
      <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-zinc-50">
        {icon}
      </div>
      <div className="space-y-1">
        <div className="flex items-center gap-2 text-sm font-medium text-zinc-950">
          {title}
          <ChevronRight className="h-4 w-4 text-zinc-400 transition-transform group-hover:translate-x-0.5" />
        </div>
        <p className="text-sm leading-6 text-zinc-600">{description}</p>
      </div>
    </Link>
  )
}

function FieldBlock({
  label,
  hint,
  htmlFor,
  input,
}: {
  label: string
  hint: string
  htmlFor: string
  input: ReactNode
}) {
  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <Label htmlFor={htmlFor}>{label}</Label>
        <p className="text-xs leading-5 text-zinc-500">{hint}</p>
      </div>
      {input}
    </div>
  )
}
