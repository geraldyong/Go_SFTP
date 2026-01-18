import UserForm from "./UserForm";

export default function Home() {
  return (
    <main style={{fontFamily:"ui-sans-serif", padding: 24, maxWidth: 760}}>
      <h1 style={{fontSize: 28, fontWeight: 700}}>SFTP Admin</h1>
      <p style={{marginTop: 8, opacity: 0.8}}>
        Create/update Vault user records for SFTP public-key authentication.
      </p>
      <UserForm />
    </main>
  );
}
