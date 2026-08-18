package cmd

import "testing"

func TestValidateInitTargetFlags(t *testing.T) {
	tests := []struct {
		name        string
		site        string
		project     string
		application string
		wantError   bool
	}{
		{name: "site", site: "site-1"},
		{name: "docker project", project: "project-1"},
		{name: "docker application", project: "project-1", application: "app-1"},
		{name: "site and project", site: "site-1", project: "project-1", wantError: true},
		{name: "site and application", site: "site-1", application: "app-1", wantError: true},
		{name: "application without project", application: "app-1", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateInitTargetFlags(test.site, test.project, test.application)
			if (err != nil) != test.wantError {
				t.Fatalf("validateInitTargetFlags() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}
