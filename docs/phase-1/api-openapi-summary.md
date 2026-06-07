# API OpenAPI/Swagger Summary

Generated OpenAPI specs are at `gen/openapi/api.swagger.yaml`.

## Status

`buf generate proto/` runs successfully and produces `gen/openapi/api.swagger.yaml` (740 lines).

The spec includes all message type definitions (schemas) for all services. HTTP path entries
(`paths:`) are currently empty because the proto RPCs do not yet have `option (google.api.http)`
annotations. The gRPC-Gateway plugin is configured with `generate_unbound_methods=true`, which
generates gateway scaffolding code without HTTP bindings.

**Phase 2 action:** Add `option (google.api.http)` annotations to each RPC in the proto files
to populate the `paths:` section of the OpenAPI spec and enable REST transcoding via
gRPC-Gateway.

## Services Covered

| Service | Proto File | RPC Count |
|---|---|---|
| tenant/v1 | TenantService | tenant CRUD |
| tenant/v1 | UserService | register, login, PIN, logout |
| pos/v1 | ProductService | product/category/addon CRUD |
| pos/v1 | OrderService | order lifecycle + shift management |
| payment/v1 | PaymentService | initiate, confirm, refund, webhook |
| inventory/v1 | InventoryService | stock movement, BOM, alerts |

## Generated Files

| File | Location | Description |
|---|---|---|
| `api.swagger.yaml` | `gen/openapi/` | Merged OpenAPI 2.0 spec (all services) |
| `*.pb.go` | `gen/go/{service}/v1/` | Go protobuf message types |
| `*_grpc.pb.go` | `gen/go/{service}/v1/` | Go gRPC service stubs |
| `*.pb.gw.go` | `gen/go/{service}/v1/` | Go gRPC-Gateway HTTP handlers |

Note: `gen/` is gitignored — regenerate with `buf generate proto/`.

## Viewing the Spec

Once HTTP annotations are added, view the spec with Swagger UI or import into Postman:

```bash
# Using swagger-ui-serve (npm)
npx @redocly/cli preview-docs gen/openapi/api.swagger.yaml

# Or import into Postman via:
# File > Import > gen/openapi/api.swagger.yaml
```

## Regenerating

```bash
buf generate proto/
```

This runs the following plugins as configured in `buf.gen.yaml`:
- `buf.build/protocolbuffers/go` — protobuf message types
- `buf.build/grpc/go` — gRPC service stubs
- `buf.build/grpc-ecosystem/gateway` — gRPC-Gateway HTTP handlers
- `buf.build/grpc-ecosystem/openapiv2` — OpenAPI 2.0 spec (this file)
- `buf.build/connectrpc/es` + `buf.build/bufbuild/es` — TypeScript client

## Next Steps: Adding HTTP Annotations

Example for `OrderService.CreateOrder`:

```protobuf
import "google/api/annotations.proto";

service OrderService {
  rpc CreateOrder (CreateOrderRequest) returns (CreateOrderResponse) {
    option (google.api.http) = {
      post: "/v1/orders"
      body: "*"
    };
  }
}
```

After adding annotations, re-run `buf generate proto/` and `paths:` will be populated in the
OpenAPI spec.
