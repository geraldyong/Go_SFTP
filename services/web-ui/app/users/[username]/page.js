"use client";

import { useEffect, useState } from "react";

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

function fmtTime(iso) {
  if (!iso) return "-";
  try {
    const d = new Date(iso);
    return d.toLocaleString();
  } catch {
    return iso;
  }
}

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

function joinKeys(keys) {
  if (!Array.isArray(keys)) return "";
  return keys.join("\n");
}

function splitKeys(text) {
  return (text || "")
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}

export default function UserDetailPage({ params }) {
  const username = params.username;

  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");

  const [disabled, setDisabled] = useState(false);
  const [rootSubdir, setRootSubdir] = useState("");
  const [keysText, setKeysText] = useState("");
  const [updatedAt, setUpdatedAt] = useState("");

  async function load() {
    setLoading(true);
    setErr("");
    setMsg("");
    try {
      const out = await apiFetch(`/api/users/${encodeURIComponent(username)}`, { method: "GET" });
      const u = out?.data || out;

      setDisabled(!!u.disabled);
      setRootSubdir(u.rootSubdir || "");
      if (Array.isArray(u.publicKeys)) setKeysText(joinKeys(u.publicKeys, "\n"));
      else setKeysText("");
      setUpdatedAt(u.updatedAt || "");
    } catch (e) {
      setErr(e.message || String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [username]);

  async function onSave() {
    setMsg("");
    try {
      const payload = {
        disabled,
        rootSubdir: rootSubdir.trim() || undefined,
        publicKeys: splitKeys(keysText),
      };

      await apiFetch(`/api/users/${encodeURIComponent(username)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, ...payload }),
      });

      setMsg("Saved.");
      await load();
    } catch (e) {
      setMsg(`Error: ${e.message || String(e)}`);
    }
  }

  async function onDelete() {
    setMsg("");
    if (!confirm(`Delete user "${username}"? This removes the Vault record.`)) return;

    try {
      await apiFetch(`/api/users/${encodeURIComponent(username)}`, { method: "DELETE" });
      window.location.href = "/users";
    } catch (e) {
      setMsg(`Error: ${e.message || String(e)}`);
    }
  }

  return (
    <div style={{ display: "grid", gap: 14 }}>
      <div style={card}>
        <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <div>
            <div style={{ fontWeight: 900, fontSize: 16 }}>User: {username}</div>
            <div style={{ opacity: 0.85, fontSize: 13, marginTop: 6 }}>
              Updated: {fmtTime(updatedAt)}
            </div>
          </div>

          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            <a href="/users" style={{ ...btn, textDecoration: "none" }}>Back</a>
            <button style={btn} onClick={onSave} disabled={loading}>Save</button>
            <button
              style={{ ...btn, borderColor: "rgba(244,63,94,0.45)", background: "rgba(244,63,94,0.12)" }}
              onClick={onDelete}
              disabled={loading}
            >
              Delete
            </button>
          </div>
        </div>
      </div>

      {loading ? (
        <div style={card}>Loading…</div>
      ) : err ? (
        <div style={{ ...card, border: "1px solid rgba(244,63,94,0.35)", background: "rgba(244,63,94,0.10)" }}>
          <div style={{ fontWeight: 900, color: "#fecdd3" }}>Error</div>
          <div style={{ marginTop: 6, opacity: 0.95 }}>{err}</div>
        </div>
      ) : (
        <div style={{ ...card, display: "grid", gap: 12 }}>
          <label style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <input type="checkbox" checked={disabled} onChange={(e) => setDisabled(e.target.checked)} />
            <span style={{ fontWeight: 900 }}>Disabled</span>
          </label>

          <div>
            <div style={label}>Root subdir</div>
            <input
              style={input}
              value={rootSubdir}
              onChange={(e) => setRootSubdir(e.target.value)}
              placeholder="alice"
            />
          </div>

          <div>
            <div style={label}>Public keys (one per line)</div>
            <textarea
              style={{ ...input, fontFamily: "ui-monospace", minHeight: 180 }}
              value={keysText}
              onChange={(e) => setKeysText(e.target.value)}
              placeholder="ssh-ed25519 AAAA... alice@laptop"
            />
          </div>

          {msg ? (
            <div style={{
              marginTop: 6,
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
        </div>
      )}
    </div>
  );
}

