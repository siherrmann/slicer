MAKEFLAGS += -j2
.PHONY: run install clean server tailwind

# Run server
run:
	wgo -file .go -file .templ -xfile _templ.go clear :: templ generate :: go run ./example/viewer/main.go -name slicer

# Clean process left by wgo in server command
# This is only needed because the inner process from templ generate does not stop properly
# With `go run .` i works, but with the filewatcher (wgo) it does not kill the inner process
clean:
	@clear; \
	PID=$$(pgrep -f slicer); \
	if [ -n "$$PID" ]; then \
		kill -9 $$PID; \
		echo "Killed process $$PID"; \
	else \
		echo "No programm running with name slicer"; \
	fi;
	