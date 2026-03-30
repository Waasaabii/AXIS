import { Suspense, lazy, useState, useEffect } from 'react'
import useSWR from 'swr'
import { api, ApiError } from '@/services/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Save, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

const Editor = lazy(() => import('@monaco-editor/react'))

function EditorFallback({ label }: { label: string }) {
  return (
    <div className="flex h-full min-h-[420px] items-center justify-center bg-zinc-50 text-sm text-zinc-500">
      {label}编辑器正在加载...
    </div>
  )
}

export default function SystemConfig() {
  const { data: configData, mutate: mutateConfig } = useSWR('/api/config', api.getConfig)
  const { data: renderedData, mutate: mutateRendered } = useSWR('/api/rendered-config', api.getRenderedConfig)
  const { data: versionsData, mutate: mutateVersions } = useSWR('/api/mihomo/versions', api.getMihomoVersions)
  
  const [editorContent, setEditorContent] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [isChangingPassword, setIsChangingPassword] = useState(false)
  const [versionActionKey, setVersionActionKey] = useState("")

  // Sync config when loaded
  useEffect(() => {
    if (configData?.config) {
      setEditorContent(JSON.stringify(configData.config, null, 2))
    }
  }, [configData])

  const handleSave = async () => {
    try {
      setIsSaving(true)
      let parsedConfig
      try {
        parsedConfig = JSON.parse(editorContent)
      } catch {
        toast.error('配置内容格式不正确，请检查括号、引号和逗号后再保存')
        return
      }

      await api.saveConfig({ config: parsedConfig })
      
      toast.success('配置已保存')
      mutateConfig()
      mutateRendered() // Saving often triggers a re-render on backend
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '保存失败')
    } finally {
      setIsSaving(false)
    }
  }

  const handleVersionAction = async (version: string, action: "download" | "install" | "activate") => {
    setVersionActionKey(`${version}:${action}`)
    try {
      const actionMap = {
        download: api.downloadMihomoVersion,
        install: api.installMihomoVersion,
        activate: api.activateMihomoVersion,
      } as const
      const data = await actionMap[action](version)
      if (!data.ok) throw new Error(`${action} 失败`)
      toast.success(`Mihomo ${version} 已${action === "download" ? "下载" : action === "install" ? "安装" : "激活"}`)
      mutateVersions()
      mutateConfig()
      mutateRendered()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "操作失败")
    } finally {
      setVersionActionKey("")
    }
  }

  const installedByVersion = Object.fromEntries((versionsData?.installedVersions || []).map((item) => [item.version, item]))

  return (
    <div className="space-y-6 h-[calc(100vh-8rem)] flex flex-col">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">系统配置</h1>
        <p className="text-zinc-500">管理基础配置、管理员密码，以及当前使用的代理核心版本。</p>
      </div>

      <Tabs defaultValue="editor" className="flex-1 flex flex-col min-h-0">
        <TabsList className="grid w-[620px] grid-cols-4">
          <TabsTrigger value="editor">基础配置</TabsTrigger>
          <TabsTrigger value="rendered">生成结果预览</TabsTrigger>
          <TabsTrigger value="password">管理员密码</TabsTrigger>
          <TabsTrigger value="mihomo">代理核心版本</TabsTrigger>
        </TabsList>
        
        <TabsContent value="editor" className="flex-1 flex flex-col mt-4 min-h-0">
          <Card className="flex-1 flex flex-col min-h-0 overflow-hidden border-zinc-200">
            <CardHeader className="py-3 px-4 border-b flex flex-row items-center justify-between bg-zinc-50/50 shrink-0">
              <div>
                <CardTitle className="text-sm">配置文件（高级）</CardTitle>
                <CardDescription className="text-xs">当前配置文件位置：{configData?.path || '加载中...'}</CardDescription>
              </div>
              <Button size="sm" onClick={handleSave} disabled={isSaving}>
                <Save className="h-4 w-4 mr-2" />
                {isSaving ? '保存中...' : '保存配置'}
              </Button>
            </CardHeader>
            <CardContent className="p-0 flex-1 min-h-0 relative">
              <div className="absolute inset-0">
                <Suspense fallback={<EditorFallback label="配置" />}>
                  <Editor
                    height="100%"
                    defaultLanguage="json"
                    theme="vs-light"
                    value={editorContent}
                    onChange={(val) => setEditorContent(val || '')}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 13,
                      wordWrap: 'on',
                      scrollBeyondLastLine: false,
                      padding: { top: 16, bottom: 16 }
                    }}
                  />
                </Suspense>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="rendered" className="flex-1 flex flex-col mt-4 min-h-0">
          <Card className="flex-1 flex flex-col min-h-0 overflow-hidden border-zinc-200">
            <CardHeader className="py-3 px-4 border-b flex flex-row items-center justify-between bg-zinc-50/50 shrink-0">
              <div>
                <CardTitle className="text-sm">当前生成的代理核心配置</CardTitle>
                <CardDescription className="text-xs">
                  最近生成时间: {renderedData?.updatedAt ? format(new Date(renderedData.updatedAt), 'PP HH:mm:ss', { locale: zhCN }) : '加载中...'}
                </CardDescription>
              </div>
              <Button size="sm" variant="outline" onClick={() => mutateRendered()}>
                <RefreshCw className="h-4 w-4 mr-2" />
                刷新预览
              </Button>
            </CardHeader>
            <CardContent className="p-0 flex-1 min-h-0 relative">
              <div className="absolute inset-0">
                <Suspense fallback={<EditorFallback label="预览" />}>
                  <Editor
                    height="100%"
                    defaultLanguage="yaml"
                    theme="vs-light"
                    value={renderedData?.content || ''}
                    options={{
                      readOnly: true,
                      minimap: { enabled: false },
                      fontSize: 13,
                      wordWrap: 'on',
                      scrollBeyondLastLine: false,
                      padding: { top: 16, bottom: 16 }
                    }}
                  />
                </Suspense>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="password" className="flex-1 flex flex-col mt-4 min-h-0">
          <Card className="max-w-md border-zinc-200">
            <CardHeader className="py-4 border-b bg-zinc-50/50">
              <CardTitle className="text-base">修改管理员密码</CardTitle>
              <CardDescription className="text-xs">
                新密码会加密保存，不会以明文写入配置文件。
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4 pt-6">
              <div className="space-y-2">
                <Label htmlFor="new-password">新密码</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">确认新密码</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                />
              </div>
              <Button 
                onClick={async () => {
                  if (newPassword !== confirmPassword) {
                    toast.error("两次输入的密码不一致")
                    return
                  }
                  if (newPassword.length < 6) {
                    toast.error("密码长度至少为 6 位")
                    return
                  }
                  setIsChangingPassword(true)
                  try {
                    await api.updatePassword({ password: newPassword })
                    toast.success("密码修改成功")
                    setNewPassword("")
                    setConfirmPassword("")
                  } catch (err) {
                    toast.error(err instanceof ApiError ? err.message : "修改失败")
                  } finally {
                    setIsChangingPassword(false)
                  }
                }} 
                disabled={isChangingPassword || !newPassword || !confirmPassword}
              >
                {isChangingPassword ? "提交中..." : "保存新密码"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="mihomo" className="flex-1 flex flex-col mt-4 min-h-0">
          <div className="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
            <Card className="border-zinc-200">
              <CardHeader className="py-4 border-b bg-zinc-50/50">
                <CardTitle className="text-base">支持版本矩阵</CardTitle>
                <CardDescription className="text-xs">
                  当前系统: {versionsData?.platform?.os || "unknown"} / {versionsData?.platform?.arch || "unknown"}，
                  推荐版本: {versionsData?.recommended || "未知"}
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-6 space-y-3">
                {!versionsData?.supportMatrix?.length ? (
                  <p className="text-sm text-zinc-500">加载中...</p>
                ) : (
                  versionsData.supportMatrix.map((item) => {
                    const installed = installedByVersion[item.version]
                    const isActive = versionsData.activeVersion === item.version
                    return (
                      <div key={item.version} className="rounded-xl border bg-white p-4 space-y-3">
                        <div className="flex items-start justify-between gap-4">
                          <div className="space-y-1">
                            <div className="flex items-center gap-2">
                              <span className="font-semibold">{item.version}</span>
                              {item.recommended && <span className="rounded bg-emerald-100 px-2 py-0.5 text-[11px] text-emerald-700">推荐</span>}
                              {isActive && <span className="rounded bg-blue-100 px-2 py-0.5 text-[11px] text-blue-700">当前激活</span>}
                              {!item.supported && <span className="rounded bg-red-100 px-2 py-0.5 text-[11px] text-red-700">不支持</span>}
                            </div>
                            <p className="text-xs text-zinc-500">{item.compatibility}</p>
                            <p className="text-xs text-zinc-400 font-mono break-all">{item.assetName || item.releaseUrl}</p>
                          </div>
                          <div className="text-right text-xs text-zinc-500">
                            <div>当前进度: {installed?.status || "未下载"}</div>
                            {installed?.binaryPath && <div className="mt-1 max-w-[220px] break-all text-zinc-400">{installed.binaryPath}</div>}
                          </div>
                        </div>

                        <div className="flex flex-wrap items-center gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={!item.supported || versionActionKey === `${item.version}:download`}
                            onClick={() => handleVersionAction(item.version, "download")}
                          >
                            {versionActionKey === `${item.version}:download` ? "下载中..." : "下载"}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={!installed?.archivePath || versionActionKey === `${item.version}:install`}
                            onClick={() => handleVersionAction(item.version, "install")}
                          >
                            {versionActionKey === `${item.version}:install` ? "安装中..." : "安装"}
                          </Button>
                          <Button
                            size="sm"
                            disabled={!installed?.binaryPath || isActive || versionActionKey === `${item.version}:activate`}
                            onClick={() => handleVersionAction(item.version, "activate")}
                          >
                            {versionActionKey === `${item.version}:activate` ? "激活中..." : isActive ? "已激活" : "激活"}
                          </Button>
                          <a
                            href={item.releaseUrl}
                            target="_blank"
                            rel="noreferrer"
                            className="text-xs text-zinc-500 underline-offset-4 hover:underline"
                          >
                            查看发布页
                          </a>
                        </div>
                      </div>
                    )
                  })
                )}
              </CardContent>
            </Card>

            <div className="space-y-4">
              <Card className="border-zinc-200">
                <CardHeader className="py-4 border-b bg-zinc-50/50">
                  <CardTitle className="text-base">当前正在使用的版本</CardTitle>
                  <CardDescription className="text-xs">
                    激活后，程序会自动切换到对应的代理核心可执行文件。
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-6 space-y-3 text-sm">
                  <div className="rounded-lg bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">当前程序使用的代理核心路径</div>
                    <div className="font-mono break-all">{versionsData?.configuredBinary || configData?.config?.runtime?.mihomo_binary || "未配置"}</div>
                  </div>
                  <div className="rounded-lg bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">当前启用的版本</div>
                    <div className="font-medium">{versionsData?.activeVersion || "还没有通过这里切换版本"}</div>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-zinc-200">
                <CardHeader className="py-4 border-b bg-zinc-50/50">
                  <CardTitle className="text-base">接口说明（高级）</CardTitle>
                  <CardDescription className="text-xs">
                    如果你需要做二次开发，可以在这里查看当前接口说明。
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-6 space-y-3 text-sm">
                  <div className="rounded-lg bg-zinc-50 p-3 font-mono break-all">/api/openapi.json</div>
                  <Button asChild variant="outline" size="sm">
                    <a href="/api/openapi.json" target="_blank" rel="noreferrer">打开接口说明</a>
                  </Button>
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
