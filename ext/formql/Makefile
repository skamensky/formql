EXTENSION = formql
MODULE_big = formql
OBJS = src/formql.o
DATA = sql/formql--0.1.0.sql
PG_CONFIG = pg_config
GO_BRIDGE_DIR = build
GO_BRIDGE_SHARED = $(GO_BRIDGE_DIR)/libformql_engine.so
GO_BRIDGE_HEADER = $(GO_BRIDGE_DIR)/libformql_engine.h
GO_BRIDGE_SOURCES = $(shell find ../../pkg -type f -name '*.go') ../../go.mod ../../go.sum

PGXS := $(shell $(PG_CONFIG) --pgxs 2>/dev/null)
ifeq ($(PGXS),)
$(error pg_config not found or PGXS unavailable; install PostgreSQL server development headers or set PG_CONFIG)
endif
PG_CPPFLAGS += -I$(CURDIR)/$(GO_BRIDGE_DIR)
PG_LDFLAGS += -Wl,-rpath,'$$ORIGIN'
SHLIB_LINK += -L$(CURDIR)/$(GO_BRIDGE_DIR) -lformql_engine
EXTRA_CLEAN += $(GO_BRIDGE_DIR)
include $(PGXS)

all: $(GO_BRIDGE_SHARED)

$(GO_BRIDGE_SHARED) $(GO_BRIDGE_HEADER): $(GO_BRIDGE_SOURCES)
	mkdir -p $(GO_BRIDGE_DIR)
	cd ../.. && go build -buildmode=c-shared -o ext/formql/$(GO_BRIDGE_SHARED) ./pkg/formql/capi

$(OBJS): $(GO_BRIDGE_HEADER)

install: install-go-bridge

install-go-bridge: $(GO_BRIDGE_SHARED)
	$(MKDIR_P) '$(DESTDIR)$(pkglibdir)'
	$(INSTALL_SHLIB) $(GO_BRIDGE_SHARED) '$(DESTDIR)$(pkglibdir)/libformql_engine.so'
