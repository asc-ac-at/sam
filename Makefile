
SAMCTR_VERSION := 0.0.6
CRTAR_VERSION := 0.0.2
SAMGX_VERSION := 0.0.1

all: samctr build_crtar

samctr:
	go build -ldflags "-X 'github.com/asc-ac-at/sam/pkg/cmd/samctr.version=v$(SAMCTR_VERSION)'" ./cmd/samctr

samgx:
	go build -ldflags "-X 'main.Version=v$(SAMGX_VERSION)'" ./cmd/samgx

build_crtar:
	go build -ldflags "-X 'main.Version=v$(CRTAR_VERSION)'" ./cmd/crtar


clean_all: clean_crtar clean_samctr

clean_crtar:
	rm -f crtar

clean_samctr:
	rm -f samctr
