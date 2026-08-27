VERSION := 0.0.8

all: samctr crtar

samctr:
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/samctr.version=v$(VERSION)'" ./cmd/samctr

samgx:
	go build -ldflags "-X 'main.Version=v$(VERSION)'" ./cmd/samgx

crtar:
	go build -ldflags "-X 'main.Version=v$(VERSION)'" ./cmd/crtar


.PHONY: clean
clean: clean_crtar clean_samctr

clean_crtar:
	rm -f crtar

clean_samctr:
	rm -f samctr
