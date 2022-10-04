proto:
	protoc -I out/ --go_out=. --proto_path=out --twirp_out=. server.proto
	protoc -I out/ --go_out=. --proto_path=out --twirp_out=. dummy/pkg1/const.proto
	protoc -I out/ --go_out=. --proto_path=out --twirp_out=. dummy/pkg2/nest/const.proto
	protoc -I out/ --go_out=. --proto_path=out --twirp_out=. dummy/pkg3/const.proto
	protoc -I out/ --go_out=. --proto_path=out --twirp_out=. dummy/pkg4/const.proto
	protoc -I out/ --go_out=. --proto_path=out --twirp_out=. meta/const.proto

test:
	CGO_ENABLED=1 go test -cover -race ./...