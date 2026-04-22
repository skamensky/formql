DO $$
BEGIN
  IF formql_verify_sql_error('SELECT 1') IS NOT NULL THEN
    RAISE EXCEPTION 'expected NULL error for valid SQL';
  END IF;
  IF NOT formql_verify_sql_ok('SELECT 1') THEN
    RAISE EXCEPTION 'expected TRUE ok for valid SQL';
  END IF;
  IF formql_verify_sql_diagnostics('SELECT 1') IS NOT NULL THEN
    RAISE EXCEPTION 'expected NULL diagnostics for valid SQL';
  END IF;

  IF formql_verify_sql_error('SELECT FROM') IS NULL THEN
    RAISE EXCEPTION 'expected non-null error for invalid SQL';
  END IF;
  IF formql_verify_sql_ok('SELECT FROM') THEN
    RAISE EXCEPTION 'expected FALSE ok for invalid SQL';
  END IF;
  IF formql_verify_sql_diagnostics('SELECT FROM') IS NULL THEN
    RAISE EXCEPTION 'expected diagnostics json for invalid SQL';
  END IF;
END $$;
