import { NextRequest } from 'next/server';
import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { eq, and } from 'drizzle-orm';
import { createHash, randomBytes } from 'crypto';
import { createClient } from '@/lib/supabase/server';
import { z } from 'zod';

function generateApiKey(): string {
  const random = randomBytes(24).toString('hex');
  return `le_${random}`;
}

function hashApiKey(apiKey: string): string {
  return createHash('sha256').update(apiKey).digest('hex');
}

const rotateKeySchema = z.object({
  projectId: z.string().uuid(),
});

// POST /api/v1/projects/api-key - Rotate API key (dashboard auth)
export async function POST(request: NextRequest) {
  try {
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.getUser();

    if (!user?.email) {
      return Response.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const result = rotateKeySchema.safeParse(body);

    if (!result.success) {
      return Response.json({ error: 'Project ID is required' }, { status: 400 });
    }

    const { projectId } = result.data;

    // Verify user owns this project
    const project = await db.query.projects.findFirst({
      where: and(
        eq(projects.id, projectId),
        eq(projects.ownerEmail, user.email)
      ),
    });

    if (!project) {
      return Response.json({ error: 'Project not found' }, { status: 404 });
    }

    const newApiKey = generateApiKey();
    const newApiKeyHash = hashApiKey(newApiKey);

    await db.update(projects)
      .set({
        apiKey: newApiKey,
        apiKeyHash: newApiKeyHash,
        updatedAt: new Date(),
      })
      .where(eq(projects.id, projectId));

    return Response.json({ apiKey: newApiKey });
  } catch (error) {
    console.error('Error rotating API key:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
