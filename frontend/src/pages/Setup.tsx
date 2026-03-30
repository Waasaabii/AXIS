import { useState, type ReactNode } from "react"
import { Link, useNavigate } from "react-router-dom"
import useSWR from "swr"
import { AlertTriangle, CheckCircle2, ChevronRight, KeyRound, Link2, Network, Radio } from "lucide-react"
import { toast } from "sonner"
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
      toast.error("两次输入的密码不一致")
      return
    }
    if (password.length < 6) {
      toast.error("密码长度至少为 6 位")
      return
    }

    setIsSavingPassword(true)
    try {
      await api.updatePassword({ password })
      toast.success("管理员密码已更新")
      setPassword("")
      setConfirmPassword("")
      await refreshSetup()
    } catch (error) {
      toast.error(error instanceof ApiError ? error.message : "密码更新失败")
    } finally {
      setIsSavingPassword(false)
    }
  }

  const handleAddSubscription = async () => {
    if (!subscriptionName.trim() || !subscriptionURL.trim()) {
      toast.error("订阅名称和 URL 不能为空")
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
      toast.success("订阅已添加")
      setSubscriptionURL("")
      await refreshSetup()
    } catch (error) {
      toast.error(error instanceof ApiError ? error.message : "添加订阅失败")
    } finally {
      setIsSavingSubscription(false)
    }
  }

  if (!setupState) {
    return <div className="flex min-h-[50vh] items-center justify-center text-sm text-zinc-500">正在检查初始化状态...</div>
  }

  if (!setupState.required) {
    return (
      <div className="mx-auto flex min-h-[50vh] max-w-3xl items-center">
        <Card className="w-full border-emerald-200 bg-emerald-50/70">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-emerald-700">
              <CheckCircle2 className="h-5 w-5" />
              初始化已完成
            </CardTitle>
            <CardDescription>核心配置已齐备，可以直接进入控制台继续操作。</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => navigate("/dashboard")}>进入总览</Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">首次初始化</h1>
        <p className="max-w-3xl text-sm leading-6 text-zinc-500">
          还差最后几步就能开始使用。先完成下面这些设置，再回到控制台即可。
        </p>
      </div>

      <Card className="border-amber-200 bg-amber-50/70">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-amber-700">
            <AlertTriangle className="h-5 w-5" />
            还需要完成这些设置
          </CardTitle>
          <CardDescription>{setupState.reasons.join(" ")}</CardDescription>
        </CardHeader>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        {setupState.checks.map((check) => (
          <SetupCheckCard key={check.key} check={check} />
        ))}
      </div>

      {setupState.needsPasswordReset && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyRound className="h-5 w-5 text-zinc-500" />
              设置管理员密码
            </CardTitle>
            <CardDescription>先把控制台密码设好，避免任何人都能直接登录。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="setup-password">新密码</Label>
                <Input id="setup-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-password-confirm">确认密码</Label>
                <Input id="setup-password-confirm" type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
              </div>
            </div>
            <Button onClick={handleSavePassword} disabled={isSavingPassword}>
              {isSavingPassword ? "保存中..." : "保存管理员密码"}
            </Button>
          </CardContent>
        </Card>
      )}

      {!setupState.hasRealSubscriptions && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Link2 className="h-5 w-5 text-zinc-500" />
              添加第一个可用订阅
            </CardTitle>
            <CardDescription>把你的机场或服务商提供的订阅链接填进来，替换当前示例内容。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-[220px_1fr]">
              <div className="space-y-2">
                <Label htmlFor="setup-subscription-name">订阅名称</Label>
                <Input id="setup-subscription-name" value={subscriptionName} onChange={(event) => setSubscriptionName(event.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="setup-subscription-url">订阅 URL</Label>
                <Input id="setup-subscription-url" value={subscriptionURL} onChange={(event) => setSubscriptionURL(event.target.value)} placeholder="https://..." />
              </div>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button onClick={handleAddSubscription} disabled={isSavingSubscription}>
                {isSavingSubscription ? "添加中..." : "添加订阅"}
              </Button>
              <Button asChild variant="outline">
                <Link to="/subscriptions">去订阅管理页</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>下一步</CardTitle>
          <CardDescription>如果你想分步骤完成，也可以直接进入下面的页面继续设置。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <QuickLinkCard title="订阅管理" description="添加你的订阅链接，导入可用节点。" icon={<Link2 className="h-5 w-5 text-zinc-500" />} to="/subscriptions" />
          <QuickLinkCard title="出口管理" description="把节点整理成常用分组，比如香港、日本或自动选择。" icon={<Network className="h-5 w-5 text-zinc-500" />} to="/interfaces" />
          <QuickLinkCard title="入口管理" description="创建本机可用的代理入口，供浏览器或其他设备连接。" icon={<Radio className="h-5 w-5 text-zinc-500" />} to="/listeners" />
        </CardContent>
      </Card>
    </div>
  )
}

function SetupCheckCard({ check }: { check: SetupStateResponse["checks"][number] }) {
  const target = actionLinks[check.key as keyof typeof actionLinks]

  return (
    <Card className={check.ready ? "border-emerald-200 bg-emerald-50/50" : "border-zinc-200"}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          {check.ready ? <CheckCircle2 className="h-5 w-5 text-emerald-600" /> : <AlertTriangle className="h-5 w-5 text-amber-500" />}
          {check.title}
        </CardTitle>
        <CardDescription>{check.summary}</CardDescription>
      </CardHeader>
      {!check.ready && check.action && target && (
        <CardContent>
          <Button asChild variant="outline" size="sm">
            <Link to={target}>
              {check.action}
              <ChevronRight className="ml-1 h-4 w-4" />
            </Link>
          </Button>
        </CardContent>
      )}
    </Card>
  )
}

function QuickLinkCard({ title, description, icon, to }: { title: string; description: string; icon: ReactNode; to: string }) {
  return (
    <Link to={to} className="rounded-xl border bg-white p-4 transition-colors hover:border-zinc-300 hover:bg-zinc-50">
      <div className="mb-3 flex items-center justify-between">
        {icon}
        <ChevronRight className="h-4 w-4 text-zinc-400" />
      </div>
      <div className="space-y-1">
        <div className="font-medium text-zinc-900">{title}</div>
        <p className="text-sm leading-6 text-zinc-500">{description}</p>
      </div>
    </Link>
  )
}
