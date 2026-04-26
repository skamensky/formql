CREATE TABLE customers (
  id bigint PRIMARY KEY,
  email text NOT NULL
);

CREATE TABLE orders (
  id bigint PRIMARY KEY,
  customer_id bigint NOT NULL REFERENCES customers(id),
  amount numeric NOT NULL
);

CREATE INDEX orders_customer_id_idx ON orders(customer_id);

DO $$
DECLARE
  live_catalog jsonb;
  compiled jsonb;
  document_compiled jsonb;
  live_compiled jsonb;
  live_document_compiled jsonb;
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

  compiled := formql_compile_catalog(
    '{
      "base_table":"orders",
      "tables":[
        {
          "name":"orders",
          "columns":[
            {"name":"id","type":"number"},
            {"name":"amount","type":"number"}
          ]
        }
      ],
      "relationships":[]
    }'::jsonb,
    'amount + 1',
    'result',
    'syntax'
  );

  IF COALESCE((compiled->>'ok')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected compile ok response, got %', compiled::text;
  END IF;
  IF compiled #>> '{compilation,sql,query}' IS NULL THEN
    RAISE EXCEPTION 'expected generated query in compile response, got %', compiled::text;
  END IF;
  IF POSITION('amount' IN compiled #>> '{compilation,sql,query}') = 0 THEN
    RAISE EXCEPTION 'expected generated query to reference amount, got %', compiled::text;
  END IF;
  IF COALESCE((compiled #>> '{verification,ok}')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected verification ok in compile response, got %', compiled::text;
  END IF;

  document_compiled := formql_compile_document_catalog(
    '{
      "base_table":"orders",
      "tables":[
        {
          "name":"orders",
          "columns":[
            {"name":"id","type":"number"},
            {"name":"amount","type":"number"}
          ]
        }
      ],
      "relationships":[]
    }'::jsonb,
    'amount, amount + 1 AS amount_plus_one',
    'syntax'
  );

  IF COALESCE((document_compiled->>'ok')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected document compile ok response, got %', document_compiled::text;
  END IF;
  IF POSITION('"amount_plus_one"' IN document_compiled #>> '{compilation,sql,query}') = 0 THEN
    RAISE EXCEPTION 'expected document query to include explicit projection alias, got %', document_compiled::text;
  END IF;
  IF COALESCE((document_compiled #>> '{verification,ok}')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected verification ok in document compile response, got %', document_compiled::text;
  END IF;

  live_catalog := formql_catalog('orders');
  IF live_catalog #>> '{base_table}' <> 'orders' THEN
    RAISE EXCEPTION 'expected live catalog base_table=orders, got %', live_catalog::text;
  END IF;
  IF POSITION('customer' IN live_catalog::text) = 0 THEN
    RAISE EXCEPTION 'expected live catalog to include relationship metadata, got %', live_catalog::text;
  END IF;

  live_compiled := formql_compile_live(
    'orders',
    'customer_id__rel.email & " / " & STRING(amount)',
    'result',
    'syntax'
  );

  IF COALESCE((live_compiled->>'ok')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected live compile ok response, got %', live_compiled::text;
  END IF;
  IF POSITION('"customers"' IN live_compiled #>> '{compilation,sql,query}') = 0 THEN
    RAISE EXCEPTION 'expected live compile query to include customers join, got %', live_compiled::text;
  END IF;
  IF COALESCE((live_compiled #>> '{verification,ok}')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected live verification ok in compile response, got %', live_compiled::text;
  END IF;

  live_document_compiled := formql_compile_document_live(
    'orders',
    'amount, customer_id__rel.email AS customer_email',
    'syntax'
  );

  IF COALESCE((live_document_compiled->>'ok')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected live document compile ok response, got %', live_document_compiled::text;
  END IF;
  IF POSITION('"customer_email"' IN live_document_compiled #>> '{compilation,sql,query}') = 0 THEN
    RAISE EXCEPTION 'expected live document query to include customer_email projection, got %', live_document_compiled::text;
  END IF;
  IF POSITION('JOIN "customers"' IN live_document_compiled #>> '{compilation,sql,query}') = 0 THEN
    RAISE EXCEPTION 'expected live document query to include customers join, got %', live_document_compiled::text;
  END IF;
  IF COALESCE((live_document_compiled #>> '{verification,ok}')::boolean, false) IS NOT TRUE THEN
    RAISE EXCEPTION 'expected live document verification ok in compile response, got %', live_document_compiled::text;
  END IF;
END $$;
