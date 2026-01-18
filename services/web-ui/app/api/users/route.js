export async function POST(req) {
  const body = await req.json();
  const base = process.env.ADMIN_API_BASE_URL || "http://admin-api:8080";

  const res = await fetch(`${base}/api/v1/users`, {
    method: "POST",
    headers: {"Content-Type":"application/json"},
    body: JSON.stringify(body)
  });

  const text = await res.text();
  return new Response(text, {
    status: res.status,
    headers: {"Content-Type": res.headers.get("content-type") || "text/plain"}
  });
}
