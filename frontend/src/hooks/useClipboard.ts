import { toast } from "sonner"

export function useClipboard() {
  const copy = async (text: string, successMessage: string) => {
    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      toast.error("当前环境不支持一键复制，请手动选择文字复制")
      return false
    }

    try {
      await navigator.clipboard.writeText(text)
      toast.success(successMessage)
      return true
    } catch {
      toast.error("复制失败，请手动选择文字进行复制")
      return false
    }
  }

  return { copy }
}
