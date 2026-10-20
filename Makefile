.DEFAULT_GOAL := help

.PHONY: help setup

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  setup  Install the local Git hooks"
	@echo "  help   Show this help"

setup:
	@git config --local core.hooksPath .githooks
	@chmod +x .githooks/*
	@echo "Local Git hooks installed from .githooks."
	@echo "Verify with: git config --local --get core.hooksPath"
