import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { eq } from 'drizzle-orm';
import { createClient } from '@/lib/supabase/server';
import { randomBytes, createHash } from 'crypto';

// GET /api/v1/projects - List all projects for the current user
export async function GET() {
  try {
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.getUser();

    if (!user?.email) {
      return Response.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const userProjects = await db.query.projects.findMany({
      where: eq(projects.ownerEmail, user.email),
      columns: {
        id: true,
        name: true,
        apiKey: true, // We will transform to hint
        createdAt: true,
        updatedAt: true,
      },
      orderBy: (projects, { desc }) => [desc(projects.createdAt)],
    });

    // Transform apiKey to a hint (first 12 chars + ...)
    const projectsWithHint = userProjects.map(({ apiKey, ...rest }) => ({
      ...rest,
      apiKeyHint: apiKey ? apiKey.slice(0, 12) + '...' : null,
    }));

    return Response.json(projectsWithHint);
  } catch (error) {
    console.error('Error fetching projects:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}

// PATCH /api/v1/projects - Update a project
const updateProjectSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
});

export async function PATCH(request: NextRequest) {
  try {
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.getUser();

    if (!user?.email) {
      return Response.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const result = updateProjectSchema.safeParse(body);

    if (!result.success) {
      return Response.json({ error: result.error.message }, { status: 400 });
    }

    const { id, name } = result.data;

    // Verify ownership
    const project = await db.query.projects.findFirst({
      where: eq(projects.id, id),
      columns: { ownerEmail: true },
    });

    if (!project || project.ownerEmail !== user.email) {
      return Response.json({ error: 'Project not found' }, { status: 404 });
    }

    await db.update(projects)
      .set({ name, updatedAt: new Date() })
      .where(eq(projects.id, id));

    return Response.json({ id, name });
  } catch (error) {
    console.error('Error updating project:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}

// POST /api/v1/projects - Create a new project
const createProjectSchema = z.object({
  name: z.string().min(1).max(100),
});

export async function POST(request: NextRequest) {
  try {
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.getUser();

    if (!user?.email) {
      return Response.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const result = createProjectSchema.safeParse(body);

    if (!result.success) {
      return Response.json({ error: result.error.message }, { status: 400 });
    }

    const { name } = result.data;

    // Generate API key
    const apiKey = `le_${randomBytes(24).toString('base64url')}`;

    // Hash the API key for secure storage
    const apiKeyHash = createHash('sha256').update(apiKey).digest('hex');

    const [project] = await db.insert(projects).values({
      name,
      ownerEmail: user.email,
      apiKey,
      apiKeyHash,
    }).returning({
      id: projects.id,
      name: projects.name,
      apiKey: projects.apiKey,
      createdAt: projects.createdAt,
    });

    return Response.json(project, { status: 201 });
  } catch (error) {
    console.error('Error creating project:', error);
    return Response.json({ error: 'Internal server error' }, { status: 500 });
  }
}
