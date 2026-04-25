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

CREATE FUNCTION formql_catalog(base_table text)
RETURNS jsonb
AS 'MODULE_PATHNAME', 'formql_catalog'
LANGUAGE C STABLE STRICT;

CREATE FUNCTION formql_compile_catalog(
  catalog jsonb,
  formula text,
  field_alias text DEFAULT 'result',
  verify_mode text DEFAULT 'syntax'
)
RETURNS jsonb
AS 'MODULE_PATHNAME', 'formql_compile_catalog'
LANGUAGE C IMMUTABLE STRICT;

CREATE FUNCTION formql_compile_live(
  base_table text,
  formula text,
  field_alias text DEFAULT 'result',
  verify_mode text DEFAULT 'syntax'
)
RETURNS jsonb
AS 'MODULE_PATHNAME', 'formql_compile_live'
LANGUAGE C STABLE STRICT;

CREATE FUNCTION formql_compile_document_catalog(
  catalog jsonb,
  document text,
  verify_mode text DEFAULT 'syntax'
)
RETURNS jsonb
AS 'MODULE_PATHNAME', 'formql_compile_document_catalog'
LANGUAGE C IMMUTABLE STRICT;

CREATE FUNCTION formql_compile_document_live(
  base_table text,
  document text,
  verify_mode text DEFAULT 'syntax'
)
RETURNS jsonb
AS 'MODULE_PATHNAME', 'formql_compile_document_live'
LANGUAGE C STABLE STRICT;
