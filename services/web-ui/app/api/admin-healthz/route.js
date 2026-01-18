export async function GET() {
  const base = process.env.ADMIN_API_BASE_URL || "http://admin-api:8080";
  const res = await fetch(`${base}/healthz`, { method: "GET", cache: "no-store" });
  const text = await res.text();
  return new Response(text, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("content-type") || "text/plain" },
  });
}

