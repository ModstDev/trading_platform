const API = "http://localhost:8080";

const state = {
  token: localStorage.getItem("access_token"),
  user: null,
  account: null,
  instruments: [],
  orders: [],
  positions: [],
  executions: [],
  marketPrices: {},
  selectedInstrumentId: null,
  side: "BUY",
  marketPriceTimer: null
};

const $ = id => document.getElementById(id);

function money(value, currency = "") {
  if (value === null || value === undefined || value === "") return "—";
  const n = Number(value);
  if (!Number.isFinite(n)) return String(value);
  return `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 })}${currency ? ` ${currency}` : ""}`;
}

function qty(value) {
  if (value === null || value === undefined || value === "") return "—";
  const n = Number(value);
  return Number.isFinite(n) ? n.toLocaleString(undefined, { maximumFractionDigits: 8 }) : String(value);
}

function date(value) {
  if (!value) return "—";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString();
}

function esc(value) {
  const s = String(value ?? "");
  return s.replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;" }[c]));
}

function toast(message, type = "") {
  const el = $("toast");
  el.textContent = message;
  el.className = `toast show ${type}`;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => el.className = "toast", 3200);
}

function setAuthError(message, type = "error") {
  $("auth-error").textContent = message || "";
  $("auth-error").className = `form-message ${type}`;
}

function unwrapNullable(value) {
  if (value === null || value === undefined) return null;
  if (typeof value !== "object") return value;
  if ("Valid" in value) return value.Valid ? value.String : null;
  if ("valid" in value) return value.valid ? (value.string ?? value.String) : null;
  if ("String" in value) return value.String;
  if ("string" in value) return value.string;
  return null;
}

async function api(path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (options.body !== undefined) headers.set("Content-Type", "application/json");
  if (state.token) headers.set("Authorization", `Bearer ${state.token}`);

  let response;
  try {
    response = await fetch(`${API}${path}`, { ...options, headers });
  } catch {
    setConnection(false);
    throw new Error(`Cannot connect to API at ${API}. Make sure the Go API is running on port 8080.`);
  }

  setConnection(true);
  const text = await response.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = text; }

  if (response.status === 401) {
    state.token = null;
    localStorage.removeItem("access_token");
    if (!$("auth-screen").classList.contains("hidden")) {
      throw new Error("Invalid or expired session.");
    }
    showAuth();
    throw new Error("Your session expired. Please sign in again.");
  }

  if (!response.ok) {
    const message = typeof data === "string"
      ? data
      : data?.error || data?.message || `Request failed (${response.status})`;
    throw new Error(message);
  }

  return data;
}

function setConnection(ok) {
  const el = $("connection");
  el.className = `connection ${ok ? "ok" : "bad"}`;
  el.innerHTML = `<i></i>${ok ? "API connected" : "API unavailable"}`;
}

function showAuth() {
  $("auth-screen").classList.remove("hidden");
  $("app").classList.add("hidden");
}

function showApp() {
  $("auth-screen").classList.add("hidden");
  $("app").classList.remove("hidden");
}

async function loadSession() {
  if (!state.token) {
    showAuth();
    return;
  }

  try {
    showApp();
    state.user = await api("/me");
    $("user-email").textContent = state.user.email;
    await loadAll();
    startMarketPricePolling();
  } catch (e) {
    showAuth();
    if (state.token) setAuthError(e.message);
  }
}

async function loadAll() {
  await Promise.all([
    loadAccount(),
    loadInstruments(),
    loadOrders(),
    loadPositions(),
    loadExecutions(),
    loadMarketPrices()
  ]);
}

async function loadAccount() {
  state.account = await api("/account");

  const balance = Number(state.account.balance || 0);
  const reserved = Number(state.account.reserved_balance || 0);
  const available = balance - reserved;
  const currency = state.account.currency || "";

  $("stat-balance").textContent = money(balance);
  $("stat-currency").textContent = currency || "—";
  $("stat-reserved").textContent = money(reserved);
  $("stat-available").textContent = money(available);

  renderAccount();
}

