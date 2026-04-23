import { toast } from "sonner"

import { ApiError } from "@/services/api"

export function toastApiError(error: unknown, fallbackMessage: string) {
  if (error instanceof ApiError) {
    toast.error(error.message || fallbackMessage)
    return
  }
  toast.error(fallbackMessage)
}

