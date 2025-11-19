.PHONY: help install run-dev run-build clean

help:
	@echo "Pepeunit Golang Client - Commands:"
	@echo ""
	@echo "install:          Install all dependencies"
	@echo "clean:            Clean cache package"

install:
	@echo "Install all dependencies"
	go mod download

clean:
	@echo "Clean cache package..."
	go clean -modcache