async function loadInstruments() {
  state.instruments = await api("/instruments") || [];
  renderInstruments();
  renderInstrumentSelect();

  if (!state.selectedInstrumentId && state.instruments.length) {
    selectInstrument(state.instruments[0].id);
  } else if (state.selectedInstrumentId) {
    const exists = state.instruments.some(i => String(i.id) === String(state.selectedInstrumentId));
    if (!exists && state.instruments.length) selectInstrument(state.instruments[0].id);
    else renderSelectedInstrument();
  }
}

async function loadOrders() {
  state.orders = await api("/orders") || [];
  renderOrders();
}

async function loadPositions() {
  state.positions = await api("/positions") || [];
  renderPositionSummary();
  renderPositions();
}

function renderPositionSummary() {
  const box = $("position-summary");

  if (!state.positions.length) {
    box.innerHTML = `<div class="position-summary-empty">No holdings yet</div>`;
    return;
  }

  box.innerHTML = state.positions.map(p => `
    <div class="position-summary-row">
      <div>
        <strong>${esc(instrumentSymbol(p.instrument_id))}</strong>
        <span>${esc(instrumentName(p.instrument_id))}</span>
      </div>
      <div class="position-summary-qty">
        <strong>${qty(p.quantity)}</strong>
        <span>owned</span>
      </div>
    </div>
  `).join("");
}

async function loadExecutions() {
  state.executions = await api("/executions") || [];
  renderExecutions();
}

async function loadMarketPrices() {
  const prices = await api("/market-prices") || [];

  state.marketPrices = {};

  for (const price of prices) {
    if (!price?.symbol) continue;

    state.marketPrices[String(price.symbol).toUpperCase()] = {
      price: price.price,
      timestamp: price.timestamp
    };
  }

  renderInstruments();
  renderSelectedInstrument();
  updatePreview();
}

function getMarketPrice(instrument) {
  if (!instrument?.symbol) return null;

  return state.marketPrices[String(instrument.symbol).toUpperCase()] || null;
}

function selectedInstrument() {
  return state.instruments.find(i => String(i.id) === String(state.selectedInstrumentId)) || null;
}

function selectInstrument(id) {
  state.selectedInstrumentId = id;
  $("order-instrument").value = id;
  renderInstruments();
  renderSelectedInstrument();
}

function renderInstruments() {
  const box = $("instruments");
  if (!state.instruments.length) {
    box.innerHTML = `<div class="empty">No instruments available.</div>`;
    return;
  }

  box.innerHTML = state.instruments.map(i => {
    const marketPrice = getMarketPrice(i);

    return `
    <button class="instrument ${String(i.id) === String(state.selectedInstrumentId) ? "active" : ""}" data-id="${esc(i.id)}">
      <div>
        <span class="symbol">${esc(i.symbol)}</span>
        <span class="name">${esc(i.name)}</span>
      </div>
      <div>
        <span class="type">${esc(i.type || "")}</span>
        <span class="market-price">
          ${marketPrice ? money(marketPrice.price) : "—"}
        </span>
      </div>
    </button>
  `;
  }).join("");

  box.querySelectorAll(".instrument").forEach(btn => {
    btn.addEventListener("click", () => selectInstrument(btn.dataset.id));
  });
}

function renderInstrumentSelect() {
  const select = $("order-instrument");
  select.innerHTML = state.instruments.map(i =>
    `<option value="${esc(i.id)}">${esc(i.symbol)} — ${esc(i.name)}</option>`
  ).join("");

  if (state.selectedInstrumentId) select.value = state.selectedInstrumentId;
}

