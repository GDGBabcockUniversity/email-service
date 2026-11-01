import { useCallback, useEffect, useState } from 'react'

export type Project = { id: string; name: string; description: string; created_at: string }
export type Stats = { total: number; pending: number; processing: number; sent: number; failed: number }
export type Email = {
  id: string; to: string; subject: string; status: 'PENDING' | 'PROCESSING' | 'SENT' | 'FAILED'
  attempts: number; last_error?: string; created_at: string; sent_at: string | null
}
export type Member = { id: string; email: string; created_at: string; subscribed: boolean }
export type Template = { id?: string; Id?: string; name: string; subject: string; data: string[]; content: string }
export type ApiKey = { id: string; created_at: string; revoked: boolean; revoked_at: string | null }

const KEY = 'gdg-admin-key'
export const adminKey = {
  get: () => sessionStorage.getItem(KEY) ?? '',
  set: (value: string) => sessionStorage.setItem(KEY, value),
  clear: () => sessionStorage.removeItem(KEY),
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function api<T = unknown>(path: string, init: { method?: string; body?: unknown; key?: string } = {}): Promise<T> {
  const response = await fetch(`/api${path}`, {
    method: init.method ?? 'GET',
    headers: { 'X-Admin-Key': init.key ?? adminKey.get(), ...(init.body ? { 'Content-Type': 'application/json' } : {}) },
    body: init.body ? JSON.stringify(init.body) : undefined,
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    const message = response.status === 401 ? 'That admin key was not accepted.' : payload.message || payload.error || 'Something went wrong. Try again.'
    throw new ApiError(response.status, message)
  }
  return payload.data as T
}

// Loads a resource and reloads it every pollMs. A null path waits for a project.
export function useResource<T>(path: string | null, pollMs = 0) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const load = useCallback(async () => {
    if (!path) return
    try {
      setData(await api<T>(path))
      setError(null)
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        adminKey.clear()
        window.location.reload()
      }
      setError(e instanceof Error ? e.message : 'Something went wrong.')
    } finally {
      setLoading(false)
    }
  }, [path])
  useEffect(() => {
    setLoading(true)
    setData(null)
    load()
    if (!pollMs) return
    const timer = setInterval(load, pollMs)
    return () => clearInterval(timer)
  }, [load, pollMs])
  return { data, error, loading, reload: load }
}

export const templateId = (t: Template) => (t.id ?? t.Id) as string

export const ago = (iso: string | null) => {
  if (!iso) return '—'
  const minutes = (Date.now() - new Date(iso).getTime()) / 60000
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${Math.floor(minutes)}m ago`
  if (minutes < 1440) return `${Math.floor(minutes / 60)}h ago`
  if (minutes < 43200) return `${Math.floor(minutes / 1440)}d ago`
  return new Date(iso).toLocaleDateString()
}
