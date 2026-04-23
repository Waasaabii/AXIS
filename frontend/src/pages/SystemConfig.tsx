import { Suspense, lazy, useState, useEffect } from 'react'
import useSWR from 'swr'
import { api } from '@/services/api'
import { apiKeys } from '@/services/api-keys'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { NoticeCard } from '@/components/NoticeCard'
import { PageHeader } from '@/components/PageHeader'
import { Save, RefreshCw, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { validatePasswordPair } from '@/lib/password'
import { mutateMany } from '@/lib/swr'
import { toastApiError } from '@/lib/toast-api-error'

const Editor = lazy(() => import('@monaco-editor/react'))

function EditorFallback({ label }: { label: string }) {
  return (
    <div className="flex h-full min-h-[420px] items-center justify-center bg-zinc-50 text-sm text-zinc-500">
      {label}编辑器正在加载...
    </div>
  )
}

export default function SystemConfig() {
  const { data: configData, mutate: mutateConfig } = useSWR(apiKeys.config, api.getConfig)
  const { data: renderedData, mutate: mutateRendered } = useSWR(apiKeys.renderedConfig, api.getRenderedConfig)
  const { data: versionsData, mutate: mutateVersions } = useSWR(apiKeys.mihomoVersions, api.getMihomoVersions)
  const { data: hostData, mutate: mutateHost } = useSWR(apiKeys.hostStatus, api.getHostStatus, {
    refreshInterval: 5000,
  })
  const { data: updaterData, mutate: mutateUpdater } = useSWR(apiKeys.updaterStatus, api.getUpdaterStatus, {
    refreshInterval: 5000,
  })
  
  const [editorContent, setEditorContent] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [isChangingPassword, setIsChangingPassword] = useState(false)
  const [versionActionKey, setVersionActionKey] = useState("")
  const [isUpdatingAutostart, setIsUpdatingAutostart] = useState(false)
  const [isOpeningHost, setIsOpeningHost] = useState(false)
  const [isCheckingUpdates, setIsCheckingUpdates] = useState(false)

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
      void mutateMany(mutateConfig, mutateRendered) // 保存通常会触发后端重新渲染
    } catch (err) {
      toastApiError(err, '保存失败')
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
      void mutateMany(mutateVersions, mutateConfig, mutateRendered)
    } catch (err) {
      toastApiError(err, "操作失败")
    } finally {
      setVersionActionKey("")
    }
  }

  const handleHostAutostartChange = async (checked: boolean) => {
    setIsUpdatingAutostart(true)
    try {
      await api.setHostAutostart(checked)
      toast.success(checked ? '已启用开机自启' : '已关闭开机自启')
      mutateHost()
    } catch (err) {
      toastApiError(err, '更新失败')
    } finally {
      setIsUpdatingAutostart(false)
    }
  }

  const handleOpenHostControlCenter = async () => {
    setIsOpeningHost(true)
    try {
      await api.openHostControlCenter()
      toast.success(hostData?.desktopMode ? '桌面控制台已拉起' : '已触发宿主动作')
      mutateHost()
    } catch (err) {
      toastApiError(err, '打开失败')
    } finally {
      setIsOpeningHost(false)
    }
  }

  const handleCheckForUpdates = async () => {
    setIsCheckingUpdates(true)
    try {
      await api.checkForUpdates()
      toast.success('已触发桌面更新检查')
      mutateUpdater()
    } catch (err) {
      toastApiError(err, '检查失败')
    } finally {
      setIsCheckingUpdates(false)
    }
  }

  const handleChangePassword = async () => {
    const passwordError = validatePasswordPair(newPassword, confirmPassword)
    if (passwordError) {
      toast.error(passwordError)
      return
    }

    setIsChangingPassword(true)
    try {
      await api.updatePassword({ password: newPassword })
      toast.success("密码修改成功")
      setNewPassword("")
      setConfirmPassword("")
    } catch (err) {
      toastApiError(err, "修改失败")
    } finally {
      setIsChangingPassword(false)
    }
  }

  const installedByVersion = Object.fromEntries((versionsData?.installedVersions || []).map((item) => [item.version, item]))
  const hostLogs = hostData?.logs || []

  return (
    <div className="space-y-6 h-[calc(100vh-8rem)] flex flex-col">
      <PageHeader
        eyebrow="System"
        title="系统与核心"
        description="管理员密码、高级配置、代理核心版本都在这里。需要时再来。"
      />

      <NoticeCard
        icon={<RefreshCw className="h-4 w-4" />}
        title="你可能会用到这里"
        description="改管理员密码、编辑高级配置、切换核心版本时再来。日常操作尽量在订阅、线路和本地代理页完成。"
      />

      <Tabs defaultValue="editor" className="flex-1 flex flex-col min-h-0">
        <TabsList className="grid w-[760px] grid-cols-5">
          <TabsTrigger value="editor">高级配置</TabsTrigger>
          <TabsTrigger value="rendered">当前生成结果</TabsTrigger>
          <TabsTrigger value="password">管理员密码</TabsTrigger>
          <TabsTrigger value="host">宿主集成</TabsTrigger>
          <TabsTrigger value="mihomo">Mihomo 版本</TabsTrigger>
        </TabsList>
        
        <TabsContent value="editor" className="flex-1 flex flex-col mt-4 min-h-0">
          <Card className="flex-1 flex flex-col min-h-0 overflow-hidden border-zinc-200">
            <CardHeader className="py-3 px-4 border-b flex flex-row items-center justify-between bg-zinc-50/50 shrink-0">
              <div>
                <CardTitle className="text-sm">高级配置内容</CardTitle>
                <CardDescription className="text-xs">适合已经明确知道自己要改什么的时候再操作。当前文件位置：{configData?.path || '加载中...'}</CardDescription>
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
                <CardTitle className="text-sm">当前生成给 Mihomo 的配置</CardTitle>
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
              <Button onClick={handleChangePassword} disabled={isChangingPassword || !newPassword || !confirmPassword}>
                {isChangingPassword ? "提交中..." : "保存新密码"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="host" className="flex-1 flex flex-col mt-4 min-h-0">
          <div className="grid gap-4 lg:grid-cols-[1.15fr_1fr]">
            <Card className="border-zinc-200">
              <CardHeader className="py-4 border-b bg-zinc-50/50">
                <CardTitle className="text-base">当前宿主状态</CardTitle>
                <CardDescription className="text-xs">
                  同一套前端会根据运行环境自动切到 Web API 或 Wails 原生绑定，这里显示当前落在哪一侧。
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-6 space-y-4 text-sm">
                <div className="grid gap-3 md:grid-cols-2">
                  <div className="rounded-lg bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">运行形态</div>
                    <div className="font-medium">{hostData?.mode === 'desktop' ? '桌面版' : hostData?.mode === 'web' ? 'Web 服务' : hostData?.mode || '加载中...'}</div>
                  </div>
                  <div className="rounded-lg bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">开机自启</div>
                    <div className="font-medium">
                      {hostData?.autostartManaged ? (hostData?.autostartEnabled ? '已启用' : '未启用') : '当前环境不支持'}
                    </div>
                  </div>
                  <div className="rounded-lg bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">配置文件</div>
                    <div className="font-mono break-all">{hostData?.configPath || '加载中...'}</div>
                  </div>
                  <div className="rounded-lg bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">运行目录</div>
                    <div className="font-mono break-all">{hostData?.runtimeDir || '加载中...'}</div>
                  </div>
                </div>

                <div className="rounded-lg border border-zinc-200 p-4 space-y-3">
                  <div className="flex items-center justify-between gap-4">
                    <div className="space-y-1">
                      <div className="font-medium">宿主动作</div>
                      <p className="text-xs text-zinc-500">
                        桌面版会拉起窗口；Web 服务不会提供此能力。
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!hostData?.desktopMode || isOpeningHost}
                      onClick={handleOpenHostControlCenter}
                    >
                      <ExternalLink className="h-4 w-4 mr-2" />
                      {isOpeningHost ? '处理中...' : '打开桌面控制台'}
                    </Button>
                  </div>

                  <div className="rounded-md bg-zinc-50 p-3">
                    <div className="text-zinc-500 text-xs mb-1">HTTP 监听地址</div>
                    <div className="font-mono break-all">{hostData?.listenAddress || '未提供'}</div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <div className="space-y-4">
              <Card className="border-zinc-200">
                <CardHeader className="py-4 border-b bg-zinc-50/50">
                  <CardTitle className="text-base">开机自启</CardTitle>
                  <CardDescription className="text-xs">
                    只有桌面版支持开机自启。Web 服务不提供这个开关。
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-6">
                  <div className="flex items-center justify-between gap-4 rounded-lg border border-zinc-200 p-4">
                    <div className="space-y-1">
                      <div className="font-medium">启用开机自启</div>
                      <p className="text-xs text-zinc-500">
                        {hostData?.autostartManaged ? '由桌面版管理系统启动项。' : '当前环境不支持。'}
                      </p>
                    </div>
                    <Switch
                      checked={Boolean(hostData?.autostartEnabled)}
                      disabled={!hostData?.autostartManaged || isUpdatingAutostart}
                      onCheckedChange={handleHostAutostartChange}
                    />
                  </div>
                </CardContent>
              </Card>

              <Card className="border-zinc-200">
                <CardHeader className="py-4 border-b bg-zinc-50/50">
                  <CardTitle className="text-base">桌面更新服务</CardTitle>
                  <CardDescription className="text-xs">
                    这里展示独立 updater 守护进程的状态，并允许手动触发一次 GitHub Releases 检查。
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4 pt-6 text-sm">
                  <div className="grid gap-3">
                    <div className="rounded-lg bg-zinc-50 p-3">
                      <div className="mb-1 text-xs text-zinc-500">当前状态</div>
                      <div className="font-medium">{updaterData?.state || '加载中...'}</div>
                    </div>
                    <div className="rounded-lg bg-zinc-50 p-3">
                      <div className="mb-1 text-xs text-zinc-500">版本信息</div>
                      <div className="font-medium">
                        {updaterData?.latestVersion
                          ? `${updaterData.currentVersion || 'unknown'} -> ${updaterData.latestVersion}`
                          : updaterData?.currentVersion || '加载中...'}
                      </div>
                    </div>
                    <div className="rounded-lg bg-zinc-50 p-3">
                      <div className="mb-1 text-xs text-zinc-500">说明</div>
                      <div>{updaterData?.message || '当前没有更新服务说明。'}</div>
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    <Button variant="outline" size="sm" disabled={!hostData?.desktopMode || isCheckingUpdates} onClick={handleCheckForUpdates}>
                      <RefreshCw className="mr-2 h-4 w-4" />
                      {isCheckingUpdates ? '检查中...' : '检查更新'}
                    </Button>
                    {updaterData?.releaseUrl ? (
                      <Button asChild variant="ghost" size="sm">
                        <a href={updaterData.releaseUrl} target="_blank" rel="noreferrer">查看发布页</a>
                      </Button>
                    ) : null}
                  </div>
                </CardContent>
              </Card>

              <Card className="border-zinc-200">
                <CardHeader className="py-4 border-b bg-zinc-50/50">
                  <CardTitle className="text-base">宿主日志</CardTitle>
                  <CardDescription className="text-xs">
                    这里只放桌面壳相关动作，便于排查开机自启、二次启动唤醒和窗口拉起。
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-6">
                  {!hostLogs.length ? (
                    <p className="text-sm text-zinc-500">当前还没有宿主事件。</p>
                  ) : (
                    <div className="space-y-2">
                      {hostLogs.map((entry: string, index: number) => (
                        <div key={`${entry}-${index}`} className="rounded-lg bg-zinc-50 px-3 py-2 text-sm">
                          {entry}
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="mihomo" className="flex-1 flex flex-col mt-4 min-h-0">
          <div className="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
            <Card className="border-zinc-200">
              <CardHeader className="py-4 border-b bg-zinc-50/50">
                <CardTitle className="text-base">可用版本列表</CardTitle>
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
                  <div className="rounded-lg bg-zinc-50 p-3 font-mono break-all">{apiKeys.openapiJson}</div>
                  <Button asChild variant="outline" size="sm">
                    <a href={apiKeys.openapiJson} target="_blank" rel="noreferrer">打开接口说明</a>
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
