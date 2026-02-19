// SPDX-License-Identifier: Apache-2.0
package init_test

import (
	"testing"

	initpkg "github.com/Work-Fort/Anvil/pkg/init"
)

func TestInitSettings_DefaultValues(t *testing.T) {
	settings := initpkg.InitSettings{}

	if settings.KeyName != "" {
		t.Errorf("expected empty KeyName, got %s", settings.KeyName)
	}
}

func TestInitSettings_SetValues(t *testing.T) {
	settings := initpkg.InitSettings{
		KeyName:       "Test Kernels",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "binary",
	}

	if settings.KeyName != "Test Kernels" {
		t.Errorf("KeyName = %s, want Test Kernels", settings.KeyName)
	}

	if settings.KeyFormat != "armored" {
		t.Errorf("KeyFormat = %s, want armored", settings.KeyFormat)
	}

	if settings.HistoryFormat != "binary" {
		t.Errorf("HistoryFormat = %s, want binary", settings.HistoryFormat)
	}
}
