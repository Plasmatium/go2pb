# go2pb
> A simple tool to generate protobuf files from Go source code.

## Usage

```
Usage of /var/folders/x7/c58k11hn46d21b8rf1mvqsyw0000gp/T/go-build1237606447/b001/exe/go2pb:
  -b, --base string      base directory for relative import path
  -i, --in string        input go file, support glob pattern
  -o, --out string       output directory

example:
go2pb -i./example/*.go -o./example/proto
go2pb -i ./users/**/*.go -o ./users/proto -b protodef

```