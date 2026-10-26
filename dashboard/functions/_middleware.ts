/**
 * Quiet Doors (October 2026): a discovery-shaped path that this site does not
 * publish answers 404 instead of the app shell. App routes keep the Pages
 * HTML fallback, and every real file passes through untouched.
 */

interface DoorContext {
  request: Request;
  next(): Promise<Response>;
}

const DISCOVERY_SHAPE = /^\/\.well-known(?:\/|$)|\.(?:txt|json|xml)$/i;

export function isDiscoveryShaped(pathname: string): boolean {
  return DISCOVERY_SHAPE.test(pathname);
}

function isAppShell(response: Response): boolean {
  const type = response.headers.get("Content-Type")?.toLowerCase() ?? "";
  return response.status === 200 && type.startsWith("text/html");
}

export function missingDoor(method: string): Response {
  const body = "404 Not Found\nNothing is published at this path.\nThe site starts at /.\n";
  return new Response(method === "HEAD" ? null : body, {
    status: 404,
    headers: {
      "Cache-Control": "public, max-age=300",
      "Content-Type": "text/plain; charset=utf-8",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

export async function onRequest(context: DoorContext): Promise<Response> {
  const { request } = context;
  if (request.method !== "GET" && request.method !== "HEAD") return context.next();
  const { pathname } = new URL(request.url);
  if (pathname.startsWith("/api/") || !isDiscoveryShaped(pathname)) return context.next();
  const response = await context.next();
  return isAppShell(response) ? missingDoor(request.method) : response;
}
