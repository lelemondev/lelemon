import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { eq } from 'drizzle-orm';
import { NextRequest } from 'next/server';
import { createHash } from 'crypto';
import { rateLimit, rateLimitResponse } from './rate-limit';

export interface AuthContext {
  projectId: string;
  project: {
    id: string;
    name: string;
    ownerEmail: string;
  };
}

/**
 * Hash an API key using SHA-256
 */
function hashApiKey(apiKey: string): string {
  return createHash('sha256').update(apiKey).digest('hex');
}

/**
 * Get client identifier for rate limiting (IP or forwarded IP)
 */
function getClientId(request: NextRequest): string {
  return (
    request.headers.get('x-forwarded-for')?.split(',')[0]?.trim() ||
    request.headers.get('x-real-ip') ||
    'unknown'
  );
}

/**
 * Check rate limit for unauthenticated requests (by IP)
 * Stricter limit: 20 requests per minute
 */
export function checkIpRateLimit(request: NextRequest): Response | null {
  const clientId = getClientId(request);
  const result = rateLimit(`ip:${clientId}`, 20, 60000);
  
  if (!result.success) {
    return rateLimitResponse(result);
  }
  return null;
}

/**
 * Check rate limit for authenticated requests (by project)
 * Higher limit: 100 requests per minute
 */
export function checkProjectRateLimit(projectId: string): Response | null {
  const result = rateLimit(`project:${projectId}`, 100, 60000);
  
  if (!result.success) {
    return rateLimitResponse(result);
  }
  return null;
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
    // Hash the API key and compare against stored hash (not plaintext)
    const apiKeyHash = hashApiKey(apiKey);

    const project = await db.query.projects.findFirst({
      where: eq(projects.apiKeyHash, apiKeyHash),
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
