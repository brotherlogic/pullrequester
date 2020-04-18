protoc --proto_path ../../../ -I=./proto --go_out=plugins=grpc:./proto proto/pullrequester.proto
mv proto/github.com/brotherlogic/pullrequester/proto/* ./proto
