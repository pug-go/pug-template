# pug-template

Backend service boilerplate.

```bash
# write proto api/{service}/v1/{service}.proto

# generate handlers (also `make fast-generate` if deps already installed)
$ make generate

# run unit tests
$ make test

# lint code
$ make lint

# fix code
$ make fmt
```

TODO:
- [ ] Move to gitlab
- [ ] Refactor protoc-gen-pug
- [ ] Add dao and postgres support
- [ ] Add gitlab ci/cd
- [ ] Add jaeger support ([tracing](https://grpc-ecosystem.github.io/grpc-gateway/docs/operations/tracing/))
