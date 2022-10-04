# dumptruck

dumptruck is a project for converting golang interfaces/structs/typedefs into the equivalent protobuf types as a starting point for a service migration

its main use case is for converting go code to fully fledged services

# status

this is still experimental, it works, but some stuff is not fully complete/ad-hoc

view the out directory to view the generated protobuf files

# example

the following interface 

```
package models

import (
	"context"

	"code.justin.tv/safety/go2proto/dummy/pkg1"
	nestpkg "code.justin.tv/safety/go2proto/dummy/pkg2/nest"
	"code.justin.tv/safety/go2proto/dummy/pkg4"
)

type TestInterface interface {
	GreatFunction() ([]string, error)
	GreatFunction2(ctx context.Context, arg *pkg1.A) ([]string, error)
	Function3(ctx context.Context) ([]string, []*string, error)
	Function4(ctx context.Context, limit uint64) ([]string, []*string, error)
	Function5(ctx context.Context, j int64) ([]string, []*string, error)
	Function6(ctx context.Context, c nestpkg.Country) ([]string, []*string, error)
	Function7(ctx context.Context, d pkg4.D) error
}
```

will generate

```
syntax = "proto3";
package root;
option go_package = "code.justin.tv/safety/gateway/testserver/rpc/testserver/gen/root";

import "google/protobuf/timestamp.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/duration.proto";

import "dummy/pkg1/const.proto";
import "dummy/pkg2/nest/const.proto";
import "dummy/pkg4/const.proto";

message Function3Request {
}

message Function3Response {
    repeated string Field1 = 1;
    repeated string Field2 = 2;
}

message Function4Request {
    uint64 limit = 1;
}

message Function4Response {
    repeated string Field1 = 1;
    repeated string Field2 = 2;
}

message Function5Request {
    int64 j = 1;
}

message Function5Response {
    repeated string Field1 = 1;
    repeated string Field2 = 2;
}

message Function6Request {
    dummy.pkg2.nest.Country c = 1;
}

message Function6Response {
    repeated string Field1 = 1;
    repeated string Field2 = 2;
}

message Function7Request {
    dummy.pkg4.D d = 1;
}

message Function7Response {
}

message GreatFunctionRequest {
}

message GreatFunctionResponse {
    repeated string Field1 = 1;
}

message GreatFunction2Request {
    optional dummy.pkg1.A arg = 1;
}

message GreatFunction2Response {
    repeated string Field1 = 1;
}

service Leviathan {
     rpc Function3(Function3Request) returns (Function3Response);
     rpc Function4(Function4Request) returns (Function4Response);
     rpc Function5(Function5Request) returns (Function5Response);
     rpc Function6(Function6Request) returns (Function6Response);
     rpc Function7(Function7Request) returns (Function7Response);
     rpc GreatFunction(GreatFunctionRequest) returns (GreatFunctionResponse);
     rpc GreatFunction2(GreatFunction2Request) returns (GreatFunction2Response);
}
```