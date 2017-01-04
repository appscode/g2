include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

.PHONY: test $(PKGS)
SHELL := /bin/bash
PKG = github.com/Clever/gearadmin
PKGS = $(shell go list ./...)
$(eval $(call golang-version-check,1.7))

test: $(PKGS)
$(PKGS): golang-test-all-deps
	$(call golang-test-all,$@)
