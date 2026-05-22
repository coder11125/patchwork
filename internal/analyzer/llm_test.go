package analyzer

import (
	"testing"
)

func TestParseLLMResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		pkgName     string
		version     string
		wantCount   int
		wantErr     bool
	}{
		{
			name:        "valid JSON array with changes",
			response:    `[{"id":"bc-1","package_name":"testpkg","version":"2.0.0","description":"breaking change","severity":"high","affected_apis":["Foo"],"migration_hint":"update","source_url":""}]`,
			pkgName:     "testpkg",
			version:     "2.0.0",
			wantCount:   1,
			wantErr:     false,
		},
		{
			name:        "empty array",
			response:    `[]`,
			pkgName:     "testpkg",
			version:     "2.0.0",
			wantCount:   0,
			wantErr:     false,
		},
		{
			name:        "text wrapped around JSON",
			response:    `Here is the analysis:\n\n[{"id":"bc-1","package_name":"testpkg","version":"2.0.0","description":"breaking change","severity":"high"}]\n\nHope this helps.`,
			pkgName:     "testpkg",
			version:     "2.0.0",
			wantCount:   1,
			wantErr:     false,
		},
		{
			name:        "no JSON found",
			response:    "no breaking changes detected",
			pkgName:     "testpkg",
			version:     "2.0.0",
			wantCount:   0,
			wantErr:     true,
		},
		{
			name:        "empty response",
			response:    "",
			pkgName:     "testpkg",
			version:     "2.0.0",
			wantCount:   0,
			wantErr:     true,
		},
		{
			name:        "fills in missing fields",
			response:    `[{"description":"breaking change"}]`,
			pkgName:     "testpkg",
			version:     "2.0.0",
			wantCount:   1,
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changes, err := parseLLMResponse(tc.response, tc.pkgName, tc.version)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none, changes: %+v", changes)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(changes) != tc.wantCount {
				t.Errorf("expected %d changes, got %d", tc.wantCount, len(changes))
			}
			if len(changes) > 0 {
				if changes[0].PackageName == "" {
					t.Error("expected PackageName to be filled")
				}
				if changes[0].Version == "" {
					t.Error("expected Version to be filled")
				}
			}
		})
	}
}
