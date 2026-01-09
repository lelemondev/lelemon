/**
 * Query production database via Tailscale
 * Usage: npx tsx scripts/db-query.ts
 */
import { config } from 'dotenv';
import { resolve } from 'path';
import pg from 'pg';

// Load .env from scripts folder
config({ path: resolve(__dirname, '.env') });

const { Client } = pg;

async function main() {
  const dbUrl = process.env.DATABASE_URL;
  if (!dbUrl) {
    console.error('Error: DATABASE_URL not found in scripts/.env');
    process.exit(1);
  }

  const client = new Client({ connectionString: dbUrl });

  try {
    await client.connect();
    console.log('Connected to production database\n');

    // Get latest trace (by session containing 'conv_')
    const traceResult = await client.query(`
      SELECT t.id, t.session_id, t.created_at
      FROM traces t
      WHERE t.session_id LIKE 'conv_%'
      ORDER BY t.created_at DESC
      LIMIT 1
    `);

    if (traceResult.rows.length === 0) {
      console.log('No sales-conversation traces found');
      return;
    }

    const trace = traceResult.rows[0];
    console.log('=== LATEST TRACE ===');
    console.log(`ID: ${trace.id}`);
    console.log(`Session: ${trace.session_id}`);
    console.log(`Created: ${trace.created_at}\n`);

    // Get all spans for this trace
    const spansResult = await client.query(`
      SELECT
        s.id,
        s.trace_id,
        s.parent_span_id,
        s.type,
        s.name,
        s.model,
        s.input_tokens,
        s.output_tokens,
        s.duration_ms,
        s.status,
        LEFT(s.input::text, 300) as input_preview,
        LEFT(s.output::text, 300) as output_preview,
        s.started_at
      FROM spans s
      WHERE s.trace_id = $1
         OR s.id = 'c1809780-eba0-4ddc-a721-d59bfe8519a2'
      ORDER BY s.started_at ASC
    `, [trace.id]);

    console.log(`=== SPANS (${spansResult.rows.length} total) ===\n`);

    for (const span of spansResult.rows) {
      console.log(`--- ${span.type.toUpperCase()}: ${span.name} ---`);
      console.log(`  ID: ${span.id}`);
      console.log(`  TraceID: ${span.trace_id}`);
      console.log(`  Parent: ${span.parent_span_id || '(root)'}`);
      console.log(`  Model: ${span.model || '-'}`);
      console.log(`  Tokens: ${span.input_tokens || 0} in / ${span.output_tokens || 0} out`);
      console.log(`  Duration: ${span.duration_ms}ms`);
      console.log(`  Input: ${span.input_preview || '-'}`);
      console.log(`  Output: ${span.output_preview || '-'}`);
      console.log('');
    }

  } catch (error) {
    console.error('Error:', error);
  } finally {
    await client.end();
  }
}

main();
