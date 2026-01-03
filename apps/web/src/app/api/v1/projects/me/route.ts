import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { authenticate, unauthorized, badRequest } from '@/lib/auth';
import { eq } from 'drizzle-orm';

// GET /api/v1/projects/me
export async function GET(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  try {
    const project = await db.query.projects.findFirst({
      where: eq(projects.id, auth.projectId),
      columns: {
        id: true,
        name: true,
        settings: true,
        createdAt: true,
        updatedAt: true,
      },
    });

    return Response.json(project);
  } catch (error) {
    console.error('Error fetching project:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}

// PATCH /api/v1/projects/me
const updateProjectSchema = z.object({
  name: z.string().max(100).optional(),
  settings: z.object({
    retentionDays: z.number().int().min(1).max(365).optional(),
    webhookUrl: z.string().url().optional(),
  }).optional(),
});

export async function PATCH(request: NextRequest) {
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  try {
    const body = await request.json();
    const result = updateProjectSchema.safeParse(body);

    if (!result.success) {
      return badRequest(result.error.message);
    }

    const { name, settings } = result.data;

    await db.update(projects)
      .set({
        ...(name && { name }),
        ...(settings && { settings }),
        updatedAt: new Date(),
      })
      .where(eq(projects.id, auth.projectId));

    return new Response(null, { status: 204 });
  } catch (error) {
    console.error('Error updating project:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
