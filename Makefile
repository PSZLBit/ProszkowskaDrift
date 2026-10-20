ifeq ($(OS),Windows_NT)
HOOKS_EXECUTABLE := @echo Hooks use the executable mode tracked by Git.
else
HOOKS_EXECUTABLE := @chmod +x .githooks/*
endif

.DEFAULT_GOAL := help

.PHONY: help setup

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  setup  Install the local Git hooks (Linux/macOS)"
	@echo "  help   Show this help"
	@echo ""
	@echo "Windows: run setup.cmd instead."

setup:
	@git config --local core.hooksPath .githooks
	$(HOOKS_EXECUTABLE)
	@echo "Local Git hooks installed from .githooks."
	@echo "Verify with: git config --local --get core.hooksPath"
