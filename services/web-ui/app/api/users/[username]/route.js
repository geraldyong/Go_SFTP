export async function GET(_req, { params }) {
  const base = process.env.ADMIN_API_BASE_URL || "http://admin-api:8080";
  const res = await fetch(`${base}/api/v1/users/${encodeURIComponent(params.username)}`, {
    method: "GET",
    headers: { "Accept": "application/json" },
    cache: "no-store",
  });

  const text = await res.text();
  return new Response(text, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("content-type") || "application/json" },
  });
}

export async function PUT(req, { params }) {
  const base = process.env.ADMIN_API_BASE_URL || "http://admin-api:8080";
  const body = await req.json();

  const res = await fetch(`${base}/api/v1/users/${encodeURIComponent(params.username)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", "Accept": "application/json" },
    body: JSON.stringify(body),
  });

  const text = await res.text();
  return new Response(text, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("content-type") || "application/json" },
  });
}

export async function PATCH(req, { params }) {
  const base = process.env.ADMIN_API_BASE_URL || "http://admin-api:8080";
  const body = await req.json();

  const res = await fetch(`${base}/api/v1/users/${encodeURIComponent(params.username)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", "Accept": "application/json" },
    body: JSON.stringify(body),
  });

  const text = await res.text();
  return new Response(text, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("content-type") || "application/json" },
  });
}

export async function DELETE(_req, { params }) {
  const base = process.env.ADMIN_API_BASE_URL || "http://admin-api:8080";
  const res = await fetch(`${base}/api/v1/users/${encodeURIComponent(params.username)}`, {
    method: "DELETE",
    headers: { "Accept": "application/json" },
  });

  const text = await res.text();
  return new Response(text, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("content-type") || "application/json" },
  });
}

