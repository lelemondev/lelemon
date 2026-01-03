import {
  pgTable,
  pgEnum,
  uuid,
  varchar,
  text,
  timestamp,
  integer,
  jsonb,
  decimal,
  index,
} from 'drizzle-orm/pg-core';

// ============================================
// ENUMS
// ============================================

export const traceStatusEnum = pgEnum('trace_status', [
  'active',
  'completed',
  'error',
]);

export const spanTypeEnum = pgEnum('span_type', [
  'llm',
  'tool',
  'retrieval',
  'custom',
]);

export const spanStatusEnum = pgEnum('span_status', [
  'pending',
  'success',
  'error',
]);

// ============================================
// PROJECTS (Multi-tenant)
// ============================================

export const projects = pgTable('projects', {
  id: uuid('id').primaryKey().defaultRandom(),
  name: varchar('name', { length: 100 }).notNull(),

  // API Key (stored as hash for security)
  apiKey: varchar('api_key', { length: 64 }).unique().notNull(),
  apiKeyHash: varchar('api_key_hash', { length: 64 }).notNull(),

  // Owner
  ownerEmail: varchar('owner_email', { length: 255 }).notNull(),

  // Settings
  settings: jsonb('settings').$type<{
    retentionDays?: number;
    webhookUrl?: string;
  }>().default({}),

  // Timestamps
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// ============================================
// TRACES (One per conversation/session)
// ============================================

export const traces = pgTable('traces', {
  id: uuid('id').primaryKey().defaultRandom(),
  projectId: uuid('project_id').references(() => projects.id, { onDelete: 'cascade' }).notNull(),

  // Flexible identification
  sessionId: varchar('session_id', { length: 100 }),
  userId: varchar('user_id', { length: 100 }),

  // Custom metadata and tags
  metadata: jsonb('metadata').$type<Record<string, unknown>>().default({}),
  tags: text('tags').array(),

  // Aggregated metrics
  totalTokens: integer('total_tokens').default(0).notNull(),
  totalCostUsd: decimal('total_cost_usd', { precision: 10, scale: 6 }).default('0'),
  totalDurationMs: integer('total_duration_ms').default(0).notNull(),
  totalSpans: integer('total_spans').default(0).notNull(),

  // Status
  status: traceStatusEnum('status').default('active').notNull(),

  // Timestamps
  createdAt: timestamp('created_at').defaultNow().notNull(),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
}, (table) => [
  index('traces_project_created_idx').on(table.projectId, table.createdAt),
  index('traces_session_idx').on(table.projectId, table.sessionId),
  index('traces_user_idx').on(table.projectId, table.userId),
]);

// ============================================
// SPANS (Each operation within a trace)
// ============================================

export const spans = pgTable('spans', {
  id: uuid('id').primaryKey().defaultRandom(),
  traceId: uuid('trace_id').references(() => traces.id, { onDelete: 'cascade' }).notNull(),
  parentSpanId: uuid('parent_span_id'),

  // Type and name
  type: spanTypeEnum('type').notNull(),
  name: varchar('name', { length: 100 }).notNull(),

  // Input/Output
  input: jsonb('input').$type<unknown>(),
  output: jsonb('output').$type<unknown>(),

  // Metrics
  inputTokens: integer('input_tokens'),
  outputTokens: integer('output_tokens'),
  costUsd: decimal('cost_usd', { precision: 10, scale: 6 }),
  durationMs: integer('duration_ms'),

  // Status
  status: spanStatusEnum('status').default('pending').notNull(),
  errorMessage: text('error_message'),

  // LLM-specific metadata
  model: varchar('model', { length: 50 }),
  provider: varchar('provider', { length: 20 }),

  // Custom metadata
  metadata: jsonb('metadata').$type<Record<string, unknown>>().default({}),

  // Timestamps
  startedAt: timestamp('started_at').defaultNow().notNull(),
  endedAt: timestamp('ended_at'),
}, (table) => [
  index('spans_trace_idx').on(table.traceId, table.startedAt),
]);

// ============================================
// TYPE EXPORTS
// ============================================

export type Project = typeof projects.$inferSelect;
export type NewProject = typeof projects.$inferInsert;

export type Trace = typeof traces.$inferSelect;
export type NewTrace = typeof traces.$inferInsert;

export type Span = typeof spans.$inferSelect;
export type NewSpan = typeof spans.$inferInsert;
