import { createClient } from '@/lib/supabase/server';
import { NextResponse } from 'next/server';
import { db } from '@/db/client';
import { projects } from '@/db/schema';
import { eq } from 'drizzle-orm';
import { randomBytes, createHash } from 'crypto';

export async function GET(request: Request) {
  const requestUrl = new URL(request.url);
  const code = requestUrl.searchParams.get('code');

  // Use env var for production, fallback to origin
  const baseUrl = process.env.NEXT_PUBLIC_APP_URL || requestUrl.origin;

  if (code) {
    const supabase = await createClient();
    const { data: { user } } = await supabase.auth.exchangeCodeForSession(code);

    // Check if user is new (has no projects)
    if (user?.email) {
      const existingProjects = await db.query.projects.findMany({
        where: eq(projects.ownerEmail, user.email),
        columns: { id: true },
        limit: 1,
      });

      // First-time user: create default project and redirect to config
      if (existingProjects.length === 0) {
        const apiKey = `le_${randomBytes(24).toString('base64url')}`;
        const apiKeyHash = createHash('sha256').update(apiKey).digest('hex');

        await db.insert(projects).values({
          name: 'My Project',
          ownerEmail: user.email,
          apiKey,
          apiKeyHash,
        });

        // Redirect to config with welcome flag to show API key
        return NextResponse.redirect(`${baseUrl}/dashboard/config?welcome=true&key=${encodeURIComponent(apiKey)}`);
      }
    }
  }

  return NextResponse.redirect(`${baseUrl}/dashboard`);
}
