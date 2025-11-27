
SAMCTR_VERSION := 0.0.3
CRTAR_VERSION := 0.0.2

all: samctr build_crtar

samctr:
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/samctr.version=v$(SAMCTR_VERSION)'" ./cmd/samctr

build_crtar:
	go build -ldflags "-X 'main.Version=v$(CRTAR_VERSION)'" ./cmd/crtar


clean_all: clean_crtar clean_samctr

clean_crtar:
	rm -f crtar

clean_samctr:
	rm -f samctr
