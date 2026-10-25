/* Admin console for the GDG Babcock email service.
   No build step: this talks straight to the same /api routes the service exposes,
   signing every request with the admin key the operator pastes in. */

(() => {
  "use strict";

  const KEY_STORE = "gdg.email.adminKey";
  const PROJECT_STORE = "gdg.email.projectId";
  const REFRESH_MS = 10000;
  // Footer the service fills in per recipient. Every template gets one so no
  // send goes out without a way for the reader to opt back out.
  const UNSUBSCRIBE_FOOTER =
    '<p><a href="{{ .UnsubscribeURL }}">Unsubscribe</a></p>';

  const state = {
    key: "",
    projects: [],
    projectId: "",
    view: "delivery",
    audience: [],
    templates: [],
    editing: null,
    editingProject: null,
  };

  let refreshTimer = null;
  let previewTimer = null;
  let toastTimer = null;

  const $ = (id) => document.getElementById(id);

  /* ------------------------------------------------------------ helpers */

  function el(tag, attrs = {}, ...children) {
    const node = document.createElement(tag);
    for (const [name, value] of Object.entries(attrs)) {
      if (value === null || value === undefined || value === false) continue;
      if (name === "class") node.className = value;
      else if (name === "text") node.textContent = value;
      else if (name.startsWith("on")) node.addEventListener(name.slice(2), value);
      else node.setAttribute(name, value === true ? "" : String(value));
    }
    for (const child of children.flat()) {
      if (child === null || child === undefined || child === false) continue;
      node.append(child instanceof Node ? child : document.createTextNode(String(child)));
    }
    return node;
  }

  function fill(parent, ...children) {
    parent.replaceChildren(...children.flat().filter(Boolean));
  }

  function emptyRow(columns, title, detail) {
    return el(
      "tr",
      {},
      el(
        "td",
        { class: "empty", colspan: columns },
        el("strong", { text: title }),
        detail || null,
      ),
    );
  }

  function when(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime()) || date.getFullYear() < 1970) return "—";
    const seconds = (Date.now() - date.getTime()) / 1000;
    if (seconds < 45) return "just now";
    if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.round(seconds / 3600)}h ago`;
    return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  }

  function exactly(value) {
    if (!value) return "";
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "" : date.toLocaleString();
  }

  const STATUS_LABELS = {
    SENT: "Sent",
    PROCESSING: "Sending",
    PENDING: "Queued",
    FAILED: "Failed",
  };

  function statusChip(status) {
    const label = STATUS_LABELS[status] || status || "Unknown";
    return el("span", { class: `chip chip--${String(status).toLowerCase()}`, text: label });
  }

  function toast(message, isError = false) {
    const existing = document.querySelector(".toast");
    if (existing) existing.remove();
    clearTimeout(toastTimer);
    const node = el("div", {
      class: isError ? "toast toast--error" : "toast",
      role: "status",
      "aria-live": "polite",
      text: message,
    });
    document.body.append(node);
    toastTimer = setTimeout(() => node.remove(), isError ? 6000 : 3000);
  }

  /* ---------------------------------------------------------------- api */

  class ApiError extends Error {
    constructor(message, status) {
      super(message);
      this.status = status;
    }
  }

  async function api(path, { method = "GET", body } = {}) {
    const headers = { "X-Admin-Key": state.key };
    if (body !== undefined) headers["Content-Type"] = "application/json";
    const response = await fetch(path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    let payload = null;
    try {
      payload = await response.json();
    } catch {
      payload = null;
    }
    if (!response.ok) {
      throw new ApiError(
        (payload && payload.message) || `The server returned ${response.status}.`,
        response.status,
      );
    }
    return payload ? payload.data : null;
  }

  // Wraps an action so one failed request reports itself instead of dying silently.
  async function run(action, { silent = false } = {}) {
    try {
      return await action();
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        signOut("Your admin key was rejected. Sign in again.");
        return undefined;
      }
      if (!silent) toast(error.message, true);
      return undefined;
    }
  }

  /* ------------------------------------------------------------- signin */

  function signOut(message) {
    state.key = "";
    state.projects = [];
    state.projectId = "";
    stopRefresh();
    try {
      localStorage.removeItem(KEY_STORE);
    } catch {
      /* storage may be unavailable; the in-memory key is already cleared */
    }
    $("shell").hidden = true;
    $("gate").hidden = false;
    $("gate-key").value = "";
    showGateError(message);
  }

  function showGateError(message) {
    const node = $("gate-error");
    node.textContent = message || "";
    node.hidden = !message;
  }

  async function signIn(key) {
    state.key = key;
    const projects = await api("/api/projects");
    state.projects = projects || [];
    try {
      localStorage.setItem(KEY_STORE, key);
    } catch {
      /* a session without storage still works, it just won't be remembered */
    }
    showGateError("");
    $("gate").hidden = true;
    $("shell").hidden = false;
    let saved = "";
    try {
      saved = localStorage.getItem(PROJECT_STORE) || "";
    } catch {
      saved = "";
    }
    state.projectId = state.projects.some((p) => p.id === saved)
      ? saved
      : (state.projects[0] && state.projects[0].id) || "";
    renderProjectPicker();
    setView(state.view);
  }

  /* -------------------------------------------------------------- shell */

  // The topbar picker and the audience form's picker are two views of one choice,
  // so they are filled from the same state and always agree.
  function renderProjectPicker() {
    for (const id of ["project-picker", "audience-project"]) {
      const picker = $(id);
      if (!picker) continue;
      if (!state.projects.length) {
        fill(picker, el("option", { value: "", text: "No projects yet" }));
        picker.disabled = true;
        continue;
      }
      picker.disabled = false;
      fill(
        picker,
        state.projects.map((project) =>
          el("option", {
            value: project.id,
            text: project.name,
            selected: project.id === state.projectId,
          }),
        ),
      );
      picker.value = state.projectId;
    }
  }

  function setView(view) {
    state.view = view;
    for (const link of document.querySelectorAll(".navlink")) {
      if (link.dataset.view === view) link.setAttribute("aria-current", "page");
      else link.removeAttribute("aria-current");
    }
    for (const section of document.querySelectorAll(".view")) {
      section.hidden = section.id !== `view-${view}`;
    }
    stopRefresh();
    if (view === "delivery") {
      loadDelivery();
      startRefresh();
    } else if (view === "audience") {
      loadAudience();
    } else if (view === "templates") {
      loadTemplates();
    } else if (view === "projects") {
      renderProjects();
      loadKeys();
    }
  }

  function startRefresh() {
    if (!$("auto-refresh").checked) return;
    refreshTimer = setInterval(() => loadDelivery({ silent: true }), REFRESH_MS);
  }

  function stopRefresh() {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }

  function requireProject() {
    return Boolean(state.projectId);
  }

  /* ----------------------------------------------------------- delivery */

  async function loadDelivery({ silent = false } = {}) {
    if (!requireProject()) {
      renderStats({ total: 0, sent: 0, processing: 0, pending: 0, failed: 0 });
      fill(
        $("email-rows"),
        emptyRow(6, "No project selected", "Create a project to start tracking delivery."),
      );
      return;
    }
    const status = $("email-status").value;
    const query = status ? `?status=${encodeURIComponent(status)}&limit=100` : "?limit=100";
    const result = await run(
      async () =>
        Promise.all([
          api(`/api/projects/${state.projectId}/emails/stats`),
          api(`/api/projects/${state.projectId}/emails${query}`),
        ]),
      { silent },
    );
    if (!result) return;
    const [stats, emails] = result;
    renderStats(stats || {});
    renderEmails(emails || []);
    $("refresh-stamp").textContent = `Updated ${new Date().toLocaleTimeString()}`;
  }

  function renderStats(stats) {
    const counts = {
      sent: stats.sent || 0,
      processing: stats.processing || 0,
      pending: stats.pending || 0,
      failed: stats.failed || 0,
      total: stats.total || 0,
    };
    for (const [name, value] of Object.entries(counts)) {
      const node = document.querySelector(`[data-tally="${name}"]`);
      if (node) node.textContent = value.toLocaleString();
    }
    $("delivery-total").textContent =
      counts.total === 0
        ? "No emails yet."
        : `${counts.total.toLocaleString()} ${counts.total === 1 ? "email" : "emails"} in total.`;
    const bar = $("delivery-bar");
    bar.classList.toggle("deliverybar--empty", counts.total === 0);
    for (const segment of bar.children) {
      const share = counts[segment.dataset.status.toLowerCase()] || 0;
      segment.style.flexGrow = String(share);
      segment.hidden = share === 0;
    }
    bar.setAttribute(
      "aria-label",
      counts.total === 0
        ? "No emails yet"
        : `${counts.sent} sent, ${counts.processing} sending, ${counts.pending} queued, ${counts.failed} failed`,
    );
  }

  function renderEmails(emails) {
    if (!emails.length) {
      fill(
        $("email-rows"),
        emptyRow(
          6,
          "Nothing here yet",
          "Emails queued through the API appear within a few seconds.",
        ),
      );
      return;
    }
    fill(
      $("email-rows"),
      emails.map((email) =>
        el(
          "tr",
          {},
          el("td", { class: "cell-mono", text: email.to }),
          el(
            "td",
            {},
            email.subject,
            email.last_error ? el("div", { class: "cell-error", text: email.last_error }) : null,
          ),
          el("td", {}, statusChip(email.status)),
          el("td", { class: "cell-faint", text: String(email.attempts ?? 0) }),
          el("td", { class: "cell-faint", title: exactly(email.created_at), text: when(email.created_at) }),
          el("td", { class: "cell-faint", title: exactly(email.sent_at), text: when(email.sent_at) }),
        ),
      ),
    );
  }

  /* ----------------------------------------------------------- audience */

  async function loadAudience() {
    if (!requireProject()) {
      fill(
        $("audience-rows"),
        emptyRow(4, "No project selected", "Create a project before building an audience."),
      );
      return;
    }
    const audience = await run(() => api(`/api/projects/${state.projectId}/audience`));
    if (!audience) return;
    state.audience = audience;
    renderAudience();
  }

  function renderAudience() {
    const term = $("audience-search").value.trim().toLowerCase();
    const matches = term
      ? state.audience.filter((person) => person.email.toLowerCase().includes(term))
      : state.audience;
    const total = state.audience.length;
    $("audience-count").textContent = total
      ? `${total.toLocaleString()} ${total === 1 ? "member" : "members"}`
      : "";
    if (!matches.length) {
      fill(
        $("audience-rows"),
        term
          ? emptyRow(4, "No match", `Nothing on the list contains “${term}”.`)
          : emptyRow(4, "No one on the list yet", "Paste addresses above to add your first members."),
      );
      return;
    }
    fill(
      $("audience-rows"),
      matches.map((person) =>
        el(
          "tr",
          {},
          el("td", { class: "cell-mono", text: person.email }),
          el("td", {
            class: person.subscribed === false ? "cell-faint" : "",
            text: person.subscribed === false ? "Unsubscribed" : "Subscribed",
          }),
          el("td", {
            class: "cell-faint",
            title: exactly(person.created_at),
            text: when(person.created_at),
          }),
          el(
            "td",
            { class: "is-right" },
            el("button", {
              class: "btn btn--quiet btn--danger",
              type: "button",
              text: "Remove",
              onclick: () => removeMember(person),
            }),
          ),
        ),
      ),
    );
  }

  function parseAddresses(raw) {
    const seen = new Set();
    const valid = [];
    const invalid = [];
    for (const candidate of raw.split(/[\s,;]+/)) {
      const address = candidate.trim().toLowerCase();
      if (!address) continue;
      if (seen.has(address)) continue;
      seen.add(address);
      if (/^[^@\s]+@[^@\s.]+\.[^@\s]+$/.test(address) && address.length <= 254) valid.push(address);
      else invalid.push(address);
    }
    return { valid, invalid };
  }

  async function addMembers(event) {
    event.preventDefault();
    if (!requireProject()) {
      toast("Create a project first.", true);
      return;
    }
    const { valid, invalid } = parseAddresses($("audience-input").value);
    if (invalid.length) {
      toast(`${invalid[0]} is not a valid email address.`, true);
      return;
    }
    if (!valid.length) {
      toast("Add at least one email address.", true);
      return;
    }
    const added = await run(() =>
      api(`/api/projects/${state.projectId}/audience`, {
        method: "POST",
        body: { audiences: valid.map((email) => ({ email })) },
      }),
    );
    if (added === undefined) return;
    $("audience-input").value = "";
    $("audience-parsed").textContent = "";
    const project = state.projects.find((entry) => entry.id === state.projectId);
    const who = `${valid.length} ${valid.length === 1 ? "member" : "members"}`;
    toast(project ? `Added ${who} to ${project.name}.` : `Added ${who}.`);
    loadAudience();
  }

  async function removeMember(person) {
    if (!confirm(`Remove ${person.email} from this audience?`)) return;
    const result = await run(() =>
      api(`/api/projects/${state.projectId}/audience/${person.id}`, { method: "DELETE" }),
    );
    if (result === undefined) return;
    toast(`Removed ${person.email}.`);
    loadAudience();
  }

  /* ------------------------------------------------------------ compose */

  // The console sends with the admin key, so it posts to the project's own send
  // route rather than holding a project API key in the browser.
  async function openCompose() {
    if (!requireProject()) {
      toast("Create a project first.", true);
      return;
    }
    const templates = await run(() => api(`/api/projects/${state.projectId}/templates`));
    if (!templates) return;
    state.templates = templates;
    const picker = $("send-template");
    const hint = $("send-hint");
    if (!templates.length) {
      fill(picker, el("option", { value: "", text: "No templates yet" }));
      picker.disabled = true;
      $("send-submit").disabled = true;
      hint.textContent = "This project has no templates. Create one before sending.";
      hint.hidden = false;
    } else {
      picker.disabled = false;
      $("send-submit").disabled = false;
      hint.hidden = true;
      fill(picker, templates.map((t) => el("option", { value: t.name, text: t.name })));
    }
    $("send-audience").value = "all";
    $("send-address").value = "";
    $("send-address-field").hidden = true;
    renderComposeValues();
    $("send-sheet").showModal();
  }

  // One input per value the chosen template declares, so the sender can see
  // exactly what the template expects.
  function renderComposeValues() {
    const template = state.templates.find((t) => t.name === $("send-template").value);
    const names = (template && template.data) || [];
    const container = $("send-values");
    container.hidden = names.length === 0;
    fill(
      container,
      names.length
        ? [
            el("span", {}, "Values", el("span", { class: "field__hint", text: " Filled into the subject and body" })),
            ...names.map((name) =>
              el("input", {
                "data-value": name,
                placeholder: name,
                "aria-label": name,
              }),
            ),
          ]
        : [],
    );
  }

  async function sendCompose() {
    const template = $("send-template").value;
    if (!template) return;
    const toAll = $("send-audience").value === "all";
    const address = $("send-address").value.trim();
    if (!toAll && !/^[^@\s]+@[^@\s.]+\.[^@\s]+$/.test(address)) {
      toast("Enter a valid email address.", true);
      return;
    }
    const data = {};
    for (const input of $("send-values").querySelectorAll("[data-value]")) {
      const value = input.value.trim();
      if (!value) {
        toast(`Fill in ${input.dataset.value} before sending.`, true);
        return;
      }
      data[input.dataset.value] = value;
    }
    $("send-submit").disabled = true;
    const accepted = await run(() =>
      api(`/api/projects/${state.projectId}/emails`, {
        method: "POST",
        body: { to: toAll ? "all" : address, template, data },
      }),
    );
    $("send-submit").disabled = false;
    if (accepted === undefined) return;
    const count = Array.isArray(accepted) ? accepted.length : 0;
    if (count === 0) {
      toast("Nobody to send to. Add someone to the audience first.", true);
      return;
    }
    $("send-sheet").close();
    toast(`Queued ${count} ${count === 1 ? "email" : "emails"}.`);
    setView("delivery");
  }

  /* ---------------------------------------------------------- templates */

  async function loadTemplates() {
    if (!requireProject()) {
      fill(
        $("template-rows"),
        emptyRow(5, "No project selected", "Create a project before adding templates."),
      );
      return;
    }
    const templates = await run(() => api(`/api/projects/${state.projectId}/templates`));
    if (!templates) return;
    state.templates = templates;
    renderTemplates();
  }

  function renderTemplates() {
    if (!state.templates.length) {
      fill(
        $("template-rows"),
        emptyRow(5, "No templates yet", "Create one to give this project something to send."),
      );
      return;
    }
    fill(
      $("template-rows"),
      state.templates.map((template) =>
        el(
          "tr",
          {},
          el("td", {}, el("strong", { text: template.name })),
          el("td", { text: template.subject }),
          el(
            "td",
            {},
            (template.data || []).length
              ? el(
                  "div",
                  { class: "chiprow" },
                  template.data.map((name) => el("span", { class: "tag", text: name })),
                )
              : el("span", { class: "cell-faint", text: "None" }),
          ),
          el("td", {
            class: "cell-faint",
            title: exactly(template.created_at),
            text: when(template.created_at),
          }),
          el(
            "td",
            { class: "is-right" },
            el(
              "div",
              { class: "filters", style: "justify-content:flex-end" },
              el("button", {
                class: "btn btn--quiet",
                type: "button",
                text: "Edit",
                onclick: () => openTemplate(template),
              }),
              el("button", {
                class: "btn btn--quiet btn--danger",
                type: "button",
                text: "Delete",
                onclick: () => deleteTemplate(template),
              }),
            ),
          ),
        ),
      ),
    );
  }

  function openTemplate(template) {
    state.editing = template || null;
    $("template-sheet-title").textContent = template ? "Edit template" : "New template";
    $("template-name").value = template ? template.name : "";
    $("template-subject").value = template ? template.subject : "";
    $("template-data").value = template ? (template.data || []).join(", ") : "";
    $("template-content").value = withUnsubscribe(
      template
        ? template.content
        : '<h1>Hello {{.name}}</h1>\n<p>Welcome to the chapter.</p>',
    );
    updatePreview();
    $("template-sheet").showModal();
  }

  // Shows where values land without executing anything: the Go actions are swapped
  // for their field names and the frame is fully sandboxed.
  function fillValues(source) {
    return source
      .replace(/\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}/g, "[$1]")
      .replace(/\{\{[^}]*\}\}/g, "");
  }

  function updatePreview() {
    const subject = escapeHtml(fillValues($("template-subject").value));
    const body = fillValues($("template-content").value);
    $("template-preview").srcdoc = [
      '<!DOCTYPE html><html><head><meta charset="utf-8"><style>',
      "body{font:16px/1.5 system-ui,sans-serif;color:#202124;margin:0;padding:1rem}",
      ".subject{font-weight:600;border-bottom:1px solid #dadce0;padding-bottom:.6rem;margin-bottom:1rem}",
      "img{max-width:100%}</style></head><body>",
      `<div class="subject">${subject}</div>`,
      body,
      "</body></html>",
    ].join("");
  }

  function escapeHtml(value) {
    return value.replace(
      /[&<>"']/g,
      (character) =>
        ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character],
    );
  }

  // Appends the unsubscribe link unless the body already places it somewhere
  // of its own, so an author can move the link but not drop it.
  function withUnsubscribe(content) {
    if (/\{\{\s*\.UnsubscribeURL\s*\}\}/.test(content)) return content;
    return content.replace(/\s*$/, "") + "\n" + UNSUBSCRIBE_FOOTER;
  }

  async function saveTemplate() {
    const name = $("template-name").value.trim();
    const subject = $("template-subject").value.trim();
    if (!name || !subject || !$("template-content").value.trim()) {
      toast("Name, subject, and body are all required.", true);
      return;
    }
    // Written back so the editor and preview show exactly what was stored.
    const content = withUnsubscribe($("template-content").value);
    $("template-content").value = content;
    updatePreview();
    const data = $("template-data")
      .value.split(",")
      .map((value) => value.trim())
      .filter(Boolean);
    const body = { name, subject, data, content };
    const editing = state.editing;
    const saved = await run(() =>
      editing
        ? api(`/api/projects/${state.projectId}/templates/${editing.id}`, { method: "PUT", body })
        : api(`/api/projects/${state.projectId}/templates`, { method: "POST", body }),
    );
    if (saved === undefined) return;
    $("template-sheet").close();
    toast(editing ? "Template saved." : "Template created.");
    loadTemplates();
  }

  async function deleteTemplate(template) {
    if (!confirm(`Delete the template “${template.name}”?`)) return;
    const result = await run(() =>
      api(`/api/projects/${state.projectId}/templates/${template.id}`, { method: "DELETE" }),
    );
    if (result === undefined) return;
    toast("Template deleted.");
    loadTemplates();
  }

  /* ----------------------------------------------------------- projects */

  function renderProjects() {
    const rows = $("project-rows");
    if (!state.projects.length) {
      fill(rows, emptyRow(4, "No projects yet", "Create one above to get started."));
      return;
    }
    fill(
      rows,
      state.projects.map((project) =>
        el(
          "tr",
          {},
          el(
            "td",
            {},
            el("strong", { text: project.name }),
            project.id === state.projectId
              ? el("div", { class: "cell-faint", text: "Currently open" })
              : null,
          ),
          el("td", { class: "cell-faint", text: project.description || "—" }),
          el("td", { class: "cell-mono cell-faint", text: project.id }),
          el(
            "td",
            { class: "is-right" },
            el(
              "div",
              { class: "filters", style: "justify-content:flex-end" },
              project.id === state.projectId
                ? null
                : el("button", {
                    class: "btn btn--quiet",
                    type: "button",
                    text: "Open",
                    onclick: () => selectProject(project.id),
                  }),
              el("button", {
                class: "btn btn--quiet",
                type: "button",
                text: "Rename",
                onclick: () => editProject(project),
              }),
              el("button", {
                class: "btn btn--quiet btn--danger",
                type: "button",
                text: "Delete",
                onclick: () => deleteProject(project),
              }),
            ),
          ),
        ),
      ),
    );
    const current = state.projects.find((project) => project.id === state.projectId);
    $("keys-project").textContent = current ? current.name : "no project";
  }

  function editProject(project) {
    state.editingProject = project;
    $("project-name").value = project.name;
    $("project-description").value = project.description || "";
    $("project-submit").textContent = "Save changes";
    $("project-cancel").hidden = false;
    $("project-name").focus();
  }

  function resetProjectForm() {
    state.editingProject = null;
    $("project-form").reset();
    $("project-submit").textContent = "Create project";
    $("project-cancel").hidden = true;
  }

  async function submitProject(event) {
    event.preventDefault();
    const name = $("project-name").value.trim();
    if (!name) return;
    const body = { name, description: $("project-description").value.trim() };
    const editing = state.editingProject;
    const saved = await run(() =>
      editing
        ? api(`/api/projects/${editing.id}`, { method: "PUT", body })
        : api("/api/projects", { method: "POST", body }),
    );
    if (saved === undefined) return;
    resetProjectForm();
    toast(editing ? "Project saved." : `Created ${name}.`);
    await reloadProjects(editing ? state.projectId : saved.id);
  }

  async function deleteProject(project) {
    if (!confirm(`Delete “${project.name}”? Its templates and API keys go with it.`)) return;
    const result = await run(() => api(`/api/projects/${project.id}`, { method: "DELETE" }));
    if (result === undefined) return;
    toast(`Deleted ${project.name}.`);
    await reloadProjects(project.id === state.projectId ? "" : state.projectId);
  }

  async function reloadProjects(preferredId) {
    const projects = await run(() => api("/api/projects"));
    if (!projects) return;
    state.projects = projects;
    const exists = projects.some((project) => project.id === preferredId);
    selectProject(exists ? preferredId : (projects[0] && projects[0].id) || "", { reload: false });
    renderProjectPicker();
    renderProjects();
    loadKeys();
  }

  function selectProject(projectId, { reload = true } = {}) {
    state.projectId = projectId;
    try {
      if (projectId) localStorage.setItem(PROJECT_STORE, projectId);
      else localStorage.removeItem(PROJECT_STORE);
    } catch {
      /* remembering the project is a convenience, not a requirement */
    }
    $("key-reveal").hidden = true;
    if (!reload) return;
    renderProjectPicker();
    setView(state.view);
  }

  /* --------------------------------------------------------------- keys */

  async function loadKeys() {
    if (!requireProject()) {
      fill($("key-rows"), emptyRow(4, "No project selected", "Keys belong to a project."));
      return;
    }
    const keys = await run(() => api(`/api/projects/${state.projectId}/api-keys`));
    if (!keys) return;
    if (!keys.length) {
      fill(
        $("key-rows"),
        emptyRow(4, "No keys yet", "Create one so an app can send through this project."),
      );
      return;
    }
    fill(
      $("key-rows"),
      keys.map((key) =>
        el(
          "tr",
          {},
          el("td", { class: "cell-mono", text: key.id }),
          el("td", { class: "cell-faint", title: exactly(key.created_at), text: when(key.created_at) }),
          el(
            "td",
            {},
            key.revoked
              ? el("span", { class: "chip chip--revoked", text: "Revoked" })
              : el("span", { class: "chip chip--sent", text: "Active" }),
          ),
          el(
            "td",
            { class: "is-right" },
            key.revoked
              ? el("span", { class: "cell-faint", text: "—" })
              : el("button", {
                  class: "btn btn--quiet btn--danger",
                  type: "button",
                  text: "Revoke",
                  onclick: () => revokeKey(key),
                }),
          ),
        ),
      ),
    );
  }

  async function createKey() {
    if (!requireProject()) {
      toast("Create a project first.", true);
      return;
    }
    const key = await run(() =>
      api(`/api/projects/${state.projectId}/api-keys`, { method: "POST" }),
    );
    if (!key) return;
    $("key-value").textContent = key.key;
    $("key-reveal").hidden = false;
    loadKeys();
  }

  async function revokeKey(key) {
    if (!confirm("Revoke this key? Anything using it stops sending immediately.")) return;
    const result = await run(() =>
      api(`/api/projects/${state.projectId}/api-keys/${key.id}`, { method: "DELETE" }),
    );
    if (result === undefined) return;
    toast("Key revoked.");
    loadKeys();
  }

  /* --------------------------------------------------------------- wire */

  function wire() {
    // The project form doubles as a rename form, so it needs a way back out.
    const submit = $("project-form").querySelector('button[type="submit"]');
    submit.id = "project-submit";
    submit.after(
      el("button", {
        class: "btn btn--quiet",
        type: "button",
        id: "project-cancel",
        text: "Cancel",
        hidden: true,
        onclick: resetProjectForm,
      }),
    );

    $("gate-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const key = $("gate-key").value.trim();
      if (!key) return;
      try {
        await signIn(key);
      } catch (error) {
        state.key = "";
        showGateError(
          error instanceof ApiError && error.status === 401
            ? "That key was rejected. Check ADMIN_API_KEY on the server."
            : `Could not reach the service: ${error.message}`,
        );
      }
    });

    $("sign-out").addEventListener("click", () => signOut(""));

    $("rail-nav").addEventListener("click", (event) => {
      const link = event.target.closest(".navlink");
      if (link) setView(link.dataset.view);
    });

    for (const id of ["project-picker", "audience-project"]) {
      $(id).addEventListener("change", (event) => selectProject(event.target.value));
    }

    $("auto-refresh").addEventListener("change", () => {
      stopRefresh();
      if (state.view === "delivery") startRefresh();
    });

    $("email-status").addEventListener("change", () => loadDelivery());
    $("email-refresh").addEventListener("click", () => loadDelivery());

    $("audience-form").addEventListener("submit", addMembers);
    $("audience-search").addEventListener("input", renderAudience);
    $("audience-input").addEventListener("input", (event) => {
      const { valid, invalid } = parseAddresses(event.target.value);
      $("audience-parsed").textContent = valid.length + invalid.length
        ? `${valid.length} valid${invalid.length ? `, ${invalid.length} to fix` : ""}`
        : "";
    });

    $("send-new").addEventListener("click", openCompose);
    $("send-submit").addEventListener("click", sendCompose);
    $("send-cancel").addEventListener("click", () => $("send-sheet").close());
    $("send-template").addEventListener("change", renderComposeValues);
    $("send-audience").addEventListener("change", (event) => {
      $("send-address-field").hidden = event.target.value !== "one";
    });

    $("template-new").addEventListener("click", () => openTemplate(null));
    $("template-save").addEventListener("click", saveTemplate);
    $("template-cancel").addEventListener("click", () => $("template-sheet").close());
    for (const id of ["template-content", "template-subject"]) {
      $(id).addEventListener("input", () => {
        clearTimeout(previewTimer);
        previewTimer = setTimeout(updatePreview, 250);
      });
    }

    $("project-form").addEventListener("submit", submitProject);
    $("key-new").addEventListener("click", createKey);

    document.addEventListener("visibilitychange", () => {
      if (document.hidden) stopRefresh();
      else if (state.view === "delivery" && state.key) {
        loadDelivery({ silent: true });
        startRefresh();
      }
    });
  }

  async function boot() {
    wire();
    let saved = "";
    try {
      saved = localStorage.getItem(KEY_STORE) || "";
    } catch {
      saved = "";
    }
    if (!saved) return;
    try {
      await signIn(saved);
    } catch {
      signOut("");
    }
  }

  boot();
})();
