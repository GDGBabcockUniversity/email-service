import { cn } from '@/lib/utils'
import type { Email } from '@/lib/api'

// Each Google colour means one delivery state, here and in the pipeline bar.
export const STATUS = {
  PENDING: { label: 'Queued', dot: 'bg-google-blue' },
  PROCESSING: { label: 'Sending', dot: 'bg-google-yellow' },
  SENT: { label: 'Sent', dot: 'bg-google-green' },
  FAILED: { label: 'Failed', dot: 'bg-google-red' },
} as const

export function StatusBadge({ status }: { status: Email['status'] }) {
  const s = STATUS[status]
  return (
    <span className="inline-flex items-center gap-2 text-sm">
      <span className={cn('size-2 rounded-full', s.dot)} /> {s.label}
    </span>
  )
}

export function Empty({ title, children }: { title: string; children?: React.ReactNode }) {
  return (
    <div className="rounded-2xl border border-dashed p-10 text-center">
      <p className="font-medium">{title}</p>
      {children && <p className="mt-1 text-sm text-muted-foreground">{children}</p>}
    </div>
  )
}
