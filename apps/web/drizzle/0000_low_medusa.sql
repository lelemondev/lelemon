CREATE TYPE "public"."span_status" AS ENUM('pending', 'success', 'error');--> statement-breakpoint
CREATE TYPE "public"."span_type" AS ENUM('llm', 'tool', 'retrieval', 'custom');--> statement-breakpoint
CREATE TYPE "public"."trace_status" AS ENUM('active', 'completed', 'error');--> statement-breakpoint
CREATE TABLE "projects" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"name" varchar(100) NOT NULL,
	"api_key" varchar(64) NOT NULL,
	"api_key_hash" varchar(64) NOT NULL,
	"owner_email" varchar(255) NOT NULL,
	"settings" jsonb DEFAULT '{}'::jsonb,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "projects_api_key_unique" UNIQUE("api_key")
);
--> statement-breakpoint
CREATE TABLE "spans" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"trace_id" uuid NOT NULL,
	"parent_span_id" uuid,
	"type" "span_type" NOT NULL,
	"name" varchar(100) NOT NULL,
	"input" jsonb,
	"output" jsonb,
	"input_tokens" integer,
	"output_tokens" integer,
	"cost_usd" numeric(10, 6),
	"duration_ms" integer,
	"status" "span_status" DEFAULT 'pending' NOT NULL,
	"error_message" text,
	"model" varchar(50),
	"provider" varchar(20),
	"metadata" jsonb DEFAULT '{}'::jsonb,
	"started_at" timestamp DEFAULT now() NOT NULL,
	"ended_at" timestamp
);
--> statement-breakpoint
CREATE TABLE "traces" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"session_id" varchar(100),
	"user_id" varchar(100),
	"metadata" jsonb DEFAULT '{}'::jsonb,
	"tags" text[],
	"total_tokens" integer DEFAULT 0 NOT NULL,
	"total_cost_usd" numeric(10, 6) DEFAULT '0',
	"total_duration_ms" integer DEFAULT 0 NOT NULL,
	"total_spans" integer DEFAULT 0 NOT NULL,
	"status" "trace_status" DEFAULT 'active' NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
ALTER TABLE "spans" ADD CONSTRAINT "spans_trace_id_traces_id_fk" FOREIGN KEY ("trace_id") REFERENCES "public"."traces"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "traces" ADD CONSTRAINT "traces_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "spans_trace_idx" ON "spans" USING btree ("trace_id","started_at");--> statement-breakpoint
CREATE INDEX "traces_project_created_idx" ON "traces" USING btree ("project_id","created_at");--> statement-breakpoint
CREATE INDEX "traces_session_idx" ON "traces" USING btree ("project_id","session_id");--> statement-breakpoint
CREATE INDEX "traces_user_idx" ON "traces" USING btree ("project_id","user_id");