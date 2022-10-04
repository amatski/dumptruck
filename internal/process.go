package internal

import (
	"fmt"
	"go/ast"
	"log"
)

func ProcessFields(fields []*ast.Field, pkgName string, path Path) []*Field {
	outFields := []*Field{}
	for _, field := range fields {
		switch field.Type.(type) {
		case *ast.Ident:
			i := field.Type.(*ast.Ident)
			fieldType := i.Name

			// If there are no names then we just use a default field an assume a name that is the type
			if len(field.Names) == 0 {
				outFields = append(outFields, &Field{
					Path: path,
					Name: fieldType,
					Type: fieldType,
				})
			}

			for _, name := range field.Names {
				outFields = append(outFields, &Field{
					Path: path,
					Name: name.Name,
					Type: fieldType,
				})
			}
		case *ast.ArrayType:
			i := field.Type.(*ast.ArrayType)
			if si, ok := i.Elt.(*ast.Ident); ok {
				name := si.Name
				if len(field.Names) > 0 {
					name = field.Names[0].Name
				}
				outFields = append(outFields, &Field{
					Path:     path,
					Name:     name,
					Type:     si.Name,
					Repeated: true,
				})
			} else if si, ok := i.Elt.(*ast.SelectorExpr); ok {
				if si2, ok := si.X.(*ast.Ident); ok {
					fieldType := si.Sel.Name
					outFields = append(outFields, &Field{
						Path:     path,
						Repeated: true,
						Selector: true,
						Package:  si2.Name,
						Name:     field.Names[0].Name,
						Type:     si2.Name + "." + fieldType,
					})
				} else {
					panic("missing parser for array of selector")
				}
			} else if i, ok := i.Elt.(*ast.StarExpr); ok {
				if si, ok := i.X.(*ast.Ident); ok {
					fieldType := si.Name
					name := "unknown_name"
					if len(field.Names) > 0 {
						name = field.Names[0].Name
					}
					outFields = append(outFields, &Field{
						Path:     path,
						Repeated: true,
						Optional: true,
						Name:     name,
						Type:     fieldType,
					})
				} else if si, ok := i.X.(*ast.SelectorExpr); ok {
					if si2, ok := si.X.(*ast.Ident); ok {
						fieldType := si.Sel.Name
						name := si2.Name // package prefix selector
						if len(field.Names) > 0 {
							name = field.Names[0].Name
						}
						outFields = append(outFields, &Field{
							Path:     path,
							Repeated: true,
							Selector: true,
							Package:  si2.Name,
							Optional: true,
							Name:     name,
							Type:     si2.Name + "." + fieldType,
						})
					} else if _, ok := si.X.(*ast.SelectorExpr); ok {
						// array of non ptr type
						panic("WTF")

					} else {
						panic("missing selector")
					}
				} else if _, ok := i.X.(*ast.MapType); !ok {
					panic("not a map")
				}
			} else {
				panic("missing")
			}
		case *ast.SelectorExpr:
			selector := field.Type.(*ast.SelectorExpr)
			if pkgtag, ok := selector.X.(*ast.Ident); ok {
				//pkgtag.Name
				outFields = append(outFields, &Field{
					Path:     path,
					Selector: true,
					Name:     field.Names[0].Name,
					Type:     fmt.Sprintf("%s.%s", pkgtag.Name, selector.Sel.Name),
				})
			} else {
				panic("not valid selector expr")
			}

		case *ast.StarExpr:
			i := field.Type.(*ast.StarExpr)
			if si, ok := i.X.(*ast.Ident); ok {
				fieldType := si.Name
				outField := &Field{
					Path:     path,
					Optional: true,
					Name:     "",
					Type:     fieldType,
				}

				if len(field.Names) > 0 {
					outField.Name = field.Names[0].Name
				}
				outFields = append(outFields, outField)
			} else if si, ok := i.X.(*ast.SelectorExpr); ok {
				if si2, ok := si.X.(*ast.Ident); ok {
					fieldType := si.Sel.Name
					name := si2.Name
					if len(field.Names) > 0 {
						name = field.Names[0].Name
					}
					outFields = append(outFields, &Field{
						Path:     path,
						Package:  si2.Name,
						Selector: true,
						Optional: true,
						Name:     name,
						Type:     si2.Name + "." + fieldType,
					})
				} else {
					panic("missing x")
				}
			} else if _, ok := i.X.(*ast.MapType); ok {
				/*outFields = append(outFields, Field{
					FilePath: filePath,
					Path:     path,
					Package:  pkgName,
					Optional: true,
					Name:     field.Names[0].Name,
					Type:     "unknown_map",
				})*/
				log.Println("Ignoring map", path, pkgName, field.Names[0].Name)
			} else {
				// The only thing we can't parse in ChatEntry.Key
				log.Println("Missing parser for this", fields)
				name := "unknown_name"
				if len(field.Names) > 0 {
					name = field.Names[0].Name
				}
				outFields = append(outFields, &Field{
					Path:     path,
					Package:  pkgName,
					Optional: true,
					Name:     name,
					Type:     "unknown",
				})
			}
		case *ast.InterfaceType:
			name := "unknown_name"
			if len(field.Names) > 0 {
				name = field.Names[0].Name
			}
			outFields = append(outFields, &Field{
				Path:     path,
				Optional: false,
				Package:  pkgName,
				Name:     name,
				Type:     "interface",
			})
		default:
			outFields = append(outFields, &Field{
				Path: path,
				Name: field.Names[0].Name,
				Type: field.Names[0].Name,
			})
		}
	}
	return outFields
}
