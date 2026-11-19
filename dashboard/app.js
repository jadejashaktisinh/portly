const TOKEN_KEY = "portly.token";
const USER_KEY = "portly.user";

const state = {
  token: localStorage.getItem(TOKEN_KEY),
  user: readUser(),
  mode: "login",
  events: [],
  selectedId: null,
  timer: null,
};

const $ = (id) => document.getElementById(id);

function readUser() {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || "null");
  } catch {
    return null;
  }
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function setMode(mode) {
  state.mode = mode;
  $("tab-login").classList.toggle("active", mode === "login");
  $("tab-signup").classList.toggle("active", mode === "signup");
  $("slug-field").classList.toggle("hidden", mode !== "signup");
  $("slug").required = mode === "signup";
  $("auth-submit").textContent = mode === "login" ? "Sign in" : "Create account";
  $("auth-error").classList.add("hidden");
}

function showAuth() {
  $("auth-view").classList.remove("hidden");
  $("dashboard-view").classList.add("hidden");
  if (state.timer) clearInterval(state.timer);
  state.timer = null;
}

function showDashboard() {
  $("auth-view").classList.add("hidden");
  $("dashboard-view").classList.remove("hidden");
  $("user-label").textContent = state.user?.email ?? "";
  if (!state.timer) state.timer = setInterval(loadEvents, 30000);
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body) headers["Content-Type"] = "application/json";
  if (state.token && options.auth !== false) {
    headers.Authorization = `Bearer ${state.token}`;
  }

  const response = await fetch(path, { ...options, headers });
  let data;
  try {
    data = await response.json();
  } catch {
    data = {};
  }

  if (response.status === 401 && options.auth !== false) {
    logout();
    throw new Error("Session expired. Please sign in again.");
  }
  if (!response.ok) {
    throw new Error(data.error || `Request failed (${response.status})`);
  }
  return data;
}

async function handleAuth(event) {
  event.preventDefault();
  const errorBox = $("auth-error");
  errorBox.classList.add("hidden");

  const payload = {
    email: $("email").value.trim(),
    password: $("password").value,
  };
  if (state.mode === "signup") payload.slug = $("slug").value.trim();

  try {
    const data = await api(state.mode === "login" ? "/auth/login" : "/auth/signup", {
      method: "POST",
      body: JSON.stringify(payload),
      auth: false,
    });
    state.token = data.token;
    state.user = data.user;
    localStorage.setItem(TOKEN_KEY, data.token);
    localStorage.setItem(USER_KEY, JSON.stringify(data.user));
    $("auth-form").reset();
    showDashboard();
    await loadEvents();
  } catch (error) {
    errorBox.textContent = error.message;
    errorBox.classList.remove("hidden");
  }
}

function logout() {
  state.token = null;
  state.user = null;
  state.events = [];
  state.selectedId = null;
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  showAuth();
}

function statusClass(code) {
  if (code >= 500) return "bad";
  if (code >= 400) return "warn";
  return "ok";
}

function formatDate(value) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function renderList() {
  const list = $("event-list");
  list.replaceChildren();
  $("list-count").textContent = state.events.length ? `${state.events.length} shown` : "";

  if (!state.events.length) {
    list.appendChild(el("p", "empty", "No deliveries yet. Send a webhook to your relay URL."));
    return;
  }

  for (const event of state.events) {
    const row = el("button", "event-row");
    row.type = "button";
    if (event.id === state.selectedId) row.classList.add("selected");
    row.appendChild(el("span", "method", event.method));

    const route = el("span", "route");
    route.appendChild(el("strong", "", event.path));
    route.appendChild(el("span", "", `${event.host}${event.slug ? ` · ${event.slug}` : ""}`));
    row.appendChild(route);

    const code = event.statusCode ?? 0;
    const badge = el("span", `badge ${statusClass(code)}`, code ? String(code) : "pending");
    row.appendChild(badge);

    row.addEventListener("click", () => {
      state.selectedId = event.id;
      renderList();
      renderDetail();
    });
    list.appendChild(row);
  }
}

function addHeaderRow(container, name, value) {
  const row = el("div", "header-row");
  row.appendChild(el("span", "", name));
  row.appendChild(el("span", "", value));
  container.appendChild(row);
}

function renderDetail() {
  const detail = $("event-detail");
  detail.replaceChildren();
  const event = state.events.find((item) => item.id === state.selectedId);
  $("copy-id").classList.toggle("hidden", !event);

  if (!event) {
    detail.appendChild(el("p", "empty", "Select a delivery to inspect it."));
    return;
  }

  const grid = el("dl", "detail-grid");
  const fields = [
    ["Request ID", event.requestId],
    ["Method", event.method],
    ["Host", event.host],
    ["Slug", event.slug || "—"],
    ["Path", event.path],
    ["Status", event.statusCode ?? "pending"],
    ["Duration", `${event.durationMs ?? 0} ms`],
    ["Received", formatDate(event.createdAt)],
    ["Error", event.error || "—"],
  ];
  for (const [label, value] of fields) {
    grid.appendChild(el("dt", "", label));
    grid.appendChild(el("dd", "", String(value)));
  }
  detail.appendChild(grid);

  const headersSection = el("div", "detail-section");
  headersSection.appendChild(el("h3", "", "Request headers"));
  for (const [name, values] of Object.entries(event.headers || {})) {
    addHeaderRow(headersSection, name, (values || []).join(", "));
  }
  detail.appendChild(headersSection);

  const bodySection = el("div", "detail-section");
  bodySection.appendChild(el("h3", "", "Request body"));
  bodySection.appendChild(el("pre", "", event.body || "(empty)"));
  detail.appendChild(bodySection);
}

async function loadEvents() {
  if (!state.token) return;
  const errorBox = $("dashboard-error");
  try {
    const data = await api("/events?limit=50");
    state.events = data.events || [];
    $("total-count").textContent = String(data.total ?? state.events.length);
    $("last-delivery").textContent = state.events.length ? formatDate(state.events[0].createdAt) : "—";
    if (!state.events.some((item) => item.id === state.selectedId)) state.selectedId = null;
    errorBox.classList.add("hidden");
    renderList();
    renderDetail();
  } catch (error) {
    if (!state.token) return;
    errorBox.textContent = error.message;
    errorBox.classList.remove("hidden");
  }
}

$("tab-login").addEventListener("click", () => setMode("login"));
$("tab-signup").addEventListener("click", () => setMode("signup"));
$("auth-form").addEventListener("submit", handleAuth);
$("logout").addEventListener("click", logout);
$("refresh").addEventListener("click", loadEvents);
$("copy-id").addEventListener("click", async () => {
  const event = state.events.find((item) => item.id === state.selectedId);
  if (!event || !navigator.clipboard) return;
  await navigator.clipboard.writeText(event.requestId);
  $("copy-id").textContent = "Copied";
  setTimeout(() => ($("copy-id").textContent = "Copy request ID"), 1500);
});

if (state.token && state.user) {
  showDashboard();
  loadEvents();
} else {
  setMode("login");
  showAuth();
}
