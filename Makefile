VERSION := 0.0.10-rc1

BUILDDIR = build

SAMCTR = $(BUILDDIR)/samctr
SAMGX = $(BUILDDIR)/samgx
CRTAR = $(BUILDDIR)/crtar
RGW = $(BUILDDIR)/rgw

EXE = $(SAMCTR) $(SAMGX) $(CRTAR) $(RGW)

all: $(EXE)

$(SAMCTR):
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/samctr.version=v$(VERSION)'" -o $@ ./cmd/samctr

$(SAMGX):
	go build -ldflags "-X 'main.Version=v$(VERSION)'" -o $@ ./cmd/samgx

$(CRTAR):
	go build -ldflags "-X 'main.Version=v$(VERSION)'" -o $@ ./cmd/crtar

$(RGW):
	go build -ldflags "-X 'main.Version=v$(VERSION)'" -o $@ ./cmd/rgw


.PHONY: clean
clean: 
	rm -rf $(BUILDDIR)

