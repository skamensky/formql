EXTENSION = formql
MODULE_big = formql
OBJS = src/formql.o
DATA = sql/formql--0.1.0.sql
PG_CONFIG = pg_config

PGXS := $(shell $(PG_CONFIG) --pgxs)
include $(PGXS)
