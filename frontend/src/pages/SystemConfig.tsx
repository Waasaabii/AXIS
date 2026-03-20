import { useState, useEffect } from 'react'
import useSWR from 'swr'
import { fetcher } from '@/services/api'
import Editor from '@monaco-editor/react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { toast } from 'sonner'
import { Save, RefreshCw } from 'lucide-react'
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'

export default function SystemConfig() {
  const { data: configData, mutate: mutateConfig } = useSWR('/api/config', fetcher)
  const { data: renderedData, mutate: mutateRendered } = useSWR('/api/rendered-config', fetcher)
  
  const [editorContent, setEditorContent] = useState('')
  const [isSaving, setIsSaving] = useState(false)

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
      } catch (err) {
        toast.error('JSON 格式不正确，请检查后再保存')
        return
      }

      const res = await fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config: parsedConfig })
      })
      
      const result = await res.json()
      if (!res.ok) throw new Error(result.error || '保存失败')
      
      toast.success('配置已保存')
      mutateConfig()
      mutateRendered() // Saving often triggers a re-render on backend
    } catch (err: any) {
      toast.error(err.message || '保存失败')
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-6 h-[calc(100vh-8rem)] flex flex-col">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">系统配置</h1>
        <p className="text-zinc-500">编辑基础配置并查看渲染后的 Mihomo 运行态配置。</p>
      </div>

      <Tabs defaultValue="editor" className="flex-1 flex flex-col min-h-0">
        <TabsList className="w-[300px] grid w-full grid-cols-2">
          <TabsTrigger value="editor">控制面配置 (JSON)</TabsTrigger>
          <TabsTrigger value="rendered">渲染产物 (YAML)</TabsTrigger>
        </TabsList>
        
        <TabsContent value="editor" className="flex-1 flex flex-col mt-4 min-h-0">
          <Card className="flex-1 flex flex-col min-h-0 overflow-hidden border-zinc-200">
            <CardHeader className="py-3 px-4 border-b flex flex-row items-center justify-between bg-zinc-50/50 shrink-0">
              <div>
                <CardTitle className="text-sm">proxyrelay.yaml (JSON 视图)</CardTitle>
                <CardDescription className="text-xs">来源: {configData?.path || '加载中...'}</CardDescription>
              </div>
              <Button size="sm" onClick={handleSave} disabled={isSaving}>
                <Save className="h-4 w-4 mr-2" />
                {isSaving ? '保存中...' : '保存配置'}
              </Button>
            </CardHeader>
            <CardContent className="p-0 flex-1 min-h-0 relative">
              <div className="absolute inset-0">
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
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="rendered" className="flex-1 flex flex-col mt-4 min-h-0">
          <Card className="flex-1 flex flex-col min-h-0 overflow-hidden border-zinc-200">
            <CardHeader className="py-3 px-4 border-b flex flex-row items-center justify-between bg-zinc-50/50 shrink-0">
              <div>
                <CardTitle className="text-sm">mihomo.yaml</CardTitle>
                <CardDescription className="text-xs">
                  生成时间: {renderedData?.updatedAt ? format(new Date(renderedData.updatedAt), 'PP HH:mm:ss', { locale: zhCN }) : '加载中...'}
                </CardDescription>
              </div>
              <Button size="sm" variant="outline" onClick={() => mutateRendered()}>
                <RefreshCw className="h-4 w-4 mr-2" />
                刷新渲染结果
              </Button>
            </CardHeader>
            <CardContent className="p-0 flex-1 min-h-0 relative">
              <div className="absolute inset-0">
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
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
