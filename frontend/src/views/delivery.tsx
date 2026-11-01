import { useState } from 'react'
import { toast } from 'sonner'
import { Send } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Empty, STATUS, StatusBadge } from '@/components/status'
import { api, ago, useResource, type Email, type Project, type Stats, type Template } from '@/lib/api'

const FILTERS = [['', 'All'], ['PENDING', 'Queued'], ['PROCESSING', 'Sending'], ['SENT', 'Sent'], ['FAILED', 'Failed']] as const

// One bar, four colours: how the whole queue splits across delivery states.
function Pipeline({ stats }: { stats: Stats }) {
  const parts = [['PENDING', stats.pending], ['PROCESSING', stats.processing], ['SENT', stats.sent], ['FAILED', stats.failed]] as const
  return (
    <section className="space-y-4 rounded-2xl border bg-card p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-4xl font-semibold tracking-tight tabular-nums">{stats.total.toLocaleString()}</p>
        <p className="text-sm text-muted-foreground">emails handled by this project</p>
      </div>
      <div className="flex h-3 gap-0.5 overflow-hidden rounded-full bg-muted" role="img"
        aria-label={parts.map(([k, n]) => `${n} ${STATUS[k].label.toLowerCase()}`).join(', ')}>
        {parts.map(([k, n]) => n > 0 && <div key={k} className={STATUS[k].dot} style={{ flexGrow: n }} />)}
      </div>
      <dl className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {parts.map(([k, n]) => (
          <div key={k}>
            <dt className="flex items-center gap-2 text-sm text-muted-foreground"><span className={`size-2 rounded-full ${STATUS[k].dot}`} />{STATUS[k].label}</dt>
            <dd className="text-xl font-medium tabular-nums">{n.toLocaleString()}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

function Compose({ project, open, onClose, onSent }: { project: Project; open: boolean; onClose: () => void; onSent: () => void }) {
  const templates = useResource<Template[]>(open ? `/projects/${project.id}/templates` : null)
  const [name, setName] = useState('')
  const [audience, setAudience] = useState<'all' | 'one'>('all')
  const [address, setAddress] = useState('')
  const [values, setValues] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const list = templates.data ?? []
  const template = list.find((t) => t.name === name) ?? list[0]

  const send = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!template) return
    setBusy(true)
    try {
      const queued = await api<unknown[]>(`/projects/${project.id}/emails`, {
        method: 'POST',
        body: { to: audience === 'all' ? 'all' : address.trim().toLowerCase(), template: template.name, data: values },
      })
      toast.success(`Queued ${queued?.length ?? 0} ${queued?.length === 1 ? 'email' : 'emails'}.`)
      onSent()
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not send.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-md">
        <form onSubmit={send} className="flex flex-col gap-5">
          <DialogHeader>
            <DialogTitle>Send an email</DialogTitle>
            <DialogDescription>Emails are queued and delivered in the background.</DialogDescription>
          </DialogHeader>
          <div className="space-y-5">
            {list.length === 0 ? (
              <Empty title="No templates yet">Create a template before sending.</Empty>
            ) : (
              <>
                <div className="space-y-2">
                  <Label>Template</Label>
                  <Select value={template?.name} onValueChange={(v) => { setName(v); setValues({}) }}>
                    <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>{list.map((t) => <SelectItem key={t.name} value={t.name}>{t.name}</SelectItem>)}</SelectContent>
                  </Select>
                  {template && <p className="text-sm text-muted-foreground">Subject: {template.subject}</p>}
                </div>
                <div className="space-y-2">
                  <Label>Send to</Label>
                  <RadioGroup value={audience} onValueChange={(v) => setAudience(v as 'all' | 'one')}>
                    <Label className="font-normal"><RadioGroupItem value="all" /> Everyone subscribed in this project</Label>
                    <Label className="font-normal"><RadioGroupItem value="one" /> One address</Label>
                  </RadioGroup>
                  {audience === 'one' && <Input type="email" required placeholder="name@example.com" value={address} onChange={(e) => setAddress(e.target.value)} />}
                </div>
                {template?.data?.map((field) => (
                  <div key={field} className="space-y-2">
                    <Label htmlFor={`f-${field}`}>{field}</Label>
                    <Input id={`f-${field}`} required value={values[field] ?? ''} onChange={(e) => setValues({ ...values, [field]: e.target.value })} />
                  </div>
                ))}
              </>
            )}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={!template || busy}><Send /> {busy ? 'Sending…' : 'Send'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default function Delivery({ project }: { project: Project }) {
  const [status, setStatus] = useState('')
  const [composing, setComposing] = useState(false)
  const stats = useResource<Stats>(`/projects/${project.id}/emails/stats`, 5000)
  const emails = useResource<Email[]>(`/projects/${project.id}/emails?limit=100${status ? `&status=${status}` : ''}`, 5000)
  const reload = () => { stats.reload(); emails.reload() }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <Tabs value={status} onValueChange={setStatus}>
          <TabsList>{FILTERS.map(([v, l]) => <TabsTrigger key={v} value={v}>{l}</TabsTrigger>)}</TabsList>
        </Tabs>
        <Button onClick={() => setComposing(true)}><Send /> Send an email</Button>
      </div>
      {stats.data && <Pipeline stats={stats.data} />}
      {emails.data && emails.data.length === 0 ? (
        <Empty title={status ? 'Nothing with that status' : 'No emails yet'}>
          {status ? 'Try another filter.' : 'Send your first email and it will show up here.'}
        </Empty>
      ) : (
        <div className="overflow-x-auto rounded-2xl border">
          <Table>
            <TableHeader>
              <TableRow><TableHead>Recipient</TableHead><TableHead>Subject</TableHead><TableHead>Status</TableHead><TableHead className="text-right">Queued</TableHead></TableRow>
            </TableHeader>
            <TableBody>
              {emails.data?.map((m) => (
                <TableRow key={m.id}>
                  <TableCell className="font-mono text-xs">{m.to}</TableCell>
                  <TableCell className="max-w-64 truncate">{m.subject}</TableCell>
                  <TableCell>
                    <StatusBadge status={m.status} />
                    {m.status === 'FAILED' && m.last_error && <p className="max-w-64 truncate text-xs text-muted-foreground" title={m.last_error}>{m.last_error}</p>}
                  </TableCell>
                  <TableCell className="text-right text-muted-foreground" title={new Date(m.created_at).toLocaleString()}>{ago(m.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
      {emails.error && <p className="text-sm text-destructive">{emails.error}</p>}
      <Compose key={String(composing)} project={project} open={composing} onClose={() => setComposing(false)} onSent={reload} />
    </div>
  )
}
