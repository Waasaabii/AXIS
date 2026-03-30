import type { ReactNode } from "react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

interface EmptyStateCardProps {
  title: string
  description: string
  icon?: ReactNode
  action?: ReactNode
  className?: string
}

export function EmptyStateCard({ title, description, icon, action, className }: EmptyStateCardProps) {
  return (
    <Card className={cn("border-dashed border-zinc-200 bg-zinc-50/80 shadow-none", className)}>
      <CardHeader className="items-center text-center">
        {icon ? (
          <div className="flex h-12 w-12 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-500">
            {icon}
          </div>
        ) : null}
        <div className="space-y-1">
          <CardTitle className="text-lg text-zinc-950">{title}</CardTitle>
          <CardDescription className="mx-auto max-w-xl text-sm leading-6 text-zinc-600">
            {description}
          </CardDescription>
        </div>
      </CardHeader>
      {action ? <CardContent className="flex justify-center pt-0">{action}</CardContent> : null}
    </Card>
  )
}
