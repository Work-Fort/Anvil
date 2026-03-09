// SPDX-License-Identifier: Apache-2.0
package kconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Option represents a single kernel config option.
type Option struct {
	Name  string // Without CONFIG_ prefix (e.g. "SYSVIPC")
	Value string // "y", "n", or arbitrary string
}

// DiffType describes how an option differs between two configs.
type DiffType string

const (
	DiffAdded   DiffType = "added"   // in B but not A
	DiffRemoved DiffType = "removed" // in A but not B
	DiffChanged DiffType = "changed" // different value
)

// DiffEntry represents a single difference between two configs.
type DiffEntry struct {
	Name   string
	Type   DiffType
	ValueA string // empty if added
	ValueB string // empty if removed
}

type lineKind int

const (
	lineComment lineKind = iota
	lineOption
	lineBlank
)

type configLine struct {
	kind   lineKind
	raw    string // original text for comments/blanks
	option string // option name without CONFIG_ prefix
	value  string // option value
}

// Config holds a parsed kernel .config file, preserving structure.
type Config struct {
	lines []configLine
}

// Parse reads a kernel .config from the given reader.
func Parse(r io.Reader) (*Config, error) {
	c := &Config{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		c.lines = append(c.lines, parseLine(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	return c, nil
}

// ParseFile reads a kernel .config from the given file path.
func ParseFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

func parseLine(line string) configLine {
	// Blank line.
	if line == "" {
		return configLine{kind: lineBlank, raw: line}
	}

	// "# CONFIG_FOO is not set" — disabled option.
	if strings.HasPrefix(line, "# CONFIG_") && strings.HasSuffix(line, " is not set") {
		name := line[len("# CONFIG_") : len(line)-len(" is not set")]
		return configLine{kind: lineOption, raw: line, option: name, value: "n"}
	}

	// "CONFIG_FOO=<value>" — enabled option.
	if strings.HasPrefix(line, "CONFIG_") {
		if idx := strings.IndexByte(line, '='); idx >= 0 {
			name := line[len("CONFIG_"):idx]
			value := line[idx+1:]
			return configLine{kind: lineOption, raw: line, option: name, value: value}
		}
	}

	// Everything else is a comment (section comments, auto-generated header, etc.).
	return configLine{kind: lineComment, raw: line}
}

// Get returns the value of the named option and whether it was found.
// The name should not include the CONFIG_ prefix.
func (c *Config) Get(name string) (string, bool) {
	for _, l := range c.lines {
		if l.kind == lineOption && l.option == name {
			return l.value, true
		}
	}
	return "", false
}

// Set sets the value of the named option. The name should not include the
// CONFIG_ prefix. If the option already exists, it is updated in place.
// If not, a new line is appended. Setting a value to "m" returns an error
// because module support is not applicable.
func (c *Config) Set(name, value string) error {
	if value == "m" {
		return fmt.Errorf("module value 'm' is not supported for CONFIG_%s", name)
	}

	for i, l := range c.lines {
		if l.kind == lineOption && l.option == name {
			c.lines[i].value = value
			c.lines[i].raw = formatOption(name, value)
			return nil
		}
	}

	// Option not found — append.
	c.lines = append(c.lines, configLine{
		kind:   lineOption,
		raw:    formatOption(name, value),
		option: name,
		value:  value,
	})
	return nil
}

func formatOption(name, value string) string {
	if value == "n" {
		return fmt.Sprintf("# CONFIG_%s is not set", name)
	}
	return fmt.Sprintf("CONFIG_%s=%s", name, value)
}

// List returns all options in the config. If filter is non-empty, only
// options whose name contains the filter string (case-insensitive) are
// returned.
func (c *Config) List(filter string) []Option {
	var opts []Option
	filterUpper := strings.ToUpper(filter)
	for _, l := range c.lines {
		if l.kind != lineOption {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToUpper(l.option), filterUpper) {
			continue
		}
		opts = append(opts, Option{Name: l.option, Value: l.value})
	}
	return opts
}

// Write writes the config to w, preserving original structure.
func (c *Config) Write(w io.Writer) error {
	bw := bufio.NewWriter(w)
	for i, l := range c.lines {
		var line string
		if l.kind == lineOption {
			line = formatOption(l.option, l.value)
		} else {
			line = l.raw
		}
		if _, err := bw.WriteString(line); err != nil {
			return err
		}
		// Write newline after every line, including the last.
		if i < len(c.lines) {
			if _, err := bw.WriteString("\n"); err != nil {
				return err
			}
		}
	}
	return bw.Flush()
}

// WriteFile writes the config to the given file path.
func (c *Config) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()
	if err := c.Write(f); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// Diff compares two configs and returns the differences. Entries are sorted
// by option name for deterministic output.
func Diff(a, b *Config) []DiffEntry {
	aOpts := make(map[string]string)
	bOpts := make(map[string]string)

	for _, l := range a.lines {
		if l.kind == lineOption {
			aOpts[l.option] = l.value
		}
	}
	for _, l := range b.lines {
		if l.kind == lineOption {
			bOpts[l.option] = l.value
		}
	}

	var diffs []DiffEntry

	// Removed or changed.
	for name, va := range aOpts {
		vb, ok := bOpts[name]
		if !ok {
			diffs = append(diffs, DiffEntry{Name: name, Type: DiffRemoved, ValueA: va})
		} else if va != vb {
			diffs = append(diffs, DiffEntry{Name: name, Type: DiffChanged, ValueA: va, ValueB: vb})
		}
	}

	// Added.
	for name, vb := range bOpts {
		if _, ok := aOpts[name]; !ok {
			diffs = append(diffs, DiffEntry{Name: name, Type: DiffAdded, ValueB: vb})
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Name < diffs[j].Name
	})

	return diffs
}
