import { useEffect, useState, type ReactNode } from "react"
import { Link, useNavigate } from "react-router-dom"
import useSWR, { useSWRConfig } from "swr"
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
import { apiKeys } from "@/services/api-keys"
import { validatePasswordPair } from "@/lib/password"
import { mutateKeys } from "@/lib/swr"
import { toastApiError } from "@/lib/toast-api-error"

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
  listeners: "本地代理",
}

function sanitizeCheckSummary(summary: string) {
  return summary
    .replace(/订阅源/g, "订阅")
    .replace(/出口组/g, "出口线路")
    .replace(/代理分组/g, "出口线路")
    .replace(/入口监听/g, "本地代理")
    .replace(/代理入口/g, "本地代理")
}

export default function Setup() {
  const navigate = useNavigate()
  const { mutate } = useSWRConfig()
  const { data: setupState, error: setupError, mutate: mutateSetup } = useSWR(apiKeys.setupState, api.getSetupState)
  const { data: session } = useSWR(apiKeys.session, api.getSession)
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [subscriptionName, setSubscriptionName] = useState("airport-main")
  const [subscriptionURL, setSubscriptionURL] = useState("")
  const [isSavingPassword, setIsSavingPassword] = useState(false)
  const [isSavingSubscription, setIsSavingSubscription] = useState(false)

  const refreshSetup = async () => {
    await mutateSetup()
  }

  useEffect(() => {
    if (!setupState?.adminUsername) {
      return
    }
    setUsername((current) => current || setupState.adminUsername)
  }, [setupState?.adminUsername])

  const handleSavePassword = async () => {
    const passwordError = validatePasswordPair(password, confirmPassword)
    if (passwordError) {
      toast.error(passwordError)
      return
    }

    setIsSavingPassword(true)
    try {
      const trimmedUsername = username.trim()
      if (setupState?.needsPasswordReset) {
        await api.bootstrapAdmin({ username: trimmedUsername, password })
        await api.login({ username: trimmedUsername, password })
        await mutateKeys(mutate, [apiKeys.session, apiKeys.bootstrapStatus, apiKeys.setupState])
        toast.success("管理员账号已创建，已自动登录。")
        navigate("/dashboard", { replace: true })
        return
      } else {
        await api.updatePassword({ password })
        toast.success("管理员密码已更新。")
      }
      setUsername((current) => current.trim())
      setPassword("")
      setConfirmPassword("")
      await refreshSetup()
    } catch (error) {
      toastApiError(error, setupState?.needsPasswordReset ? "管理员账号保存失败，请稍后再试。" : "管理员密码保存失败，请稍后再试。")
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
      toastApiError(error, "添加订阅失败，请检查链接后重试。")
    } finally {
      setIsSavingSubscription(false)
    }
  }

  if (!setupState) {
    if (setupError) {
      return (
        <div className="mx-auto flex min-h-[50vh] max-w-xl flex-col items-center justify-center gap-4 text-center">
          <div className="text-sm text-red-500">{setupError instanceof ApiError ? setupError.message : "初始化状态读取失败"}</div>
          <Button variant="outline" onClick={() => mutateSetup()}>
            重新检查
          </Button>
        </div>
      )
    }
    return <div className="flex min-h-[50vh] items-center justify-center text-sm text-zinc-500">正在检查初始化状态...</div>
  }

  if (!setupState.required) {
    return (
      <EmptyStateCard
        className="mx-auto max-w-3xl"
        icon={<CheckCircle2 className="h-5 w-5" />}
        title="首次初始化已经完成"
        description="密码、订阅和本地代理都已准备好。现在可以直接进入总览继续使用。"
        action={
          <Button onClick={() => navigate(session?.authenticated ? "/dashboard" : "/login")}>
            {session?.authenticated ? "进入总览" : "去登录"}
          </Button>
        }
      />
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Getting Started"
        title="先完成这几步，再开始使用"
        description="把密码、订阅和本地代理配好，系统就能进入可用状态。"
      />

      <NoticeCard
        tone="warning"
        icon={<Sparkles className="h-4 w-4" />}
        title="按提示补齐缺口即可"
        description="缺什么就补什么。每张卡片都会告诉你下一步该点哪里。"
      />

      <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
        <Card className="border-zinc-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">当前还没完成的准备项</CardTitle>
            <CardDescription>{setupState.reasons.join(" ")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {setupState.checks.map((check) => (
              <SetupCheckCard key={check.key} check={check} passwordTarget={session?.authenticated ? "/system" : "/setup"} />
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
              title="先设置管理员密码"
              description="先把默认密码换掉，避免被直接登录。"
            />
            <QuickStep
              index={2}
              title="再导入订阅"
              description="导入后，系统会开始拉取节点。"
            />
            <QuickStep
              index={3}
              title="最后配置本地代理"
              description="生成设备要填的代理地址。"
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
            <CardDescription>先把管理员账号和密码设置好，后面再导入订阅和配置本地代理。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              {setupState.needsPasswordReset ? (
                <FieldBlock
                  label="管理员账号"
                  hint="首次初始化时，这就是你后续登录要使用的账号名。"
                  htmlFor="setup-username"
                  input={
                    <Input id="setup-username" value={username} onChange={(event) => setUsername(event.target.value)} placeholder="admin" />
                  }
                />
              ) : null}
              <FieldBlock
                label={setupState.needsPasswordReset ? "管理员密码" : "新密码"}
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
              {isSavingPassword ? "保存中..." : setupState.needsPasswordReset ? "创建管理员账号" : "保存管理员密码"}
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
          <CardDescription>填入订阅链接后，系统会开始拉取节点。</CardDescription>
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
              <Button onClick={handleAddSubscription} disabled={isSavingSubscription || !session?.authenticated}>
                {isSavingSubscription ? "添加中..." : "添加订阅"}
              </Button>
              <Button asChild variant="outline" disabled={!session?.authenticated}>
                <Link to="/subscriptions">去订阅与节点页</Link>
              </Button>
            </div>
            {!session?.authenticated ? (
              <p className="text-sm leading-6 text-zinc-500">先创建管理员账号并登录，后面才能继续。</p>
            ) : null}
          </CardContent>
        </Card>
      )}

      <Card className="border-zinc-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-base">也可以分开做</CardTitle>
          <CardDescription>直接去下面页面分别设置即可。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <QuickLinkCard title="订阅与节点" description="导入你的订阅链接，让可用节点进入系统。" icon={<Link2 className="h-5 w-5 text-zinc-500" />} to="/subscriptions" disabled={!session?.authenticated} />
          <QuickLinkCard title="出口线路" description="把节点整理成常用线路，例如香港、日本或自动选择。" icon={<Network className="h-5 w-5 text-zinc-500" />} to="/interfaces" disabled={!session?.authenticated} />
          <QuickLinkCard title="本地代理" description="生成浏览器和设备要填写的代理地址。" icon={<Radio className="h-5 w-5 text-zinc-500" />} to="/listeners" disabled={!session?.authenticated} />
        </CardContent>
      </Card>
    </div>
  )
}

function SetupCheckCard({ check, passwordTarget }: { check: SetupStateResponse["checks"][number]; passwordTarget: string }) {
  const target = check.key === "password" ? passwordTarget : actionLinks[check.key as keyof typeof actionLinks]

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

function QuickLinkCard({ title, description, icon, to, disabled = false }: { title: string; description: string; icon: ReactNode; to: string; disabled?: boolean }) {
  return (
    <Link
      to={disabled ? "/setup" : to}
      className={`group rounded-2xl border border-zinc-200 bg-white px-4 py-4 transition-colors ${disabled ? "pointer-events-none opacity-50" : "hover:border-zinc-300 hover:bg-zinc-50"}`}
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
