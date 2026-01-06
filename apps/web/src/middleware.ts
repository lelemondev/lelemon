import { NextResponse, type NextRequest } from 'next/server';

export async function middleware(request: NextRequest) {
  // Auth is handled client-side via JWT token in localStorage
  // The AuthProvider in auth-context.tsx handles redirecting unauthenticated users
  // This middleware only ensures proper routing
  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
};
