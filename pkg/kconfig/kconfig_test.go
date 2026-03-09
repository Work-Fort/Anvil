// SPDX-License-Identifier: Apache-2.0
package kconfig

import (
	"strings"
	"testing"
)

func TestParseEnabled(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_FOO=y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("FOO")
	if !ok {
		t.Fatal("expected FOO to be found")
	}
	if v != "y" {
		t.Errorf("expected value 'y', got %q", v)
	}
}

func TestParseDisabled(t *testing.T) {
	c, err := Parse(strings.NewReader("# CONFIG_FOO is not set"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("FOO")
	if !ok {
		t.Fatal("expected FOO to be found")
	}
	if v != "n" {
		t.Errorf("expected value 'n', got %q", v)
	}
}

func TestParseValueOption(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_LOG_BUF_SHIFT=17"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("LOG_BUF_SHIFT")
	if !ok {
		t.Fatal("expected LOG_BUF_SHIFT to be found")
	}
	if v != "17" {
		t.Errorf("expected value '17', got %q", v)
	}
}

func TestParseHexValue(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_PHYSICAL_START=0x1000000"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("PHYSICAL_START")
	if !ok {
		t.Fatal("expected PHYSICAL_START to be found")
	}
	if v != "0x1000000" {
		t.Errorf("expected value '0x1000000', got %q", v)
	}
}

func TestParseStringValue(t *testing.T) {
	c, err := Parse(strings.NewReader(`CONFIG_DEFAULT_HOSTNAME="(none)"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("DEFAULT_HOSTNAME")
	if !ok {
		t.Fatal("expected DEFAULT_HOSTNAME to be found")
	}
	if v != `"(none)"` {
		t.Errorf("expected value '\"(none)\"', got %q", v)
	}
}

func TestSectionCommentsPassThrough(t *testing.T) {
	input := "# General setup\nCONFIG_FOO=y"
	c, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The comment should be preserved; only FOO should be an option.
	opts := c.List("")
	if len(opts) != 1 {
		t.Fatalf("expected 1 option, got %d", len(opts))
	}
	if opts[0].Name != "FOO" {
		t.Errorf("expected option name 'FOO', got %q", opts[0].Name)
	}

	// Write and verify the comment is still there.
	var buf strings.Builder
	if err := c.Write(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "# General setup") {
		t.Error("expected section comment to be preserved in output")
	}
}

func TestBlankLinesPreserved(t *testing.T) {
	input := "CONFIG_FOO=y\n\nCONFIG_BAR=y"
	c, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var buf strings.Builder
	if err := c.Write(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\n\n") {
		t.Error("expected blank line to be preserved in output")
	}
}

func TestGetExisting(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_SYSVIPC=y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("SYSVIPC")
	if !ok {
		t.Fatal("expected SYSVIPC to be found")
	}
	if v != "y" {
		t.Errorf("expected 'y', got %q", v)
	}
}

func TestGetNonexistent(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_FOO=y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := c.Get("NONEXISTENT")
	if ok {
		t.Error("expected NONEXISTENT to not be found")
	}
}

func TestSetDisabledToEnabled(t *testing.T) {
	c, err := Parse(strings.NewReader("# CONFIG_FOO is not set"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.Set("FOO", "y"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("FOO")
	if !ok {
		t.Fatal("expected FOO to be found")
	}
	if v != "y" {
		t.Errorf("expected 'y', got %q", v)
	}
	var buf strings.Builder
	if err := c.Write(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "CONFIG_FOO=y") {
		t.Errorf("expected output to contain 'CONFIG_FOO=y', got:\n%s", buf.String())
	}
}

func TestSetEnabledToDisabled(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_FOO=y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.Set("FOO", "n"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("FOO")
	if !ok {
		t.Fatal("expected FOO to be found")
	}
	if v != "n" {
		t.Errorf("expected 'n', got %q", v)
	}
	var buf strings.Builder
	if err := c.Write(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "# CONFIG_FOO is not set") {
		t.Errorf("expected output to contain '# CONFIG_FOO is not set', got:\n%s", buf.String())
	}
}

func TestSetModuleReturnsError(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_FOO=y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.Set("FOO", "m"); err == nil {
		t.Error("expected error when setting value to 'm'")
	}
}

func TestSetNewOptionAppends(t *testing.T) {
	c, err := Parse(strings.NewReader("CONFIG_FOO=y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.Set("BAR", "y"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := c.Get("BAR")
	if !ok {
		t.Fatal("expected BAR to be found")
	}
	if v != "y" {
		t.Errorf("expected 'y', got %q", v)
	}
	opts := c.List("")
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	// BAR should come after FOO since it was appended.
	if opts[0].Name != "FOO" || opts[1].Name != "BAR" {
		t.Errorf("expected order [FOO, BAR], got [%s, %s]", opts[0].Name, opts[1].Name)
	}
}

func TestRoundtrip(t *testing.T) {
	input := `#
# Automatically generated file; DO NOT EDIT.
# Linux/x86_64 6.12.6 Kernel Configuration
#

#
# General setup
#
CONFIG_SYSVIPC=y
CONFIG_POSIX_MQUEUE=y
# CONFIG_USELIB is not set
CONFIG_DEFAULT_HOSTNAME="(none)"
CONFIG_LOG_BUF_SHIFT=17
CONFIG_PHYSICAL_START=0x1000000

#
# Processor type and features
#
CONFIG_SMP=y
# CONFIG_X86_MPPARSE is not set
CONFIG_NR_CPUS=256
`
	c, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var buf strings.Builder
	if err := c.Write(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != input {
		t.Errorf("roundtrip mismatch.\nexpected:\n%s\ngot:\n%s", input, buf.String())
	}
}

func TestListAll(t *testing.T) {
	input := "CONFIG_FOO=y\n# CONFIG_BAR is not set\nCONFIG_BAZ=17"
	c, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts := c.List("")
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
	expected := []Option{
		{Name: "FOO", Value: "y"},
		{Name: "BAR", Value: "n"},
		{Name: "BAZ", Value: "17"},
	}
	for i, o := range opts {
		if o.Name != expected[i].Name || o.Value != expected[i].Value {
			t.Errorf("option %d: expected %+v, got %+v", i, expected[i], o)
		}
	}
}

func TestListWithFilter(t *testing.T) {
	input := "CONFIG_NET_CORE=y\nCONFIG_SMP=y\nCONFIG_NET_SCHED=y"
	c, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts := c.List("net")
	if len(opts) != 2 {
		t.Fatalf("expected 2 options matching 'net', got %d", len(opts))
	}
	for _, o := range opts {
		if !strings.Contains(strings.ToUpper(o.Name), "NET") {
			t.Errorf("option %q should not match filter 'net'", o.Name)
		}
	}
}

func TestDiff(t *testing.T) {
	inputA := "CONFIG_FOO=y\nCONFIG_BAR=y\nCONFIG_BAZ=17"
	inputB := "CONFIG_FOO=n\nCONFIG_BAZ=17\nCONFIG_QUX=y"

	a, err := Parse(strings.NewReader(inputA))
	if err != nil {
		t.Fatalf("unexpected error parsing A: %v", err)
	}
	b, err := Parse(strings.NewReader(inputB))
	if err != nil {
		t.Fatalf("unexpected error parsing B: %v", err)
	}

	diffs := Diff(a, b)

	// Expect 3 diffs sorted by name: BAR (removed), FOO (changed), QUX (added).
	if len(diffs) != 3 {
		t.Fatalf("expected 3 diffs, got %d: %+v", len(diffs), diffs)
	}

	// BAR: removed
	if diffs[0].Name != "BAR" || diffs[0].Type != DiffRemoved {
		t.Errorf("expected BAR removed, got %+v", diffs[0])
	}
	if diffs[0].ValueA != "y" || diffs[0].ValueB != "" {
		t.Errorf("BAR diff values wrong: %+v", diffs[0])
	}

	// FOO: changed y -> n
	if diffs[1].Name != "FOO" || diffs[1].Type != DiffChanged {
		t.Errorf("expected FOO changed, got %+v", diffs[1])
	}
	if diffs[1].ValueA != "y" || diffs[1].ValueB != "n" {
		t.Errorf("FOO diff values wrong: %+v", diffs[1])
	}

	// QUX: added
	if diffs[2].Name != "QUX" || diffs[2].Type != DiffAdded {
		t.Errorf("expected QUX added, got %+v", diffs[2])
	}
	if diffs[2].ValueA != "" || diffs[2].ValueB != "y" {
		t.Errorf("QUX diff values wrong: %+v", diffs[2])
	}
}
