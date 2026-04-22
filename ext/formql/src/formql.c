#include "postgres.h"

#include "fmgr.h"
#include "lib/stringinfo.h"
#include "nodes/parsenodes.h"
#include "parser/parser.h"
#include "utils/builtins.h"
#include "utils/json.h"

PG_MODULE_MAGIC;

PG_FUNCTION_INFO_V1(formql_verify_sql_error);
PG_FUNCTION_INFO_V1(formql_verify_sql_ok);
PG_FUNCTION_INFO_V1(formql_verify_sql_diagnostics);

typedef struct VerifyFailure {
	char *message;
	char sqlstate[6];
} VerifyFailure;

static bool verify_sql_syntax(const char *sql, VerifyFailure *failure) {
	PG_TRY();
	{
		raw_parser(sql, RAW_PARSE_DEFAULT);
		return true;
	}
	PG_CATCH();
	{
		ErrorData *edata = CopyErrorData();
		FlushErrorState();
		failure->message = pstrdup(edata->message ? edata->message : "parse failure");
		memcpy(failure->sqlstate, unpack_sql_state(edata->sqlerrcode), 5);
		failure->sqlstate[5] = '\0';
		FreeErrorData(edata);
		return false;
	}
	PG_END_TRY();
}

static Datum diagnostics_jsonb_datum(const VerifyFailure *failure) {
	StringInfoData json;
	initStringInfo(&json);
	appendStringInfoString(&json, "{\"ok\":false,\"diagnostics\":[{\"message\":");
	escape_json(&json, failure->message);
	appendStringInfoString(&json, ",\"sqlstate\":");
	escape_json(&json, failure->sqlstate);
	appendStringInfoString(&json, "}]}");
	return DirectFunctionCall1(jsonb_in, CStringGetDatum(json.data));
}

Datum formql_verify_sql_error(PG_FUNCTION_ARGS) {
	text *sql_text = PG_GETARG_TEXT_PP(0);
	char *sql = text_to_cstring(sql_text);
	VerifyFailure failure;

	if (verify_sql_syntax(sql, &failure)) {
		PG_RETURN_NULL();
	}

	PG_RETURN_TEXT_P(cstring_to_text(failure.message));
}

Datum formql_verify_sql_ok(PG_FUNCTION_ARGS) {
	text *sql_text = PG_GETARG_TEXT_PP(0);
	char *sql = text_to_cstring(sql_text);
	VerifyFailure failure;

	if (verify_sql_syntax(sql, &failure)) {
		PG_RETURN_BOOL(true);
	}

	PG_RETURN_BOOL(false);
}

Datum formql_verify_sql_diagnostics(PG_FUNCTION_ARGS) {
	text *sql_text = PG_GETARG_TEXT_PP(0);
	char *sql = text_to_cstring(sql_text);
	VerifyFailure failure;

	if (verify_sql_syntax(sql, &failure)) {
		PG_RETURN_NULL();
	}

	PG_RETURN_DATUM(diagnostics_jsonb_datum(&failure));
}
