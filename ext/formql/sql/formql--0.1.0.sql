CREATE FUNCTION formql_verify_sql_error(sql text)
RETURNS text
AS 'MODULE_PATHNAME', 'formql_verify_sql_error'
LANGUAGE C IMMUTABLE STRICT;

CREATE FUNCTION formql_verify_sql_ok(sql text)
RETURNS boolean
AS 'MODULE_PATHNAME', 'formql_verify_sql_ok'
LANGUAGE C IMMUTABLE STRICT;

CREATE FUNCTION formql_verify_sql_diagnostics(sql text)
RETURNS jsonb
AS 'MODULE_PATHNAME', 'formql_verify_sql_diagnostics'
LANGUAGE C IMMUTABLE STRICT;
