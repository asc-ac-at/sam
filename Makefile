VERSION := 0.0.8

BUILDDIR = build

SAMCTR = $(BUILDDIR)/samctr
SAMGX = $(BUILDDIR)/samgx
CRTAR = $(BUILDDIR)/crtar

EXE = $(SAMCTR) $(SAMGX) $(CRTAR)

all: $(EXE)

$(SAMCTR):
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/samctr.version=v$(VERSION)'" -o $@ ./cmd/samctr

$(SAMGX):
	go build -ldflags "-X 'main.Version=v$(VERSION)'" -o $@ ./cmd/samgx

$(CRTAR):
	go build -ldflags "-X 'main.Version=v$(VERSION)'" -o $@ ./cmd/crtar


.PHONY: clean
clean: 
	rm -rf $(BUILDDIR)

