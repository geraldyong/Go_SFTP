"use client";

import { useEffect, useMemo, useState } from "react";

const card = {
  background: "rgba(255,255,255,0.06)",
  border: "1px solid rgba(255,255,255,0.10)",
  borderRadius: 16,
  padding: 16,
};

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
  fontWeight: 700,
};

const badge = (enabled) => ({
  display: "inline-flex",
  alignItems: "center",
  gap: 8,
  padding: "6px 10px",
  borderRadius: 999,
  border: "1px solid rgba(255,255,255,0.14)",
  background: enabled ? "rgba(34,197,94,0.18)" : "rgba(244,63,94,0.18)",
  color: enabled ? "#bbf7d0" : "#fecdd3",
  fontSize: 12,
  fontWeight: 800,
});

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

export default function UsersPage() {
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("all"); // all | enabled | disabled
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [users, setUsers] = useState([]);

  const queryString = useMemo(() => {
    const p = new URLSearchParams();
    if (q.trim()) p.set("q", q.trim());
    if (status === "enabled") p.set("disabled", "false");
    if (status === "disabled") p.set("disabled", "true");
    p.set("limit", "200");
    return p.toString();
  }, [q, status]);

  async function load() {
    setLoading(true);
    setErr("");
    try {
      const data = await apiFetch(`/api/users?${queryString}`, { method: "GET" });
      // supports either {ok,data} or raw array (defensive)
      const rows = Array.isArray(data) ? data : data?.data || [];
      setUsers(rows);
    } catch (e) {
      setErr(e.message || String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queryString]);

  async function toggleDisable(u) {
    const nextDisabled = !u.disabled;
    const msg = nextDisabled ? "Disable this user?" : "Enable this user?";
    if (!confirm(msg)) return;

    try {
      await apiFetch(`/api/users/${encodeURIComponent(u.username)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ disabled: nextDisabled }),
      });
      await load();
    } catch (e) {
      alert(e.message || String(e));
    }
  }

  return (
    <div style={{ display: "grid", gap: 14 }}>
      <div style={card}>
        <div style={{ display: "flex", alignItems: "flex-end", gap: 12, flexWrap: "wrap" }}>
          <div style={{ flex: "1 1 320px" }}>
            <div style={{ fontSize: 12, opacity: 0.8, marginBottom: 6 }}>Search username</div>
            <input
              style={input}
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="alice"
            />
          </div>

          <div style={{ width: 220 }}>
            <div style={{ fontSize: 12, opacity: 0.8, marginBottom: 6 }}>Status</div>
            <select
              style={input}
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value="all">All</option>
              <option value="enabled">Enabled</option>
              <option value="disabled">Disabled</option>
            </select>
          </div>

          <button style={btn} onClick={load}>Refresh</button>
          <a href="/create" style={{ ...btn, textDecoration: "none", display: "inline-flex", alignItems: "center" }}>
            + Create
          </a>
        </div>
      </div>

      <div style={card}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 10 }}>
          <div style={{ fontWeight: 900, fontSize: 16 }}>Users</div>
          <div style={{ fontSize: 12, opacity: 0.8 }}>
            {loading ? "Loading…" : `${users.length} user(s)`}
          </div>
        </div>

        {err ? (
          <div style={{ marginTop: 12, padding: 12, borderRadius: 12, border: "1px solid rgba(244,63,94,0.35)", background: "rgba(244,63,94,0.10)" }}>
            <div style={{ fontWeight: 900, color: "#fecdd3" }}>Error</div>
            <div style={{ marginTop: 6, opacity: 0.9 }}>{err}</div>
          </div>
        ) : null}

        <div style={{ marginTop: 12, overflowX: "auto" }}>
          <table style={{ width: "100%", borderCollapse: "separate", borderSpacing: 0 }}>
            <thead>
              <tr style={{ textAlign: "left", fontSize: 12, opacity: 0.8 }}>
                <th style={{ padding: "10px 8px" }}>Username</th>
                <th style={{ padding: "10px 8px" }}>Status</th>
                <th style={{ padding: "10px 8px" }}>Root Subdir</th>
                <th style={{ padding: "10px 8px" }}>Keys</th>
                <th style={{ padding: "10px 8px" }}>Updated</th>
                <th style={{ padding: "10px 8px" }}>Actions</th>
              </tr>
            </thead>

            <tbody>
              {users.map((u) => {
                const enabled = !u.disabled;
                return (
                  <tr key={u.username} style={{ borderTop: "1px solid rgba(255,255,255,0.08)" }}>
                    <td style={{ padding: "12px 8px", fontWeight: 800 }}>
                      <a
                        href={`/users/${encodeURIComponent(u.username)}`}
                        style={{ color: "#93c5fd", textDecoration: "none" }}
                      >
                        {u.username}
                      </a>
                    </td>
                    <td style={{ padding: "12px 8px" }}>
                      <span style={badge(enabled)}>
                        <span style={{
                          width: 8,
                          height: 8,
                          borderRadius: 999,
                          background: enabled ? "#22c55e" : "#f43f5e",
                          display: "inline-block",
                        }} />
                        {enabled ? "ENABLED" : "DISABLED"}
                      </span>
                    </td>
                    <td style={{ padding: "12px 8px", fontFamily: "ui-monospace", fontSize: 13 }}>
                      {u.rootSubdir || "-"}
                    </td>
                    <td style={{ padding: "12px 8px" }}>{u.keyCount ?? "-"}</td>
                    <td style={{ padding: "12px 8px", fontSize: 13, opacity: 0.9 }}>
                      {fmtTime(u.updatedAt)}
                    </td>
                    <td style={{ padding: "12px 8px" }}>
                      <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                        <a href={`/users/${encodeURIComponent(u.username)}`} style={{ ...btn, textDecoration: "none" }}>
                          View
                        </a>
                        <button style={btn} onClick={() => toggleDisable(u)}>
                          {u.disabled ? "Enable" : "Disable"}
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}

              {!loading && users.length === 0 ? (
                <tr>
                  <td colSpan={6} style={{ padding: 14, opacity: 0.8 }}>
                    No users found.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

