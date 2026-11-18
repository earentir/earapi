import { AUTH_KEY } from "./constants.js";
import { state } from "./state.js";

let renderFn = () => {};

export function setRender(fn) {
  renderFn = fn;
}

export function loadAuth() {
  try {
    const a = JSON.parse(localStorage.getItem(AUTH_KEY) || "null");
    if (a?.user && a?.token) {
      state.user = a.user;
      state.token = a.token;
    }
  } catch {
    /* ignore */
  }
}

export function saveAuth() {
  localStorage.setItem(AUTH_KEY, JSON.stringify({ user: state.user, token: state.token }));
}

export async function api(path, opts = {}) {
  const headers = {
    Accept: "application/json",
    "X-User": state.user,
    "X-API-Key": state.token,
    ...(opts.body ? { "Content-Type": "application/json" } : {}),
    ...(opts.headers || {}),
  };
  const res = await fetch(`/health/v1${path}`, { ...opts, headers });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = { error: text };
  }
  if (!res.ok) {
    let msg = data?.error || res.statusText || "request failed";
    if (typeof msg === "string" && (msg.includes("<html") || msg.includes("<!DOCTYPE"))) {
      msg = res.status === 504
        ? "Health API timed out (504). Try again; if it keeps happening, redeploy earapi with the latest build."
        : `Health API error (${res.status})`;
    } else if (typeof msg === "string" && msg.length > 240) {
      msg = `${msg.slice(0, 240)}…`;
    }
    const err = new Error(msg);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

export function toast(msg) {
  state.toast = msg;
  renderFn();
  setTimeout(() => {
    if (state.toast === msg) {
      state.toast = "";
      renderFn();
    }
  }, 2400);
}
