GO := /usr/bin/env go

BUILDDIR = build

SAMI = $(BUILDDIR)/sami
SAMI_VERSION = 0.0.9-rc6

default: sami

sami: $(SAMI)

$(SAMI): cmd/sami/main.go
	mkdir -p $(BUILDDIR)
	$(GO) build -ldflags "-X 'gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/pkg/cmd/sami.version=v$(SAMI_VERSION)'" -o $@ $^


.phony: clean
clean:
	rm -f $(SAMI)

