export const metadata = {
  title: "SFTP Admin",
  description: "Admin portal for managing SFTP users stored in Vault",
};

const shell = {
  fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial",
  background: "#0b0f19",
  minHeight: "100vh",
  color: "#e5e7eb",
};

const container = {
  maxWidth: 980,
  margin: "0 auto",
  padding: "22px 18px 40px",
};

const header = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: 16,
  padding: "14px 14px",
  background: "rgba(255,255,255,0.06)",
  border: "1px solid rgba(255,255,255,0.10)",
  borderRadius: 16,
};

const brand = { display: "flex", alignItems: "center", gap: 10 };
const dot = {
  width: 10,
  height: 10,
  borderRadius: 999,
  background: "#22c55e",
  boxShadow: "0 0 0 4px rgba(34,197,94,0.15)",
};

const nav = { display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" };
const linkStyle = {
  display: "inline-block",
  padding: "8px 10px",
  borderRadius: 12,
  textDecoration: "none",
  color: "#e5e7eb",
  border: "1px solid rgba(255,255,255,0.12)",
  background: "rgba(255,255,255,0.04)",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body style={shell}>
        <div style={container}>
          <div style={header}>
            <div style={brand}>
              <div style={dot} />
              <div>
                <div style={{ fontWeight: 800, fontSize: 16 }}>SFTP Admin</div>
                <div style={{ opacity: 0.8, fontSize: 12 }}>
                  Manage users stored in Vault KV (via admin-api)
                </div>
              </div>
            </div>

            <nav style={nav}>
              <a href="/users" style={linkStyle}>Users</a>
              <a href="/create" style={linkStyle}>Create User</a>
              <a href="/system" style={linkStyle}>System</a>
            </nav>
          </div>

          <div style={{ marginTop: 18 }}>{children}</div>
        </div>
      </body>
    </html>
  );
}

