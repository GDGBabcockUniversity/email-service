import { useState } from 'react'
import { toast } from 'sonner'
import { Inbox, KeyRound, LogOut, FileText, Users, FolderKanban } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Sidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupLabel, SidebarHeader, SidebarInset, SidebarMenu,
  SidebarMenuButton, SidebarMenuItem, SidebarProvider, SidebarRail, SidebarSeparator, SidebarTrigger,
} from '@/components/ui/sidebar'
import { api, adminKey, useResource, type Project } from '@/lib/api'
import Delivery from '@/views/delivery'
import Audience from '@/views/audience'
import Templates from '@/views/templates'
import Projects from '@/views/projects'

const VIEWS = [
  { id: 'delivery', label: 'Delivery', icon: Inbox },
  { id: 'audience', label: 'Audience', icon: Users },
  { id: 'templates', label: 'Templates', icon: FileText },
  { id: 'projects', label: 'Projects & keys', icon: FolderKanban },
] as const
type ViewId = (typeof VIEWS)[number]['id']

function SignIn({ onDone }: { onDone: () => void }) {
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    try {
      await api('/projects', { key })
      adminKey.set(key)
      onDone()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Could not sign in.')
    } finally {
      setBusy(false)
    }
  }
  return (
    <main className="grid min-h-svh place-items-center p-4">
      <form onSubmit={submit} className="w-full max-w-sm space-y-6 rounded-2xl border bg-card p-8">
        <img src="/admin/gdg-logo.png" alt="Google Developer Group, Babcock University" className="h-9 w-auto" />
        <div className="space-y-1.5">
          <h1 className="text-2xl font-semibold tracking-tight">Email console</h1>
          <p className="text-sm text-muted-foreground">
            See what the service has delivered, keep the member list current, and edit the templates your projects send.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="key">Admin key</Label>
          <Input id="key" type="password" autoFocus autoComplete="current-password" value={key} onChange={(e) => setKey(e.target.value)} className="font-mono" />
        </div>
        <Button type="submit" className="w-full" disabled={!key || busy}>
          <KeyRound /> {busy ? 'Checking…' : 'Sign in'}
        </Button>
      </form>
    </main>
  )
}

function Console({ onSignOut }: { onSignOut: () => void }) {
  const projects = useResource<Project[]>('/projects')
  const [view, setView] = useState<ViewId>('delivery')
  const [chosen, setChosen] = useState(() => localStorage.getItem('gdg-project') ?? '')
  const list = projects.data ?? []
  // A stale id (deleted project) falls back to the first one.
  const project = list.find((p) => p.id === chosen) ?? list[0]
  const pick = (id: string) => {
    setChosen(id)
    localStorage.setItem('gdg-project', id)
  }
  const title = VIEWS.find((v) => v.id === view)!.label

  return (
    <SidebarProvider>
      <Sidebar collapsible="icon">
        <SidebarHeader className="gap-4 p-3">
          <div className="flex h-9 items-center px-1">
            <img src="/admin/gdg-logo.png" alt="GDG Babcock" className="h-7 w-auto group-data-[collapsible=icon]:hidden" />
            <img src="/admin/favicon.svg" alt="GDG Babcock" className="hidden size-6 group-data-[collapsible=icon]:block" />
          </div>
          <div className="space-y-1.5 group-data-[collapsible=icon]:hidden">
            <Label className="px-1 text-xs text-muted-foreground">Project</Label>
            <Select value={project?.id ?? ''} onValueChange={pick} disabled={!list.length}>
              <SelectTrigger className="w-full"><SelectValue placeholder="No projects yet" /></SelectTrigger>
              <SelectContent>
                {list.map((p) => <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
        </SidebarHeader>
        <SidebarSeparator />
        <SidebarContent>
          <SidebarGroup className="p-3">
            <SidebarGroupLabel>Console</SidebarGroupLabel>
            <SidebarMenu className="gap-1.5">
              {VIEWS.map(({ id, label, icon: Icon }) => (
                <SidebarMenuItem key={id}>
                  <SidebarMenuButton className="h-10 gap-3" tooltip={label} isActive={view === id} onClick={() => setView(id)}>
                    <Icon /> <span>{label}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <SidebarSeparator />
        <SidebarFooter className="p-3">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton tooltip="Sign out" className="h-10 gap-3 text-muted-foreground" onClick={onSignOut}>
                <LogOut /> <span>Sign out</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset>
        <header className="flex h-14 items-center gap-3 border-b px-4">
          <SidebarTrigger />
          <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
          {project && view !== 'projects' && <span className="text-sm text-muted-foreground">{project.name}</span>}
        </header>
        <div className="mx-auto w-full max-w-5xl p-4 md:p-8">
          {projects.loading ? null : view === 'projects' ? (
            <Projects projects={list} selected={project} onChange={projects.reload} onPick={pick} />
          ) : !project ? (
            <div className="rounded-2xl border border-dashed p-10 text-center">
              <p className="font-medium">No projects yet</p>
              <p className="mt-1 text-sm text-muted-foreground">Create a project to start sending.</p>
              <Button className="mt-4" onClick={() => setView('projects')}>Create a project</Button>
            </div>
          ) : view === 'delivery' ? <Delivery project={project} />
            : view === 'audience' ? <Audience project={project} />
            : <Templates project={project} />}
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}

export default function App() {
  const [signedIn, setSignedIn] = useState(!!adminKey.get())
  return signedIn ? (
    <Console onSignOut={() => { adminKey.clear(); setSignedIn(false) }} />
  ) : (
    <SignIn onDone={() => setSignedIn(true)} />
  )
}