function renderSelectedInstrument() {
  const i = selectedInstrument();
  if (!i) return;

  $("instrument-symbol").textContent = i.symbol || "—";

  const marketPrice = getMarketPrice(i);

  $("instrument-name").textContent =
    marketPrice
      ? `${i.name || "—"} · ${money(marketPrice.price)}`
      : (i.name || "—");
  $("info-name").textContent = i.name || "—";
  $("info-symbol").textContent = i.symbol || "—";
  $("info-type").textContent = i.type || "—";
  $("info-currency").textContent = i.currency || "—";
  $("info-id").textContent = i.id || "—";
}

function renderOrders() {
  const box = $("orders-view");

  if (!state.orders.length) {
    box.innerHTML = `<div class="empty">You have no orders yet.</div>`;
    return;
  }

  box.innerHTML = `
    <table>
      <thead><tr>
        <th>Instrument</th><th>Side</th><th>Type</th><th>Quantity</th>
        <th>Filled</th><th>Price</th><th>Status</th><th>Created</th><th></th>
      </tr></thead>
      <tbody>
        ${state.orders.map(o => {
    const price = unwrapNullable(o.price);
    const status = String(o.status || "").toUpperCase();
    const side = String(o.side || "").toUpperCase();
    const canCancel = status === "PENDING" || status === "PARTIALLY_FILLED";
    return `<tr>
            <td><strong>${esc(instrumentSymbol(o.instrument_id))}</strong></td>
            <td class="${side === "BUY" ? "positive" : "negative"}">${esc(o.side)}</td>
            <td>${esc(o.type)}</td>
            <td>${qty(o.quantity)}</td>
            <td>${qty(o.filled_quantity)}</td>
            <td>${price === null ? "Market" : money(price)}</td>
            <td><span class="status-badge">${esc(o.status)}</span></td>
            <td>${esc(date(o.created_at))}</td>
            <td>${canCancel ? `<button class="cancel" data-id="${esc(o.id)}">Cancel</button>` : ""}</td>
          </tr>`;
  }).join("")}
      </tbody>
    </table>
  `;

  box.querySelectorAll(".cancel").forEach(btn => {
    btn.addEventListener("click", () => cancelOrder(btn.dataset.id));
  });
}

function renderPositions() {
  const box = $("positions-view");
  if (!state.positions.length) {
    box.innerHTML = `<div class="empty">You have no positions yet.</div>`;
    return;
  }

  box.innerHTML = `
    <table>
      <thead><tr><th>Instrument</th><th>Quantity</th><th>Reserved</th><th>Average price</th><th>Created</th></tr></thead>
      <tbody>
        ${state.positions.map(p => `<tr>
          <td><strong>${esc(instrumentSymbol(p.instrument_id))}</strong></td>
          <td>${qty(p.quantity)}</td>
          <td>${qty(p.reserved_quantity)}</td>
          <td>${money(p.average_price)}</td>
          <td>${esc(date(p.created_at))}</td>
        </tr>`).join("")}
      </tbody>
    </table>
  `;
}

function renderExecutions() {
  const box = $("executions-view");
  if (!state.executions.length) {
    box.innerHTML = `<div class="empty">No executions yet.</div>`;
    return;
  }

  box.innerHTML = `
    <table>
      <thead><tr><th>Instrument</th><th>Quantity</th><th>Price</th><th>Executed</th><th>Order ID</th></tr></thead>
      <tbody>
        ${state.executions.map(e => `<tr>
          <td><strong>${esc(instrumentSymbol(e.instrument_id))}</strong></td>
          <td>${qty(e.quantity)}</td>
          <td>${money(e.price)}</td>
          <td>${esc(date(e.executed_at))}</td>
          <td class="mono">${esc(e.order_id)}</td>
        </tr>`).join("")}
      </tbody>
    </table>
  `;
}

