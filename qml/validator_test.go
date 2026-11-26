package qml

import (
	"fmt"
	"testing"
)

func TestValidateQML(t *testing.T) {
	tests := []struct {
		name    string
		qml     string
		wantErr bool
	}{
		{
			name: "valid VStack layout",
			qml: `Window {
				VStack {
					Label { text: "Hello" }
				}
			}`,
			wantErr: false,
		},
		{
			name: "valid Column layout",
			qml: `Window {
				Column {
					Label { text: "Hello" }
				}
			}`,
			wantErr: false,
		},
		{
			name: "invalid direct child with x,y",
			qml: `Window {
				Label {
					x: 10
					y: 10
					text: "Hello"
				}
			}`,
			wantErr: true,
		},
		{
			name: "invalid multiple direct children with x,y",
			qml: `Window {
				Label {
					x: 10
					y: 10
					text: "Hello"
				}
				Label {
					x: 20
					y: 20
					text: "World"
				}
			}`,
			wantErr: true,
		},
		{
			name: "valid nested layout",
			qml: `Window {
				VStack {
					Label { x: 10; y: 10; text: "Hello" }
					HStack {
						Label { text: "World" }
					}
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := ParseQML(tt.qml)
			if err != nil {
				t.Fatalf("ParseQML failed: %v", err)
			}

			fmt.Printf("Test %s: obj.Type=%s, children=%d\n", tt.name, obj.Type, len(obj.Children))
			for i, child := range obj.Children {
				fmt.Printf("  child[%d]: type=%s, props=%d\n", i, child.Type, len(child.Properties))
				for j, prop := range child.Properties {
					fmt.Printf("    prop[%d]: name=%s\n", j, prop.Name)
				}
			}

			err = ValidateQML(obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
