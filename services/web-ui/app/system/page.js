const card = {
  background: "rgba(255,255,255,0.06)",
  border: "1px solid rgba(255,255,255,0.10)",
  borderRadius: 16,
  padding: 16,
};

const linkStyle = {
  display: "inline-block",
  padding: "10px 12px",
  borderRadius: 12,
  textDecoration: "none",
  color: "#e5e7eb",
  border: "1px solid rgba(255,255,255,0.12)",
  background: "rgba(255,255,255,0.04)",
};

export default function SystemPage() {
  return (
    <div style={{ display: "grid", gap: 14 }}>
      <div style={card}>
        <div style={{ fontWeight: 900, fontSize: 16 }}>System</div>
        <div style={{ marginTop: 6, opacity: 0.85, fontSize: 13 }}>
          Quick links for checking service health.
        </div>
      </div>

      <div style={{ ...card, display: "grid", gap: 10 }}>
        <a style={linkStyle} href="/api/healthz" target="_blank" rel="noreferrer">
          Web UI health proxy
        </a>
        <a style={linkStyle} href="/api/admin-healthz" target="_blank" rel="noreferrer">
          Admin API health (via proxy)
        </a>
      </div>

      <div style={{ ...card, opacity: 0.8, fontSize: 13 }}>
        <div style={{ fontWeight: 900, marginBottom: 6 }}>Notes</div>
        <div>
          In docker-compose, the admin-api and sftp-server metrics endpoints are usually reachable
          from the host via mapped ports (see docker-compose.yml).
        </div>
      </div>
    </div>
  );
}

