import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { authenticate, unauthorized } from '@/lib/auth';
import { eq } from 'drizzle-orm';
import { createHash, randomBytes } from 'crypto';

function generateApiKey(): string {
  const random = randomBytes(24).toString('hex');
  return `le_${random}`;
}

function hashApiKey(apiKey: string): string {
  return createHash('sha256').update(apiKey).digest('hex');
}

// POST /api/v1/projects/api-key - Rotate API key
export async function POST(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  try {
    const newApiKey = generateApiKey();
    const newApiKeyHash = hashApiKey(newApiKey);

    await db.update(projects)
      .set({
        apiKey: newApiKey,
        apiKeyHash: newApiKeyHash,
        updatedAt: new Date(),
      })
      .where(eq(projects.id, auth.projectId));

    return Response.json({ apiKey: newApiKey });
  } catch (error) {
    console.error('Error rotating API key:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
