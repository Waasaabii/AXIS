import { useState } from "react"
import { ShieldCheck, Sparkles } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api, ApiError } from "@/services/api"

export default function Login() {
  const navigate = useNavigate()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [requiresReset, setRequiresReset] = useState(false)
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      const data = await api.login({ username, password })

      if (data.requiresPasswordReset) {
        setRequiresReset(true)
        return
      }

      toast.success("登录成功。")
      navigate(data.setupRequired ? "/setup" : "/dashboard", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "登录失败，请检查账号和密码。")
    } finally {
      setIsLoading(false)
    }
  }

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      toast.error("两次输入的密码不一致。")
      return
    }
    if (newPassword.length < 6) {
      toast.error("密码至少需要 6 位。")
      return
    }

    setIsLoading(true)
    try {
      await api.updatePassword({ password: newPassword })

      toast.success("密码已经更新。")
      navigate("/setup", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "修改失败，请稍后再试。")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[radial-gradient(circle_at_top,_rgba(255,255,255,1),_rgba(244,244,245,0.92)_44%,_rgba(228,228,231,0.72)_100%)] px-4 py-10">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_bottom,rgba(255,255,255,0.12),rgba(255,255,255,0))]" />
      <div className="relative grid w-full max-w-5xl gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <div className="hidden rounded-[2rem] border border-white/70 bg-white/65 p-8 shadow-[0_30px_80px_-40px_rgba(24,24,27,0.35)] backdrop-blur lg:flex lg:flex-col lg:justify-between">
          <div className="space-y-4">
            <div className="text-[11px] font-semibold uppercase tracking-[0.28em] text-zinc-400">AXIS Console</div>
            <div className="space-y-3">
              <h1 className="text-4xl font-semibold tracking-tight text-zinc-950">用更少的步骤，管理你的代理路径。</h1>
              <p className="max-w-md text-sm leading-7 text-zinc-600">
                AXIS 把订阅、线路、本地入口和系统状态整理成更容易理解的流程。登录后，你会先看到当前系统是否可用，以及接下来最该做什么。
              </p>
            </div>
          </div>
          <div className="grid gap-3">
            <FeatureItem
              icon={<ShieldCheck className="h-4 w-4" />}
              title="先看可用性，再看细节"
              description="总览页会先告诉你系统能不能用，而不是一上来就堆满配置名词。"
            />
            <FeatureItem
              icon={<Sparkles className="h-4 w-4" />}
              title="把复杂概念翻译成人话"
              description="订阅、线路、本地入口和运行状态，都尽量用目标导向的表达来呈现。"
            />
          </div>
        </div>

        <Card className="w-full rounded-[2rem] border-white/80 bg-white/88 shadow-[0_30px_80px_-40px_rgba(24,24,27,0.4)] backdrop-blur">
          <CardHeader className="space-y-2 px-8 pt-8 text-left">
            <CardTitle className="text-3xl font-semibold tracking-tight text-zinc-950">
              {requiresReset ? "先更新管理员密码" : "登录 AXIS"}
            </CardTitle>
            <CardDescription className="text-sm leading-6 text-zinc-600">
              {requiresReset
                ? "系统检测到你还在使用默认凭据。先换成自己的密码，再继续进入初始化流程。"
                : "输入管理员账号和密码，进入控制台继续配置或管理。"}
            </CardDescription>
          </CardHeader>
          <CardContent className="px-8 pb-8">
            {!requiresReset ? (
              <form onSubmit={handleLogin} className="space-y-5">
                <FieldBlock
                  label="用户名"
                  input={
                    <Input
                      id="username"
                      type="text"
                      value={username}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => setUsername(e.target.value)}
                      placeholder="输入管理员用户名"
                      required
                    />
                  }
                />
                <FieldBlock
                  label="密码"
                  input={
                    <Input
                      id="password"
                      type="password"
                      value={password}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
                      placeholder="输入管理员密码"
                      required
                    />
                  }
                />
                <Button type="submit" className="w-full" disabled={isLoading}>
                  {isLoading ? "登录中..." : "进入控制台"}
                </Button>
              </form>
            ) : (
              <form onSubmit={handleResetPassword} className="space-y-5">
                <FieldBlock
                  label="新密码"
                  input={
                    <Input
                      id="new-password"
                      type="password"
                      value={newPassword}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewPassword(e.target.value)}
                      placeholder="至少 6 位"
                      required
                    />
                  }
                />
                <FieldBlock
                  label="确认新密码"
                  input={
                    <Input
                      id="confirm-password"
                      type="password"
                      value={confirmPassword}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => setConfirmPassword(e.target.value)}
                      placeholder="再输入一次"
                      required
                    />
                  }
                />
                <Button type="submit" className="w-full" disabled={isLoading}>
                  {isLoading ? "更新中..." : "保存密码并继续"}
                </Button>
              </form>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function FieldBlock({ label, input }: { label: string; input: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <Label className="text-sm font-medium text-zinc-700">{label}</Label>
      {input}
    </div>
  )
}

function FeatureItem({ icon, title, description }: { icon: React.ReactNode; title: string; description: string }) {
  return (
    <div className="rounded-2xl border border-zinc-200/80 bg-white/75 px-4 py-4">
      <div className="mb-2 flex h-9 w-9 items-center justify-center rounded-full border border-zinc-200 bg-zinc-50 text-zinc-600">
        {icon}
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-zinc-950">{title}</div>
        <p className="text-sm leading-6 text-zinc-600">{description}</p>
      </div>
    </div>
  )
}
