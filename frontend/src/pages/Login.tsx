import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { api, ApiError } from "@/services/api"
import { toast } from "sonner"
import { useNavigate } from "react-router-dom"

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

      toast.success("登录成功")
      navigate(data.setupRequired ? "/setup" : "/dashboard", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "登录失败")
    } finally {
      setIsLoading(false)
    }
  }

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      toast.error("两次输入的密码不一致")
      return
    }
    if (newPassword.length < 6) {
      toast.error("密码长度至少为6个字符")
      return
    }

    setIsLoading(true)
    try {
      await api.updatePassword({ password: newPassword })

      toast.success("密码修改成功")
      navigate("/setup", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "修改失败")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50">
      <Card className="w-full max-w-sm">
        <CardHeader className="space-y-1 text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">
            AXIS
          </CardTitle>
          <CardDescription>
            {requiresReset ? "检测到默认凭据，请先修改密码" : "输入管理员账号密码以访问控制台"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!requiresReset ? (
            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setUsername(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading ? "登录中..." : "登录"}
              </Button>
            </form>
          ) : (
            <form onSubmit={handleResetPassword} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-password">新密码</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewPassword(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">确认新密码</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setConfirmPassword(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading ? "更新中..." : "确认修改进入系统"}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
