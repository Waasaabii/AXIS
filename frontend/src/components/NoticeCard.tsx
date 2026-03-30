import type { ReactNode } from "react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

type NoticeTone = "neutral" | "warning" | "success"

const toneStyles: Record<NoticeTone, string> = {
  neutral: "border-zinc-200 bg-white/90",
  warning: "border-amber-200 bg-amber-50/80",
  success: "border-emerald-200 bg-emerald-50/80",
}

interface NoticeCardProps {
  title: string
  description: string
  icon?: ReactNode
  tone?: NoticeTone
  action?: ReactNode
  className?: string
}

export function NoticeCard({
  title,
  description,
  icon,
  tone = "neutral",
  action,
  className,
}: NoticeCardProps) {
  return (
    <Card className={cn("shadow-sm", toneStyles[tone], className)}>
      <CardHeader className="gap-3 pb-3">
        <div className="flex items-start gap-3">
          {icon ? (
            <div className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-white text-zinc-700 shadow-sm">
              {icon}
            </div>
          ) : null}
          <div className="space-y-1">
            <CardTitle className="text-base text-zinc-950">{title}</CardTitle>
            <CardDescription className="text-sm leading-6 text-zinc-600">{description}</CardDescription>
          </div>
        </div>
      </CardHeader>
      {action ? <CardContent className="pt-0">{action}</CardContent> : null}
    </Card>
  )
}
