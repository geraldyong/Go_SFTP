"use client";

import { useState } from "react";

const card = {
  background: "rgba(255,255,255,0.06)",
  border: "1px solid rgba(255,255,255,0.10)",
  borderRadius: 16,
  padding: 16,
};

const label = { fontSize: 12, opacity: 0.8, marginBottom: 6 };

const input = {
  width: "100%",
  padding: 10,
  borderRadius: 12,
  border: "1px solid rgba(255,255,255,0.14)",
  background: "rgba(0,0,0,0.25)",
  color: "#e5e7eb",
  outline: "none",
};

const btn = {
  padding: "10px 12px",
  borderRadius: 12,
  border: "1px solid rgba(255,255,255,0.14)",
  background: "rgba(255,255,255,0.06)",
  color: "#e5e7eb",
  cursor: "pointer",
  fontWeight: 800,
};

async function apiFetch(url, init) {
  const res = await fetch(url, init);
  const ct = res.headers.get("content-type") || "";
  const body = ct.includes("application/json") ? await res.json() : await res.text();
  if (!res.ok) {
    const msg =
      typeof body === "string"
        ? body
        : body?.error?.message || body?.message || JSON.stringify(body);
    throw new Error(msg);
  }
  return body;
}

function splitKeys(text) {
  const lines = (text || "")
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
  return lines;
}

export default function CreateUserPage() {
  const [username, setUsername] = useState("");
  const [rootSubdir, setRootSubdir] = useState("");
  const [disabled, setDisabled] = useState(false);
  const [keysText, setKeysText] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");

  async function onSubmit(e) {
    e.preventDefault();
    setBusy(true);
    setMsg("");

    try {
      const payload = {
        username: username.trim(),
        rootSubdir: (rootSubdir || "").trim() || undefined,
        disabled,
        publicKeys: splitKeys(keysText),
      };

      const out = await apiFetch("/api/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const u = out?.data || out;
      setMsg(`Saved user: ${u?.username || payload.username}`);
    } catch (e2) {
      setMsg(`Error: ${e2.message || String(e2)}`);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ display: "grid", gap: 14 }}>
      <div style={card}>
        <div style={{ fontWeight: 900, fontSize: 16 }}>Create User</div>
        <div style={{ marginTop: 6, opacity: 0.85, fontSize: 13 }}>
          Creates or updates a user record in Vault (KV v2) for SFTP public-key authentication.
        </div>
      </div>

      <form onSubmit={onSubmit} style={{ ...card, display: "grid", gap: 12 }}>
        <div>
          <div style={label}>Username</div>
          <input
            style={input}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="alice"
            required
          />
        </div>

        <div>
          <div style={label}>Root subdir (optional)</div>
          <input
            style={input}
            value={rootSubdir}
            onChange={(e) => setRootSubdir(e.target.value)}
            placeholder="alice"
          />
          <div style={{ fontSize: 12, opacity: 0.75, marginTop: 6 }}>
            If empty, server can default to username.
          </div>
        </div>

        <label style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <input
            type="checkbox"
            checked={disabled}
            onChange={(e) => setDisabled(e.target.checked)}
          />
          <span style={{ fontWeight: 800 }}>Disabled</span>
        </label>

        <div>
          <div style={label}>Public keys (one per line)</div>
          <textarea
            style={{ ...input, fontFamily: "ui-monospace", minHeight: 140 }}
            value={keysText}
            onChange={(e) => setKeysText(e.target.value)}
            placeholder={`ssh-ed25519 AAAA... alice@laptop\nssh-ed25519 AAAA... alice@desktop`}
            required
          />
        </div>

        <div style={{ display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
          <button style={btn} type="submit" disabled={busy}>
            {busy ? "Saving…" : "Save User"}
          </button>
          <a href="/users" style={{ ...btn, textDecoration: "none" }}>Back to Users</a>
        </div>

        {msg ? (
          <div style={{
            marginTop: 8,
            padding: 12,
            borderRadius: 12,
            border: "1px solid rgba(255,255,255,0.12)",
            background: "rgba(0,0,0,0.18)",
            fontSize: 13,
            opacity: 0.95,
            whiteSpace: "pre-wrap",
          }}>
            {msg}
          </div>
        ) : null}
      </form>
    </div>
  );
}

