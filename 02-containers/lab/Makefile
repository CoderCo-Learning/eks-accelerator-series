# EP2 lab Makefile.
#
#   make build         build all nine images, tagged with the short SHA
#   make build SVC=order-service   build just one service
#   make sizes         print the image sizes for the current SHA
#   make clean         remove every image we tagged
#
# Run from this folder. Paths assume the repo layout.

SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
PLATFORM_DIR := ../../platform/services
DOCKERFILE := $(CURDIR)/Dockerfile
DOCKERFILE_WORKER := $(CURDIR)/Dockerfile.worker

SERVICES := api-gateway order-service inventory-service payment-service \
            notification-service shipping-service scheduler dashboard-api
WORKERS  := worker

ALL := $(SERVICES) $(WORKERS)

.PHONY: build sizes clean $(ALL)

build: $(if $(SVC),$(SVC),$(ALL))

$(SERVICES):
	@echo "→ building $@:$(SHA)"
	@docker build \
	  -f $(DOCKERFILE) \
	  -t $@:$(SHA) \
	  -t $@:latest \
	  $(PLATFORM_DIR)/$@

$(WORKERS):
	@echo "→ building $@:$(SHA)"
	@docker build \
	  -f $(DOCKERFILE_WORKER) \
	  -t $@:$(SHA) \
	  -t $@:latest \
	  $(PLATFORM_DIR)/$@

sizes:
	@for s in $(ALL); do \
	  docker images --format "{{.Repository}}:{{.Tag}}\t{{.Size}}" $$s:$(SHA); \
	done

clean:
	@for s in $(ALL); do \
	  docker rmi $$s:$(SHA) $$s:latest 2>/dev/null || true; \
	done
