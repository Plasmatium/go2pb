# go2pb
> A simple tool to generate protobuf files from Go source code.

## Usage

```
Usage of go2pb:
  -i, --in string    input go file, support glob pattern
  -o, --out string   output directory

example:
go2pb -i./example/*.go -o./example/proto
go2pb -i ./example/**/*.go -o./example/proto

```