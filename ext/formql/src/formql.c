#include "postgres.h"

#include "catalog/pg_type_d.h"
#include "executor/spi.h"
#include "fmgr.h"
#include "utils/builtins.h"
#include "utils/jsonb.h"

#include "libformql_engine.h"

PG_MODULE_MAGIC;

PG_FUNCTION_INFO_V1(formql_verify_sql_error);
PG_FUNCTION_INFO_V1(formql_verify_sql_ok);
PG_FUNCTION_INFO_V1(formql_verify_sql_diagnostics);
PG_FUNCTION_INFO_V1(formql_catalog);
PG_FUNCTION_INFO_V1(formql_compile_catalog);
PG_FUNCTION_INFO_V1(formql_compile_live);

typedef struct QualifiedName {
	char *schema_name;
	char *table_name;
} QualifiedName;

static const char *live_catalog_query =
	"SELECT jsonb_build_object("
	"'namespace', lower($1), "
	"'base_table', lower($2), "
	"'columns', COALESCE(("
	"  SELECT jsonb_agg("
	"    jsonb_build_object("
	"      'table_name', lower(c.table_name), "
	"      'column_name', lower(c.column_name), "
	"      'data_type', lower(c.data_type), "
	"      'udt_name', lower(c.udt_name)"
	"    ) "
	"    ORDER BY lower(c.table_name), c.ordinal_position"
	"  ) "
	"  FROM information_schema.columns c "
	"  WHERE c.table_schema = $1"
	"), '[]'::jsonb), "
	"'relationships', COALESCE(("
	"  SELECT jsonb_agg("
	"    jsonb_build_object("
	"      'source_table', lower(tc.table_name), "
	"      'source_column', lower(kcu.column_name), "
	"      'target_table', lower(ccu.table_name), "
	"      'target_column', lower(ccu.column_name), "
	"      'join_column_indexed', EXISTS ("
	"        SELECT 1 "
	"        FROM pg_index idx "
	"        JOIN pg_class cls ON cls.oid = idx.indrelid "
	"        JOIN pg_namespace ns ON ns.oid = cls.relnamespace "
	"        JOIN pg_attribute attr ON attr.attrelid = cls.oid "
	"        WHERE ns.nspname = tc.table_schema "
	"          AND cls.relname = tc.table_name "
	"          AND attr.attname = kcu.column_name "
	"          AND attr.attnum = ANY(idx.indkey)"
	"      ), "
	"      'target_column_indexed', EXISTS ("
	"        SELECT 1 "
	"        FROM pg_index idx "
	"        JOIN pg_class cls ON cls.oid = idx.indrelid "
	"        JOIN pg_namespace ns ON ns.oid = cls.relnamespace "
	"        JOIN pg_attribute attr ON attr.attrelid = cls.oid "
	"        WHERE ns.nspname = tc.table_schema "
	"          AND cls.relname = ccu.table_name "
	"          AND attr.attname = ccu.column_name "
	"          AND attr.attnum = ANY(idx.indkey)"
	"      )"
	"    ) "
	"    ORDER BY lower(tc.table_name), lower(kcu.column_name)"
	"  ) "
	"  FROM information_schema.table_constraints tc "
	"  JOIN information_schema.key_column_usage kcu "
	"    ON tc.constraint_name = kcu.constraint_name "
	"    AND tc.table_schema = kcu.table_schema "
	"  JOIN information_schema.constraint_column_usage ccu "
	"    ON ccu.constraint_name = tc.constraint_name "
	"    AND ccu.table_schema = tc.table_schema "
	"  WHERE tc.constraint_type = 'FOREIGN KEY' "
	"    AND tc.table_schema = $1"
	"), '[]'::jsonb)"
	")::text";

static Datum jsonb_datum_from_formql_cstring(char *json) {
	Datum datum = DirectFunctionCall1(jsonb_in, CStringGetDatum(json));
	FormQLFreeCString(json);
	return datum;
}

static text *cstring_to_text_and_free(char *value) {
	text *result = cstring_to_text(value);
	FormQLFreeCString(value);
	return result;
}

static char *lowercase_copy(const char *value) {
	char *copy = pstrdup(value);
	for (char *cursor = copy; *cursor != '\0'; cursor++) {
		*cursor = pg_tolower((unsigned char) *cursor);
	}
	return copy;
}

static QualifiedName parse_qualified_name(const char *raw) {
	QualifiedName result;
	char *copy;
	char *dot;

	copy = lowercase_copy(raw);
	dot = strchr(copy, '.');
	if (dot == NULL) {
		result.schema_name = pstrdup("public");
		result.table_name = copy;
		return result;
	}

	*dot = '\0';
	result.schema_name = pstrdup(copy);
	result.table_name = pstrdup(dot + 1);
	return result;
}

