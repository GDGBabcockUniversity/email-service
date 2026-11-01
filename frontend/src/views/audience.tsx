import { useState } from 'react'
import { toast } from 'sonner'
import { Trash2, UserPlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Empty } from '@/components/status'
import { api, ago, useResource, type Member, type Project } from '@/lib/api'

const ADDRESS = /^[^@\s]+@[^@\s.]+\.[^@\s]+$/

export default function Audience({ project }: { project: Project }) {
  const base = `/projects/${project.id}/audience`
  const members = useResource<Member[]>(base)
  const [query, setQuery] = useState('')
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const all = members.data ?? []
  const shown = all.filter((m) => m.email.toLowerCase().includes(query.trim().toLowerCase()))

  const add = async (e: React.FormEvent) => {
    e.preventDefault()
    const emails = [...new Set(text.toLowerCase().split(/[\s,;]+/).filter(Boolean))]
    const bad = emails.find((a) => !ADDRESS.test(a) || a.length > 254)
    if (bad) return toast.error(`${bad} is not a valid email address.`)
    if (!emails.length) return toast.error('Add at least one email address.')
    setBusy(true)
    try {
      await api(base, { method: 'POST', body: { audiences: emails.map((email) => ({ email })) } })
      toast.success(`Added ${emails.length} ${emails.length === 1 ? 'member' : 'members'}.`)
      setText('')
      members.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not add members.')
    } finally {
      setBusy(false)
    }
  }

  const remove = async (m: Member) => {
    try {
      await api(`${base}/${m.id}`, { method: 'DELETE' })
      toast.success('Removed.')
      members.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not remove.')
    }
  }

  return (
    <div className="space-y-6">
      <form onSubmit={add} className="space-y-3 rounded-2xl border bg-card p-5">
        <label htmlFor="addresses" className="text-sm font-medium">Add members</label>
        <Textarea id="addresses" rows={3} placeholder="Paste addresses separated by commas, spaces or new lines" value={text} onChange={(e) => setText(e.target.value)} className="font-mono text-sm" />
        <Button type="submit" disabled={!text.trim() || busy}><UserPlus /> Add to audience</Button>
      </form>
      {all.length === 0 && !members.loading ? (
        <Empty title="No members yet">Add addresses above and they will receive this project's emails.</Empty>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <Input type="search" aria-label="Search members" placeholder="Search addresses" value={query} onChange={(e) => setQuery(e.target.value)} className="max-w-xs" />
            <p className="text-sm text-muted-foreground">{shown.length === all.length ? `${all.length} members` : `${shown.length} of ${all.length}`}</p>
          </div>
          <div className="overflow-x-auto rounded-2xl border">
            <Table>
              <TableHeader><TableRow><TableHead>Address</TableHead><TableHead>Status</TableHead><TableHead>Added</TableHead><TableHead className="w-12"><span className="sr-only">Remove</span></TableHead></TableRow></TableHeader>
              <TableBody>
                {shown.map((m) => (
                  <TableRow key={m.id}>
                    <TableCell className="font-mono text-xs">{m.email}</TableCell>
                    <TableCell className={m.subscribed ? '' : 'text-muted-foreground'}>{m.subscribed ? 'Subscribed' : 'Unsubscribed'}</TableCell>
                    <TableCell className="text-muted-foreground">{ago(m.created_at)}</TableCell>
                    <TableCell><Button variant="ghost" size="icon" aria-label={`Remove ${m.email}`} onClick={() => remove(m)}><Trash2 /></Button></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}
    </div>
  )
}
