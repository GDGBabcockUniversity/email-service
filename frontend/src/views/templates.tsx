import { useState } from 'react'
import { toast } from 'sonner'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty } from '@/components/status'
import { Confirm } from '@/components/confirm'
import { api, templateId, useResource, type Project, type Template } from '@/lib/api'

const STARTER: Template = { name: '', subject: '', data: ['name'], content: '<h1>Hello {{.name}}</h1>\n<p>Welcome to the chapter.</p>' }

function Editor({ project, template, onClose, onSaved }: { project: Project; template: Template | null; onClose: () => void; onSaved: () => void }) {
  const [form, setForm] = useState(() => template ? { ...template, data: template.data ?? [] } : STARTER)
  const [fields, setFields] = useState(form.data.join(', '))
  const [busy, setBusy] = useState(false)
  const editing = template && templateId(template)
  const set = (patch: Partial<Template>) => setForm({ ...form, ...patch })

  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    const body = { name: form.name.trim(), subject: form.subject.trim(), content: form.content, data: fields.split(',').map((f) => f.trim()).filter(Boolean) }
    try {
      await api(`/projects/${project.id}/templates${editing ? `/${editing}` : ''}`, { method: editing ? 'PUT' : 'POST', body })
      toast.success(editing ? 'Template saved.' : 'Template created.')
      onSaved()
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not save the template.')
    } finally {
      setBusy(false)
    }
  }

  // Placeholders inside attributes (href, src) can't be highlighted without breaking the markup,
  // so they get inert values; placeholders in the text show as highlighted field names.
  const field = /\{\{\s*\.(\w+)\s*\}\}/g
  const blank = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='600' height='300'%3E%3Crect width='100%25' height='100%25' fill='%23d4d4d9'/%3E%3C/svg%3E"
  const preview = form.content
    .replace(/(\w+)="([^"]*)"/g, (_, attr: string, value: string) => `${attr}="${value.replace(field, attr === 'src' ? blank : '#')}"`)
    .replace(field, '<mark style="background:#fff3bf;padding:0 2px;border-radius:3px">$1</mark>')
  const previewSubject = form.subject.replace(field, '$1')

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="flex max-h-[80svh] flex-col sm:max-w-6xl!">
        <form onSubmit={save} className="flex min-h-0 flex-1 flex-col gap-5">
          <DialogHeader>
            <DialogTitle>{editing ? 'Edit template' : 'New template'}</DialogTitle>
            <DialogDescription>Write the body as HTML. Insert fields with {'{{.fieldName}}'}.</DialogDescription>
          </DialogHeader>
          <div className="grid min-h-0 flex-1 gap-6 overflow-y-auto md:grid-cols-[minmax(0,5fr)_minmax(0,6fr)]">
            <div className="space-y-5">
              <div className="space-y-2"><Label htmlFor="t-name">Name</Label><Input id="t-name" required maxLength={200} value={form.name} onChange={(e) => set({ name: e.target.value })} /></div>
              <div className="space-y-2"><Label htmlFor="t-subject">Subject</Label><Input id="t-subject" required maxLength={998} value={form.subject} onChange={(e) => set({ subject: e.target.value })} /></div>
              <div className="space-y-2">
                <Label htmlFor="t-fields">Fields</Label>
                <Input id="t-fields" value={fields} onChange={(e) => setFields(e.target.value)} placeholder="name, link" />
                <p className="text-sm text-muted-foreground">Comma-separated. You fill these in each time you send.</p>
              </div>
              <div className="space-y-2"><Label htmlFor="t-body">Body</Label><Textarea id="t-body" required rows={9} value={form.content} onChange={(e) => set({ content: e.target.value })} className="font-mono text-xs" /></div>
            </div>
            <div className="flex min-h-64 flex-col gap-2">
              <Label>Preview</Label>
              <div className="flex flex-1 flex-col overflow-hidden rounded-xl border bg-white text-black">
                <p className="truncate border-b bg-neutral-100 px-4 py-2 text-sm font-medium">{previewSubject || 'No subject yet'}</p>
                <iframe title="Email preview" sandbox="" srcDoc={`<body style="font-family:system-ui,sans-serif;margin:16px">${preview}</body>`} className="w-full flex-1 bg-white" />
              </div>
            </div>
          </div>
          <DialogFooter><Button type="submit" disabled={busy}>{busy ? 'Saving…' : editing ? 'Save changes' : 'Create template'}</Button></DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default function Templates({ project }: { project: Project }) {
  const templates = useResource<Template[]>(`/projects/${project.id}/templates`)
  const [editing, setEditing] = useState<Template | null | undefined>(undefined) // undefined = closed, null = new
  const list = templates.data ?? []

  const remove = async (t: Template) => {
    try {
      await api(`/projects/${project.id}/templates/${templateId(t)}`, { method: 'DELETE' })
      toast.success('Template deleted.')
      templates.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not delete.')
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-end"><Button onClick={() => setEditing(null)}><Plus /> New template</Button></div>
      {list.length === 0 && !templates.loading ? (
        <Empty title="No templates yet">A template is the email you send, with fields you fill in each time.</Empty>
      ) : (
        <ul className="divide-y rounded-2xl border">
          {list.map((t) => (
            <li key={templateId(t)} className="flex items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{t.name}</p>
                <p className="truncate text-sm text-muted-foreground">{t.subject}</p>
                {t.data?.length > 0 && <p className="mt-1 truncate font-mono text-xs text-muted-foreground">{t.data.join(' · ')}</p>}
              </div>
              <Button variant="ghost" size="icon" aria-label={`Edit ${t.name}`} onClick={() => setEditing(t)}><Pencil /></Button>
              <Confirm title={`Delete “${t.name}”?`} detail="Emails already sent are kept. You can't send this template again." action="Delete template" onConfirm={() => remove(t)}>
                <Button variant="ghost" size="icon" aria-label={`Delete ${t.name}`}><Trash2 /></Button>
              </Confirm>
            </li>
          ))}
        </ul>
      )}
      {editing !== undefined && <Editor key={editing ? templateId(editing) : 'new'} project={project} template={editing} onClose={() => setEditing(undefined)} onSaved={templates.reload} />}
    </div>
  )
}