static char *live_catalog_json(const char *base_table) {
	Oid argtypes[2] = {TEXTOID, TEXTOID};
	Datum values[2];
	Datum json_datum;
	bool isnull = false;
	char nulls[2] = {' ', ' '};
	char *json;
	int spi_status;
	QualifiedName name = parse_qualified_name(base_table);

	values[0] = CStringGetTextDatum(name.schema_name);
	values[1] = CStringGetTextDatum(name.table_name);

	spi_status = SPI_connect();
	if (spi_status != SPI_OK_CONNECT) {
		ereport(ERROR, (errmsg("SPI_connect failed: %d", spi_status)));
	}

	spi_status = SPI_execute_with_args(live_catalog_query, 2, argtypes, values, nulls, true, 1);
	if (spi_status != SPI_OK_SELECT || SPI_tuptable == NULL || SPI_processed != 1) {
		SPI_finish();
		ereport(ERROR, (errmsg("catalog introspection query failed")));
	}

	json_datum = SPI_getbinval(SPI_tuptable->vals[0], SPI_tuptable->tupdesc, 1, &isnull);
	if (isnull) {
		SPI_finish();
		ereport(ERROR, (errmsg("catalog introspection returned NULL")));
	}

	json = TextDatumGetCString(json_datum);
	SPI_finish();

	return json;
}

Datum formql_verify_sql_error(PG_FUNCTION_ARGS) {
	text *sql_text = PG_GETARG_TEXT_PP(0);
	char *sql = text_to_cstring(sql_text);
	char *message = FormQLVerifySQLError(sql, "syntax");

	if (message == NULL) {
		PG_RETURN_NULL();
	}

	PG_RETURN_TEXT_P(cstring_to_text_and_free(message));
}

Datum formql_verify_sql_ok(PG_FUNCTION_ARGS) {
	text *sql_text = PG_GETARG_TEXT_PP(0);
	char *sql = text_to_cstring(sql_text);

	PG_RETURN_BOOL(FormQLVerifySQLOk(sql, "syntax") != 0);
}

Datum formql_verify_sql_diagnostics(PG_FUNCTION_ARGS) {
	text *sql_text = PG_GETARG_TEXT_PP(0);
	char *sql = text_to_cstring(sql_text);
	char *json = FormQLVerifySQLDiagnostics(sql, "syntax");

	if (json == NULL) {
		PG_RETURN_NULL();
	}

	PG_RETURN_DATUM(jsonb_datum_from_formql_cstring(json));
}

Datum formql_catalog(PG_FUNCTION_ARGS) {
	text *base_table_text = PG_GETARG_TEXT_PP(0);
	char *base_table = text_to_cstring(base_table_text);
	char *introspection_json = live_catalog_json(base_table);
	char *json = FormQLCatalogFromIntrospectionJSON(introspection_json);

	if (json == NULL) {
		ereport(ERROR, (errmsg("formql catalog bridge returned NULL")));
	}

	pfree(introspection_json);
	PG_RETURN_DATUM(jsonb_datum_from_formql_cstring(json));
}

Datum formql_compile_catalog(PG_FUNCTION_ARGS) {
	Datum catalog = PG_GETARG_DATUM(0);
	text *formula_text = PG_GETARG_TEXT_PP(1);
	text *field_alias_text = PG_GETARG_TEXT_PP(2);
	text *verify_mode_text = PG_GETARG_TEXT_PP(3);
	char *catalog_json = DatumGetCString(DirectFunctionCall1(jsonb_out, catalog));
	char *formula = text_to_cstring(formula_text);
	char *field_alias = text_to_cstring(field_alias_text);
	char *verify_mode = text_to_cstring(verify_mode_text);
	char *json = FormQLCompileCatalogJSON(catalog_json, formula, field_alias, verify_mode);

	if (json == NULL) {
		ereport(ERROR, (errmsg("formql compile bridge returned NULL")));
	}

	PG_RETURN_DATUM(jsonb_datum_from_formql_cstring(json));
}

Datum formql_compile_live(PG_FUNCTION_ARGS) {
	text *base_table_text = PG_GETARG_TEXT_PP(0);
	text *formula_text = PG_GETARG_TEXT_PP(1);
	text *field_alias_text = PG_GETARG_TEXT_PP(2);
	text *verify_mode_text = PG_GETARG_TEXT_PP(3);
	char *base_table = text_to_cstring(base_table_text);
	char *introspection_json = live_catalog_json(base_table);
	char *formula = text_to_cstring(formula_text);
	char *field_alias = text_to_cstring(field_alias_text);
	char *verify_mode = text_to_cstring(verify_mode_text);
	char *json = FormQLCompileIntrospectionJSON(introspection_json, formula, field_alias, verify_mode);

	if (json == NULL) {
		ereport(ERROR, (errmsg("formql live compile bridge returned NULL")));
	}

	pfree(introspection_json);
	PG_RETURN_DATUM(jsonb_datum_from_formql_cstring(json));
}
