UID=$(shell id -u)
GID=$(shell id -g)

DOCKER_BUILDKIT=1
DOCKER_TAG:=$(shell pwd | md5sum | cut -f1 -d ' ')
DOCKER_RUN_FLAGS+=-v $(PWD):/workspace
DOCKER_RUN_FLAGS+=-it
DOCKER_RUN_FLAGS+=--rm
DOCKER_RUN_FLAGS+=--user $(UID):$(GID)

TARGETS=run build test lint format clean setup

ifeq ($(SKIP_DOCKER), true)

#================No Docker/Already in container================
$(TARGETS):
	make -f Makefile.main $@
#==============================================================

else

#========================Use Docker============================
default: enter

setup-docker:
	docker buildx build --build-arg UID=$(UID) --build-arg GID=$(GID) --tag $(DOCKER_TAG) .

$(TARGETS): setup-docker
	@docker run $(DOCKER_RUN_FLAGS) $(DOCKER_TAG) make -f Makefile.main $@

enter: setup-docker
	@docker run $(DOCKER_RUN_FLAGS) $(DOCKER_TAG) /bin/sh
#==============================================================

endif

