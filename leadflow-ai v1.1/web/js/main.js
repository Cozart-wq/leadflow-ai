const TOKEN_STORAGE_KEY = "leadflow_token";

const authPanel = document.getElementById("auth-panel");
const dashboard = document.getElementById("dashboard");
const userBar = document.getElementById("user-bar");
const userEmail = document.getElementById("user-email");
const logoutBtn = document.getElementById("logout-btn");

const tabLogin = document.getElementById("tab-login");
const tabRegister = document.getElementById("tab-register");
const loginForm = document.getElementById("login-form");
const registerForm = document.getElementById("register-form");
const authStatus = document.getElementById("auth-status");

const taskForm = document.getElementById("task-form");
const queryInput = document.getElementById("query-input");
const taskStatus = document.getElementById("task-status");
const tasksTableBody = document.querySelector("#tasks-table tbody");
const leadsTableBody = document.querySelector("#leads-table tbody");

let refreshTimer = null;

// api() централизует все запросы к бэкенду: подставляет Authorization,
// если токен есть, и при 401 сбрасывает сессию — так каждому вызывающему
// коду не нужно самому решать, что делать с истёкшим токеном.
async function api(path, options = {}) {
    const token = getToken();
    const headers = Object.assign({}, options.headers);
    if (token) {
        headers["Authorization"] = "Bearer " + token;
    }

    const res = await fetch("/api/v1" + path, Object.assign({}, options, { headers }));

    if (res.status === 401) {
        clearSession();
        throw new Error("Сессия истекла, войдите снова");
    }
    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
    }
    if (res.status === 204) return null;
    return res.json();
}

function getToken() {
    return localStorage.getItem(TOKEN_STORAGE_KEY);
}

function setSession(token, email) {
    localStorage.setItem(TOKEN_STORAGE_KEY, token);
    showDashboard(email);
}

function clearSession() {
    localStorage.removeItem(TOKEN_STORAGE_KEY);
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }
    dashboard.classList.add("hidden");
    userBar.classList.add("hidden");
    authPanel.classList.remove("hidden");
}

function showDashboard(email) {
    authPanel.classList.add("hidden");
    dashboard.classList.remove("hidden");
    userBar.classList.remove("hidden");
    userEmail.textContent = email || "";
    refreshAll();
    if (!refreshTimer) {
        refreshTimer = setInterval(refreshAll, 5000); // поллинг прогресса пайплайна
    }
}

tabLogin.addEventListener("click", () => switchAuthTab("login"));
tabRegister.addEventListener("click", () => switchAuthTab("register"));

function switchAuthTab(which) {
    const isLogin = which === "login";
    tabLogin.classList.toggle("active", isLogin);
    tabRegister.classList.toggle("active", !isLogin);
    loginForm.classList.toggle("hidden", !isLogin);
    registerForm.classList.toggle("hidden", isLogin);
    authStatus.textContent = "";
}

loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    authStatus.textContent = "";
    try {
        const res = await fetch("/api/v1/auth/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                email: document.getElementById("login-email").value,
                password: document.getElementById("login-password").value,
            }),
        });
        const body = await res.json();
        if (!res.ok) throw new Error(body.error || "Не удалось войти");
        setSession(body.token, body.user.email);
    } catch (err) {
        authStatus.textContent = "Ошибка: " + err.message;
    }
});

registerForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    authStatus.textContent = "";
    try {
        const res = await fetch("/api/v1/auth/register", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                name: document.getElementById("register-name").value,
                email: document.getElementById("register-email").value,
                password: document.getElementById("register-password").value,
            }),
        });
        const body = await res.json();
        if (!res.ok) throw new Error(body.error || "Не удалось зарегистрироваться");
        setSession(body.token, body.user.email);
    } catch (err) {
        authStatus.textContent = "Ошибка: " + err.message;
    }
});

logoutBtn.addEventListener("click", () => clearSession());

async function loadTasks() {
    const tasks = await api("/tasks");
    tasksTableBody.innerHTML = "";
    (tasks || []).forEach(t => {
        const tr = document.createElement("tr");
        tr.innerHTML = `
            <td>${escapeHtml(t.query)}</td>
            <td><span class="badge badge-${t.status}">${t.status}</span></td>
            <td>${new Date(t.created_at).toLocaleString("ru-RU")}</td>
        `;
        tasksTableBody.appendChild(tr);
    });
}

async function loadLeads() {
    const leads = await api("/leads");
    leadsTableBody.innerHTML = "";
    (leads || []).forEach(l => {
        const tr = document.createElement("tr");
        tr.innerHTML = `
            <td>${escapeHtml(l.company_name)}</td>
            <td>${l.website ? `<a href="${escapeHtml(l.website)}" target="_blank" rel="noopener">${escapeHtml(l.website)}</a>` : "—"}</td>
            <td>${escapeHtml(l.email || "—")}</td>
            <td>${escapeHtml(l.phone || "—")}</td>
            <td>${l.quality_score ?? "—"}</td>
            <td>${escapeHtml(l.ai_recommendation || "—")}</td>
            <td><button data-id="${l.id}" class="delete-btn">Удалить</button></td>
        `;
        leadsTableBody.appendChild(tr);
    });

    document.querySelectorAll(".delete-btn[data-id]").forEach(btn => {
        btn.addEventListener("click", async () => {
            await api(`/leads/${btn.dataset.id}`, { method: "DELETE" });
            await loadLeads();
        });
    });
}

taskForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    taskStatus.textContent = "Запускаем поиск...";
    try {
        await api("/tasks", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ query: queryInput.value }),
        });
        queryInput.value = "";
        taskStatus.textContent = "Задача создана, обработка идёт в фоне.";
        await loadTasks();
    } catch (err) {
        taskStatus.textContent = "Ошибка: " + err.message;
    }
});

function escapeHtml(str) {
    const div = document.createElement("div");
    div.textContent = str ?? "";
    return div.innerHTML;
}

async function refreshAll() {
    try {
        await Promise.all([loadTasks(), loadLeads()]);
    } catch (err) {
        taskStatus.textContent = "Ошибка: " + err.message;
    }
}

document.addEventListener("DOMContentLoaded", async () => {
    const token = getToken();
    if (!token) {
        clearSession();
        return;
    }
    try {
        const user = await api("/auth/me");
        showDashboard(user.email);
    } catch {
        clearSession();
    }
});