function renderAccount() {
  const a = state.account;
  if (!a) return;

  const balance = Number(a.balance || 0);
  const reserved = Number(a.reserved_balance || 0);
  const available = balance - reserved;

  $("account-view").innerHTML = `
    <table>
      <tbody>
        <tr><th>Account ID</th><td class="mono">${esc(a.id)}</td></tr>
        <tr><th>User ID</th><td class="mono">${esc(a.user_id)}</td></tr>
        <tr><th>Balance</th><td>${money(balance, a.currency)}</td></tr>
        <tr><th>Reserved balance</th><td>${money(reserved, a.currency)}</td></tr>
        <tr><th>Available balance</th><td>${money(available, a.currency)}</td></tr>
        <tr><th>Currency</th><td>${esc(a.currency)}</td></tr>
      </tbody>
    </table>
  `;
}

function instrumentSymbol(id) {
  return state.instruments.find(i => String(i.id) === String(id))?.symbol || id || "—";
}

function instrumentName(id) {
  return state.instruments.find(i => String(i.id) === String(id))?.name || "Unknown instrument";
}

async function cancelOrder(id) {
  if (!confirm("Cancel this order?")) return;

  try {
    await api(`/orders/${encodeURIComponent(id)}`, { method: "DELETE" });
    toast("Order cancelled.", "success");
    await Promise.all([loadOrders(), loadAccount(), loadPositions()]);
  } catch (e) {
    toast(e.message, "error");
  }
}

async function submitOrder() {
  const instrumentId = $("order-instrument").value;
  const type = $("order-type").value;
  const quantity = $("quantity").value;
  const price = $("price").value;
  const maxCost = $("max-cost").value;

  if (!instrumentId) return showOrderError("Select an instrument.");
  if (!quantity || Number(quantity) <= 0) return showOrderError("Quantity must be greater than 0.");
  if (type === "LIMIT" && (!price || Number(price) <= 0)) return showOrderError("Limit price must be greater than 0.");
  if (type === "MARKET" && maxCost && Number(maxCost) <= 0) return showOrderError("Maximum cost must be greater than 0.");

  const body = {
    instrument_id: instrumentId,
    side: state.side,
    type,
    quantity: quantity
  };

  if (type === "LIMIT") body.price = price;
  if (type === "MARKET" && maxCost) body.max_cost = maxCost;

  const button = $("submit-order");
  button.disabled = true;

  try {
    await api("/orders", { method: "POST", body: JSON.stringify(body) });
    $("quantity").value = "";
    $("price").value = "";
    $("max-cost").value = "";
    $("order-message").textContent = "Order created successfully.";
    $("order-message").className = "form-message success";
    toast("Order created.", "success");
    await Promise.all([loadOrders(), loadAccount(), loadPositions()]);
  } catch (e) {
    showOrderError(e.message);
  } finally {
    button.disabled = false;
  }
}

function showOrderError(message) {
  $("order-message").textContent = message;
  $("order-message").className = "form-message error";
}

function updateOrderForm() {
  const market = $("order-type").value === "MARKET";
  $("price-group").classList.toggle("hidden", market);
  $("max-cost-group").classList.toggle("hidden", !market);
  updatePreview();
}

function updatePreview() {
  const q = Number($("quantity").value);
  const p = Number($("price").value);
  const market = $("order-type").value === "MARKET";

  if (q <= 0) {
    $("order-preview").querySelector("strong").textContent = "—";
    return;
  }

  if (!market && p > 0) {
    $("order-preview").querySelector("strong").textContent = money(q * p);
    return;
  }

  if (market) {
    const marketPrice = getMarketPrice(selectedInstrument());

    $("order-preview").querySelector("strong").textContent =
      marketPrice
        ? `≈ ${money(q * Number(marketPrice.price))}`
        : "Waiting for market price...";

    return;
  }

  $("order-preview").querySelector("strong").textContent = "—";
}

function setSide(side) {
  state.side = side;
  const buy = side === "BUY";

  $("buy-button").className = `segment ${buy ? "active-buy" : ""}`;
  $("sell-button").className = `segment ${!buy ? "active-sell" : ""}`;
  $("selected-side-label").textContent = side;
  $("selected-side-label").className = `order-side-label ${buy ? "buy-text" : "sell-text"}`;
  $("submit-order").className = `button ${buy ? "buy" : "sell"} wide`;
  $("submit-order").textContent = `Place ${buy ? "buy" : "sell"} order`;
}

