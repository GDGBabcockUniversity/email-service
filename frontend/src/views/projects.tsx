import { useState } from 'react'
import { toast } from 'sonner'
import { Copy, KeyRound, Pencil, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty } from '@/components/status'
import { Confirm } from '@/components/confirm'
import { api, ago, useResource, type ApiKey, type Project } from '@/lib/api'

const fail = (err: unknown) => toast.error(err instanceof Error ? err.message : 'Something went wrong.')

function ProjectForm({ project, onClose, onSaved }: { project: Project | null; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(project?.name ?? '')
  const [description, setDescription] = useState(project?.description ?? '')
  const [busy, setBusy] = useState(false)
  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      await api(project ? `/projects/${project.id}` : '/projects', { method: project ? 'PUT' : 'POST', body: { name: name.trim(), description: description.trim() } })
      toast.success(project ? 'Project saved.' : 'Project created.')
      onSaved()
      onClose()
    } catch (err) {
      fail(err)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={save} className="space-y-5">
          <DialogHeader>
            <DialogTitle>{project ? 'Edit project' : 'New project'}</DialogTitle>
            <DialogDescription>Each project has its own templates, audience and API keys.</DialogDescription>
          </DialogHeader>
          <div className="space-y-2"><Label htmlFor="p-name">Name</Label><Input id="p-name" required maxLength={200} value={name} onChange={(e) => setName(e.target.value)} /></div>
          <div className="space-y-2"><Label htmlFor="p-desc">Description</Label><Textarea id="p-desc" maxLength={2000} rows={3} value={description} onChange={(e) => setDescription(e.target.value)} /></div>
          <DialogFooter><Button type="submit" disabled={busy || !name.trim()}>{project ? 'Save changes' : 'Create project'}</Button></DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function Keys({ project }: { project: Project }) {
  const keys = useResource<ApiKey[]>(`/projects/${project.id}/api-keys`)
  const [fresh, setFresh] = useState('')
  const create = async () => {
    try {
      const made = await api<{ key: string }>(`/projects/${project.id}/api-keys`, { method: 'POST' })
      setFresh(made.key)
      keys.reload()
    } catch (err) {
      fail(err)
    }
  }
  const revoke = async (k: ApiKey) => {
    try {
      await api(`/projects/${project.id}/api-keys/${k.id}`, { method: 'DELETE' })
      toast.success('Key revoked.')
      keys.reload()
    } catch (err) {
      fail(err)
    }
  }
  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-semibold">API keys for {project.name}</h2>
          <p className="text-sm text-muted-foreground">Apps send mail with a key in the <code className="font-mono">x-gdg_email-apikey</code> header.</p>
        </div>
        <Button variant="outline" onClick={create}><KeyRound /> Create key</Button>
      </div>
      {fresh && (
        <div className="space-y-2 rounded-2xl border border-google-yellow/50 bg-card p-4">
          <p className="text-sm font-medium">Copy this key now. It won't be shown again.</p>
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 break-all rounded-md bg-muted p-2 font-mono text-xs">{fresh}</code>
            <Button variant="outline" size="icon" aria-label="Copy key" onClick={() => navigator.clipboard.writeText(fresh).then(() => toast.success('Copied.'))}><Copy /></Button>
          </div>
        </div>
      )}
      {keys.data?.length === 0 ? <Empty title="No keys yet">Create one so an app can send through this project.</Empty> : (
        <ul className="divide-y rounded-2xl border">
          {keys.data?.map((k) => (
            <li key={k.id} className="flex items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="truncate font-mono text-xs">{k.id}</p>
                <p className="text-sm text-muted-foreground">{k.revoked ? `Revoked ${ago(k.revoked_at)}` : `Created ${ago(k.created_at)}`}</p>
              </div>
              {!k.revoked && (
                <Confirm title="Revoke this key?" detail="Apps using it stop being able to send straight away." action="Revoke key" onConfirm={() => revoke(k)}>
                  <Button variant="ghost" size="sm">Revoke</Button>
                </Confirm>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

export default function Projects({ projects, selected, onChange, onPick }: { projects: Project[]; selected?: Project; onChange: () => void; onPick: (id: string) => void }) {
  const [editing, setEditing] = useState<Project | null | undefined>(undefined)
  const remove = async (p: Project) => {
    try {
      await api(`/projects/${p.id}`, { method: 'DELETE' })
      toast.success('Project deleted.')
      onChange()
    } catch (err) {
      fail(err)
    }
  }
  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <div className="flex justify-end"><Button onClick={() => setEditing(null)}><Plus /> New project</Button></div>
        {projects.length === 0 ? <Empty title="No projects yet">A project groups the templates, audience and keys for one app.</Empty> : (
          <ul className="divide-y rounded-2xl border">
            {projects.map((p) => (
              <li key={p.id} className="flex items-center gap-3 p-4">
                <button className="min-w-0 flex-1 text-left" onClick={() => onPick(p.id)}>
                  <p className="truncate font-medium">{p.name}{p.id === selected?.id && <span className="ml-2 text-xs font-normal text-muted-foreground">Open</span>}</p>
                  <p className="truncate text-sm text-muted-foreground">{p.description || 'No description'}</p>
                </button>
                <Button variant="ghost" size="icon" aria-label={`Edit ${p.name}`} onClick={() => setEditing(p)}><Pencil /></Button>
                <Confirm title={`Delete “${p.name}”?`} detail="Its templates, audience and keys go with it." action="Delete project" onConfirm={() => remove(p)}>
                  <Button variant="ghost" size="icon" aria-label={`Delete ${p.name}`}><Trash2 /></Button>
                </Confirm>
              </li>
            ))}
          </ul>
        )}
      </section>
      {selected && <Keys key={selected.id} project={selected} />}
      {editing !== undefined && <ProjectForm key={editing?.id ?? 'new'} project={editing} onClose={() => setEditing(undefined)} onSaved={onChange} />}
    </div>
  )
}
