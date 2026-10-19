VERSION := $(shell git describe --tags --always --dirty)

BUILDDIR = build

SAMI = $(BUILDDIR)/sami
SAMCTR = $(BUILDDIR)/samctr
SAMGX = $(BUILDDIR)/samgx
CRTAR = $(BUILDDIR)/crtar
RGW = $(BUILDDIR)/rgw

EXE = $(SAMI) $(SAMCTR) $(SAMGX) $(CRTAR) $(RGW)

all: $(EXE)

$(SAMI):
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/sami.version=$(VERSION)'" -o $@ ./cmd/sami

$(SAMCTR):
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/samctr.version=$(VERSION)'" -o $@ ./cmd/samctr

$(SAMGX):
	go build -ldflags "-X 'main.Version=$(VERSION)'" -o $@ ./cmd/samgx

$(CRTAR):
	go build -ldflags "-X 'main.Version=$(VERSION)'" -o $@ ./cmd/crtar

$(RGW):
	go build -ldflags "-X 'main.Version=$(VERSION)'" -o $@ ./cmd/rgw


.PHONY: clean
clean: 
	rm -rf $(BUILDDIR)

