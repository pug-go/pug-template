include pug.mk

# Add your commands here

protoc-gen:
	go build -o bin/protoc-gen-pog ./cmd/protoc-gen-pog
	make fast-generate