function startMarketPricePolling() {
  if (state.marketPriceTimer) {
    clearInterval(state.marketPriceTimer);
  }

  state.marketPriceTimer = setInterval(async () => {
    if (!state.token || $("app").classList.contains("hidden")) {
      return;
    }

    try {
      await loadMarketPrices();
    } catch (e) {
      console.warn("Failed to refresh market prices:", e.message);
    }
  }, 5000);
}

async function login(e) {
  e.preventDefault();
  setAuthError("");

  try {
    const data = await api("/login", {
      method: "POST",
      body: JSON.stringify({
        email: $("login-email").value.trim(),
        password: $("login-password").value
      })
    });

    if (!data?.access_token) throw new Error("Login succeeded but no access token was returned.");

    state.token = data.access_token;
    localStorage.setItem("access_token", state.token);
    state.user = await api("/me");
    $("user-email").textContent = state.user.email;
    showApp();
    await loadAll();
    startMarketPricePolling();
  } catch (e) {
    setAuthError(e.message);
  }
}

async function register(e) {
  e.preventDefault();
  setAuthError("");

  const email = $("register-email").value.trim();
  const password = $("register-password").value;

  if (password.length < 8) {
    setAuthError("Password must contain at least 8 characters.");
    return;
  }

  try {
    await api("/users", {
      method: "POST",
      body: JSON.stringify({ email, password })
    });

    $("login-email").value = email;
    $("login-password").value = "";
    switchAuthTab("login");
    setAuthError("Account created. Sign in to continue.", "success");
  } catch (e) {
    setAuthError(e.message);
  }
}

function logout() {
  state.token = null;
  localStorage.removeItem("access_token");
  state.user = null;
  if (state.marketPriceTimer) {
    clearInterval(state.marketPriceTimer);
    state.marketPriceTimer = null;
  }

  state.marketPrices = {};
  showAuth();
}

function switchAuthTab(tab) {
  document.querySelectorAll(".auth-tab").forEach(b => b.classList.toggle("active", b.dataset.tab === tab));
  $("login-form").classList.toggle("hidden", tab !== "login");
  $("register-form").classList.toggle("hidden", tab !== "register");
  if (tab === "register") setAuthError("");
}

$("login-form").addEventListener("submit", login);
$("register-form").addEventListener("submit", register);
$("logout").addEventListener("click", logout);
$("buy-button").addEventListener("click", () => setSide("BUY"));
$("sell-button").addEventListener("click", () => setSide("SELL"));
$("quick-buy").addEventListener("click", () => { setSide("BUY"); $("quantity").focus(); });
$("quick-sell").addEventListener("click", () => { setSide("SELL"); $("quantity").focus(); });
$("submit-order").addEventListener("click", submitOrder);
$("order-type").addEventListener("change", updateOrderForm);
$("quantity").addEventListener("input", updatePreview);
$("price").addEventListener("input", updatePreview);
$("order-instrument").addEventListener("change", e => selectInstrument(e.target.value));
$("refresh-all").addEventListener("click", async () => {
  try { await loadAll(); toast("Data refreshed.", "success"); }
  catch (e) { toast(e.message, "error"); }
});

document.querySelectorAll(".auth-tab").forEach(b => b.addEventListener("click", () => switchAuthTab(b.dataset.tab)));

document.querySelectorAll(".data-tab").forEach(tab => {
  tab.addEventListener("click", () => {
    document.querySelectorAll(".data-tab").forEach(t => t.classList.toggle("active", t === tab));
    document.querySelectorAll(".data-view").forEach(v => v.classList.add("hidden"));
    $(`${tab.dataset.view}-view`).classList.remove("hidden");
  });
});

setSide("BUY");
updateOrderForm();
setConnection(false);
loadSession();
