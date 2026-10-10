-- Migration 091: Fix all Supabase Security Advisor warnings and errors
-- Fixes: 216 errors (RLS not enabled) + 83 warnings (function search_path mutable)
-- Safe: Go backend connects as postgres/service_role which bypasses RLS
-- Note: No explicit BEGIN/COMMIT — the Go migration runner handles transactions
-- Date: 2026-02-24

-- ============================================================================
-- PART 1: Fix Function Search Path Mutable (83 warnings)
-- Sets search_path = public on ALL public schema functions that don't have it
-- ============================================================================

DO $$
DECLARE
  func_record RECORD;
  alter_sql TEXT;
BEGIN
  FOR func_record IN
    SELECT
      n.nspname AS schema_name,
      p.proname AS function_name,
      pg_get_function_identity_arguments(p.oid) AS args,
      p.oid
    FROM pg_proc p
    JOIN pg_namespace n ON p.pronamespace = n.oid
    WHERE n.nspname = 'public'
      AND p.prokind IN ('f', 'p')
      AND (
        p.proconfig IS NULL
        OR NOT EXISTS (
          SELECT 1 FROM unnest(p.proconfig) c WHERE c LIKE 'search_path=%'
        )
      )
  LOOP
    alter_sql := format(
      'ALTER FUNCTION %I.%I(%s) SET search_path = public',
      func_record.schema_name,
      func_record.function_name,
      func_record.args
    );
    BEGIN
      EXECUTE alter_sql;
      RAISE NOTICE 'Fixed search_path: %.%(%)', func_record.schema_name, func_record.function_name, func_record.args;
    EXCEPTION WHEN OTHERS THEN
      RAISE WARNING 'Could not fix %.%(%): %', func_record.schema_name, func_record.function_name, func_record.args, SQLERRM;
    END;
  END LOOP;
END $$;

-- ============================================================================
-- PART 2: Enable Row Level Security on ALL public tables (216 errors)
-- Go backend uses postgres/service_role which bypasses RLS automatically.
-- This blocks direct access via Supabase PostgREST (anon/authenticated roles).
-- ============================================================================

DO $$
DECLARE
  tbl RECORD;
BEGIN
  FOR tbl IN
    SELECT schemaname, tablename
    FROM pg_tables
    WHERE schemaname = 'public'
      AND NOT rowsecurity
    ORDER BY tablename
  LOOP
    EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', tbl.schemaname, tbl.tablename);
    RAISE NOTICE 'Enabled RLS: %.%', tbl.schemaname, tbl.tablename;
  END LOOP;
END $$;

-- ============================================================================
-- PART 3: Force RLS for table owners (defense-in-depth)
-- By default, table owners bypass RLS. FORCE ensures even owners are subject
-- to policies when accessing via non-superuser connections.
-- ============================================================================

DO $$
DECLARE
  tbl RECORD;
BEGIN
  FOR tbl IN
    SELECT schemaname, tablename
    FROM pg_tables
    WHERE schemaname = 'public'
  LOOP
    EXECUTE format('ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY', tbl.schemaname, tbl.tablename);
  END LOOP;
END $$;

-- ============================================================================
-- PART 4: Create service_role bypass policies
-- Ensures the service_role (used by Go backend via Supabase) has full access
-- even with FORCE ROW LEVEL SECURITY enabled.
-- ============================================================================

DO $$
DECLARE
  tbl RECORD;
  policy_name TEXT;
BEGIN
  FOR tbl IN
    SELECT schemaname, tablename
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY tablename
  LOOP
    policy_name := 'service_role_bypass_' || tbl.tablename;

    -- Drop if exists (idempotent)
    EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', policy_name, tbl.schemaname, tbl.tablename);

    -- Create permissive policy for service_role
    EXECUTE format(
      'CREATE POLICY %I ON %I.%I FOR ALL TO service_role USING (true) WITH CHECK (true)',
      policy_name,
      tbl.schemaname,
      tbl.tablename
    );
  END LOOP;
END $$;

-- ============================================================================
-- PART 5: Create postgres role bypass policies
-- In case Go backend connects directly as postgres user
-- ============================================================================

DO $$
DECLARE
  tbl RECORD;
  policy_name TEXT;
BEGIN
  FOR tbl IN
    SELECT schemaname, tablename
    FROM pg_tables
    WHERE schemaname = 'public'
    ORDER BY tablename
  LOOP
    policy_name := 'postgres_bypass_' || tbl.tablename;

    EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', policy_name, tbl.schemaname, tbl.tablename);

    EXECUTE format(
      'CREATE POLICY %I ON %I.%I FOR ALL TO postgres USING (true) WITH CHECK (true)',
      policy_name,
      tbl.schemaname,
      tbl.tablename
    );
  END LOOP;
END $$;
