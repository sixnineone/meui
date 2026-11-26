package qml

import (
	"strings"
	"testing"
)

func TestValidateBraces(t *testing.T) {
	tests := []struct {
		name        string
		qml         string
		wantErr     bool
		errContains string
	}{
		{
			name: "balanced braces",
			qml: `Window {
				VStack {
					Label { text: "Hello" }
				}
			}`,
			wantErr: false,
		},
		{
			name: "missing closing brace",
			qml: `Window {
				VStack {
					Label { text: "Hello" }
				}`,
			wantErr:     true,
			errContains: "unmatched",
		},
		{
			name: "extra closing brace",
			qml: `Window {
				VStack {
					Label { text: "Hello" }
				}
			}}`,
			wantErr:     true,
			errContains: "unexpected",
		},
		{
			name: "multiple missing braces",
			qml: `Window {
				VStack {
					Label { text: "Hello" }`,
			wantErr:     true,
			errContains: "2 unmatched",
		},
		{
			name: "brace inside string ignored",
			qml: `Window {
				Label { text: "Hello { World }" }
			}`,
			wantErr: false,
		},
		{
			name: "brace in comment ignored",
			qml: `Window {
				// This is a comment { with braces }
				Label { text: "Hello" }
			}`,
			wantErr: false,
		},
		{
			name: "brace in block comment ignored",
			qml: `Window {
				/* Block comment { with braces } */
				Label { text: "Hello" }
			}`,
			wantErr: false,
		},
		{
			name: "deep nesting balanced",
			qml: `Window {
				VStack {
					HStack {
						Grid {
							Rectangle { color: "red" }
						}
					}
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseQML(tt.qml)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseQML() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}
