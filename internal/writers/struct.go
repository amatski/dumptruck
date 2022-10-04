package writers

import (
	"fmt"
	"os"
	"strings"

	"code.justin.tv/safety/go2proto/internal"
)

func WriteStructConverters(structs []internal.Struct, deps map[string]internal.DependencySet, pkgPrefixSlash string) {
	pkgFiles := map[string]*strings.Builder{} // each converter is 1:1 with the package it belongs to

	// Write header of converter file
	for _, structImpl := range structs {
		if _, ok := pkgFiles[structImpl.Package]; !ok {
			pkgFiles[structImpl.Package] = &strings.Builder{}

			sb := pkgFiles[structImpl.Package]
			sb.WriteString(fmt.Sprintf("package %s\n\n", structImpl.Package))
			sb.WriteString("import (\n")

			pkg := structImpl.Package
			sb.WriteString(fmt.Sprintf("    pb%s \"%s/%s\"\n", pkg, pkgPrefixSlash, pkg))
			sb.WriteString(fmt.Sprintf("    %s \"code.justin.tv/safety/datastore/v5/%s\"\n", pkg, *structImpl.Path.Path))

			// import the converters
			for pkg := range deps[structImpl.Package] {
				sb.WriteString(fmt.Sprintf("    converter%s \"code.justin.tv/safety/gateway/testserver/rpc/testserver/gen/converters/%s\"\n", pkg, pkg))
			}
			sb.WriteString("	\"google.golang.org/protobuf/types/known/timestamppb\"")

			sb.WriteString(")\n\n")
		}
	}

	// Write all of the structs
	for _, structImpl := range structs {
		sb := pkgFiles[structImpl.Package]
		goType := fmt.Sprintf("%s.%s", structImpl.Package, structImpl.Name)
		sb.WriteString(fmt.Sprintf("func %sFromPb(ent pb%s) (%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString(fmt.Sprintf("     return %s.%s {\n", structImpl.Package, structImpl.Name))
		for _, field := range structImpl.Fields {
			convert := ""
			if field.Type == "time.Time" {
				convert = ".AsTime()"
			} else if field.Type == "interface" { // interfaces are encoded as google.Protobuf.Value
				convert = ".AsInterface()"
			}

			slice := ""
			if field.Repeated {
				slice = "Slice"
			}

			if isPodType(field.Type) {
				sb.WriteString(fmt.Sprintf("         %s: ent.%s%s,\n", field.Name, field.Name, convert))
			} else {
				ptr := ""
				if field.Optional {
					ptr = "Ptr"
				}
				// we stupidly sometimes prepend selector with the dot instead of just
				// having a field as to whether or not it references another package
				actualType := field.Type
				if field.ComputeSelector() != nil {
					actualType = fmt.Sprintf("converter%s", field.Type)
				}
				sb.WriteString(fmt.Sprintf("         %s: %sFromPb%s(ent.%s%s),\n", field.Name, actualType, ptr+slice, field.Name, convert))
			}
		}
		sb.WriteString("     }\n")
		sb.WriteString("}\n\n")

		// Write the same function but for ptr
		// todo: refactor this so it reuses the code above
		sb.WriteString(fmt.Sprintf("func %sFromPbPtr(ent *pb%s) (*%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString("     if ent == nil{\n")
		sb.WriteString("         return nil\n")
		sb.WriteString("     }\n")
		sb.WriteString(fmt.Sprintf("     return &%s.%s {\n", structImpl.Package, structImpl.Name))
		for _, field := range structImpl.Fields {
			convert := ""
			if field.Type == "time.Time" {
				convert = ".AsTime()"
			} else if field.Type == "interface" { // interfaces are encoded as google.Protobuf.Value
				convert = ".AsInterface()"
			}

			slice := ""
			if field.Repeated {
				slice = "Slice"
			}

			if isPodType(field.Type) {
				sb.WriteString(fmt.Sprintf("         %s: ent.%s%s,\n", field.Name, field.Name, convert))
			} else {
				ptr := ""
				if field.Optional {
					ptr = "Ptr"
				}
				actualType := field.Type
				if field.ComputeSelector() != nil {
					actualType = fmt.Sprintf("converter%s", field.Type)
				}
				sb.WriteString(fmt.Sprintf("         %s: %sFromPb%s(ent.%s%s),\n", field.Name, actualType, ptr, field.Name+slice, convert))
			}
		}
		sb.WriteString("     }\n")
		sb.WriteString("}\n\n")

		sb.WriteString(fmt.Sprintf("func %sFromGo(ent %s) (pb%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString(fmt.Sprintf("     return pb%s {\n", goType))
		for _, field := range structImpl.Fields {
			convert := ""
			// note these timestamps are always pointers when given to us in protobuf
			if field.Type == "time.Time" {
				convert = "timestamppb.New"
			}

			slice := ""
			if field.Repeated {
				slice = "Slice"
			}

			if isPodType(field.Type) {
				if convert != "" {
					// some weird shit to deref a ptr in case cause this convert func (our only one) returns a ptr
					sb.WriteString(fmt.Sprintf("         %s: %s(ent.%s),\n", field.Name, convert, field.Name))
				} else {
					sb.WriteString(fmt.Sprintf("         %s: ent.%s,\n", field.Name, field.Name))
				}
			} else {
				ptr := ""
				if field.Optional {
					ptr = "Ptr"
				}
				actualType := field.Type
				if field.ComputeSelector() != nil {
					actualType = fmt.Sprintf("converter%s", field.Type)
				}
				sb.WriteString(fmt.Sprintf("         %s: %sFromGo%s(ent.%s),\n", field.Name, actualType, ptr+slice, field.Name))
			}
		}
		sb.WriteString("     }\n")
		sb.WriteString("}\n\n")

		sb.WriteString(fmt.Sprintf("func %sFromGoPtr(ent *%s) (*pb%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString("     if ent == nil{\n")
		sb.WriteString("         return nil\n")
		sb.WriteString("     }\n")
		sb.WriteString(fmt.Sprintf("     return &pb%s {\n", goType))
		for _, field := range structImpl.Fields {
			convert := ""
			// note these timestamps are always pointers when given to us in protobuf
			if field.Type == "time.Time" {
				convert = "timestamppb.New"
			}

			slice := ""
			if field.Repeated {
				slice = "Slice"
			}

			if isPodType(field.Type) {
				if convert != "" {
					// some weird shit to deref a ptr in case cause this convert func (our only one) returns a ptr
					sb.WriteString(fmt.Sprintf("         %s: %s(ent.%s),\n", field.Name, convert, field.Name))
				} else {
					sb.WriteString(fmt.Sprintf("         %s: ent.%s,\n", field.Name, field.Name))
				}
			} else {
				ptr := ""
				if field.Optional {
					ptr = "Ptr"
				}
				actualType := field.Type
				if field.ComputeSelector() != nil {
					actualType = fmt.Sprintf("converter%s", field.Type)
				}
				sb.WriteString(fmt.Sprintf("         %s: %sFromGo%s(ent.%s),\n", field.Name, actualType, ptr+slice, field.Name))
			}
		}
		sb.WriteString("     }\n")
		sb.WriteString("}\n\n")

		sb.WriteString(fmt.Sprintf("func %sFromPbSlice(ents []pb%s) ([]%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString(fmt.Sprintf("     var out []%s\n", goType))
		sb.WriteString("     for _, e := range ents {\n")
		sb.WriteString(fmt.Sprintf("         out = append(out, %sFromPb(e))\n", structImpl.Name))
		sb.WriteString("     }\n")
		sb.WriteString("     return out\n")
		sb.WriteString("}\n\n")

		sb.WriteString(fmt.Sprintf("func %sFromPbPtrSlice(ents []*pb%s) ([]*%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString(fmt.Sprintf("     var out []*%s\n", goType))
		sb.WriteString("     for _, e := range ents {\n")
		sb.WriteString(fmt.Sprintf("         out = append(out, %sFromPbPtr(e))\n", structImpl.Name))
		sb.WriteString("     }\n")
		sb.WriteString("     return out\n")
		sb.WriteString("}\n\n")

		sb.WriteString(fmt.Sprintf("func %sFromGoSlice(ents []%s) ([]pb%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString(fmt.Sprintf("     var out []pb%s\n", goType))
		sb.WriteString("     for _, e := range ents {\n")
		sb.WriteString(fmt.Sprintf("         out = append(out, %sFromGo(e))\n", structImpl.Name))
		sb.WriteString("     }\n")
		sb.WriteString("     return out\n")
		sb.WriteString("}\n\n")

		sb.WriteString(fmt.Sprintf("func %sFromGoPtrSlice(ents []*%s) ([]*pb%s) {\n", structImpl.Name, goType, goType))
		sb.WriteString(fmt.Sprintf("     var out []*pb%s\n", goType))
		sb.WriteString("     for _, e := range ents {\n")
		sb.WriteString(fmt.Sprintf("         out = append(out, %sFromGoPtr(e))\n", structImpl.Name))
		sb.WriteString("     }\n")
		sb.WriteString("     return out\n")
		sb.WriteString("}\n\n")

	}

	for pkg, sb := range pkgFiles {
		if _, err := os.Stat(fmt.Sprintf("converters/%s", pkg)); os.IsNotExist(err) {
			os.Mkdir(fmt.Sprintf("converters/%s", pkg), 0755)
		}
		WriteFile(fmt.Sprintf("converters/%s/struct.go", pkg), []byte(sb.String()))
	}
}
