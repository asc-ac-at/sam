GO := /usr/bin/env go

BUILDDIR = build

SAMI = $(BUILDDIR)/sami
SAMI_VERSION = 0.0.8rc1

default: sami

sami: $(SAMI)

$(SAMI): main.go
	mkdir -p $(BUILDDIR)
	$(GO) build -ldflags "-X 'gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/cmd.version=v$(SAMI_VERSION)'" -o $@ $^


.phony: clean
clean:
	rm -f $(SAMI)

