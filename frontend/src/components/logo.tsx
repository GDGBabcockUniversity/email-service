import { cn } from '@/lib/utils'

// The black-wordmark logo reads on light surfaces, the white-wordmark one on dark.
export function Logo({ className }: { className?: string }) {
  const alt = 'Google Developer Group, Babcock University'
  return (
    <>
      <img src="/admin/gdg-logo.png" alt={alt} className={cn('dark:hidden', className)} />
      <img src="/admin/gdg-logo-light.png" alt={alt} className={cn('hidden dark:block', className)} />
    </>
  )
}
