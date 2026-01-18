"use client";

export default function UserForm() {
  async function onSubmit(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const username = form.get("username");
    const key = form.get("publicKey");

    const res = await fetch("/api/users", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ username, publicKeys: [key] })
    });

    const text = await res.text();
    alert(res.ok ? "Saved in Vault." : text);
  }

  return (
    <form onSubmit={onSubmit} style={{marginTop: 16, display:"grid", gap: 12}}>
      <label>
        <div>Username</div>
        <input name="username" required style={inputStyle} placeholder="alice" />
      </label>

      <label>
        <div>Public key (authorized_keys line)</div>
        <textarea
          name="publicKey"
          required
          rows={4}
          style={{...inputStyle, fontFamily:"ui-monospace"}}
          placeholder="ssh-ed25519 AAAA... alice@laptop"
        />
      </label>

      <button type="submit" style={btnStyle}>Create / Update User</button>
    </form>
  );
}

const inputStyle = {
  width: "100%",
  padding: 10,
  borderRadius: 10,
  border: "1px solid #ccc",
  marginTop: 6
};

const btnStyle = {
  padding: "10px 14px",
  borderRadius: 12,
  border: "1px solid #111",
  background: "#111",
  color: "white",
  cursor: "pointer",
  width: "fit-content"
};

