import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { eq } from 'drizzle-orm';
import { NextRequest } from 'next/server';

export interface AuthContext {
  projectId: string;
  project: {
    id: string;
    name: string;
    ownerEmail: string;
  };
}

/**
 * Authenticate request using Bearer token
 * Returns project context or null if unauthorized
 */
export async function authenticate(request: NextRequest): Promise<AuthContext | null> {
  const authHeader = request.headers.get('authorization');

  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return null;
  }

  const apiKey = authHeader.slice(7); // Remove 'Bearer '

  if (!apiKey.startsWith('le_')) {
    return null;
  }

  try {
    const project = await db.query.projects.findFirst({
      where: eq(projects.apiKey, apiKey),
      columns: {
        id: true,
        name: true,
        ownerEmail: true,
      },
    });

    if (!project) {
      return null;
    }

    return {
      projectId: project.id,
      project,
    };
  } catch {
    return null;
  }
}

/**
 * Create unauthorized response
 */
export function unauthorized(message = 'Unauthorized') {
  return Response.json({ error: message }, { status: 401 });
}

/**
 * Create bad request response
 */
export function badRequest(message: string) {
  return Response.json({ error: message }, { status: 400 });
}

/**
 * Create not found response
 */
export function notFound(message = 'Not found') {
  return Response.json({ error: message }, { status: 404 });
}
