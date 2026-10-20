# Targets must work on Linux/macOS and Windows. Use OS conditionals for
# platform-specific commands and test both paths when adding a target.
ifeq ($(OS),Windows_NT)
HOOKS_EXECUTABLE := @echo Skipping chmod on Windows.
else
HOOKS_EXECUTABLE := @chmod +x .githooks/*
endif

.DEFAULT_GOAL := help

.PHONY: help setup

help:
	@echo Usage: make TARGET
	@echo Targets:
	@echo setup: Install the local Git hooks
	@echo help: Show this help

setup:
	@git config --local core.hooksPath .githooks
	$(HOOKS_EXECUTABLE)
	@echo Local Git hooks installed from .githooks.
	@echo Verify with: git config --local --get core.hooksPath
