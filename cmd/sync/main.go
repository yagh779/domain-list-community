// Command sync provides change detection and data extraction helpers for CI.
//
// It reads v2fly-format dlc.dat (GeoSiteList protobuf) and either
//   - prints a stable fingerprint of one tag's domains (hashtag), or
//   - rewrites data/cn-overseas from a source cn@!cn subset.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  sync hashtag <dlc.dat> <code@attr>")
	fmt.Fprintln(os.Stderr, "  sync extract-cn-overseas <dlc.dat> <out-file>")
	os.Exit(2)
}

func load(path string) *routercommon.GeoSiteList {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	list := &routercommon.GeoSiteList{}
	if err = proto.Unmarshal(b, list); err != nil {
		fmt.Fprintln(os.Stderr, "unmarshal:", err)
		os.Exit(1)
	}
	return list
}

func findEntry(list *routercommon.GeoSiteList, code string) *routercommon.GeoSite {
	for _, e := range list.Entry {
		if strings.EqualFold(e.CountryCode, code) {
			return e
		}
	}
	return nil
}

func hasAttr(d *routercommon.Domain, attr string) bool {
	for _, a := range d.Attribute {
		if a.Key == attr {
			return true
		}
	}
	return false
}

func splitTag(tag string) (code, attr string, ok bool) {
	i := strings.Index(tag, "@")
	if i <= 0 || i == len(tag)-1 {
		return "", "", false
	}
	return tag[:i], tag[i+1:], true
}

func typeName(t routercommon.Domain_Type) string {
	switch t {
	case routercommon.Domain_Plain:
		return "keyword"
	case routercommon.Domain_Regex:
		return "regex"
	case routercommon.Domain_RootDomain:
		return "domain"
	case routercommon.Domain_Full:
		return "full"
	default:
		return fmt.Sprintf("type%d", int(t))
	}
}

func dslPrefix(t routercommon.Domain_Type) string {
	switch t {
	case routercommon.Domain_Plain:
		return "keyword:"
	case routercommon.Domain_Regex:
		return "regexp:"
	case routercommon.Domain_RootDomain:
		return "domain:"
	case routercommon.Domain_Full:
		return "full:"
	default:
		return ""
	}
}

func collectAttr(list *routercommon.GeoSiteList, code, attr string) []*routercommon.Domain {
	e := findEntry(list, code)
	if e == nil {
		fmt.Fprintf(os.Stderr, "entry %q not found\n", code)
		os.Exit(1)
	}
	var out []*routercommon.Domain
	for _, d := range e.Domain {
		if hasAttr(d, attr) {
			out = append(out, d)
		}
	}
	return out
}

func dedupe(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := lines[:1]
	for _, l := range lines[1:] {
		if l != out[len(out)-1] {
			out = append(out, l)
		}
	}
	return out
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "hashtag":
		if len(os.Args) != 4 {
			usage()
		}
		list := load(os.Args[2])
		code, attr, ok := splitTag(os.Args[3])
		if !ok {
			fmt.Fprintln(os.Stderr, "bad tag:", os.Args[3])
			os.Exit(2)
		}
		domains := collectAttr(list, code, attr)
		lines := make([]string, 0, len(domains))
		for _, d := range domains {
			lines = append(lines, typeName(d.Type)+"\x00"+d.Value)
		}
		sort.Strings(lines)
		lines = dedupe(lines)
		h := sha256.New()
		for _, l := range lines {
			h.Write([]byte(l))
			h.Write([]byte{'\n'})
		}
		fmt.Println(hex.EncodeToString(h.Sum(nil)))
		fmt.Fprintf(os.Stderr, "tag %s: %d raw domains, %d items\n", os.Args[3], len(domains), len(lines))
	case "extract-cn-overseas":
		if len(os.Args) != 4 {
			usage()
		}
		list := load(os.Args[2])
		domains := collectAttr(list, "cn", "!cn")
		rows := make([]string, 0, len(domains))
		for _, d := range domains {
			p := dslPrefix(d.Type)
			if p == "" {
				continue
			}
			rows = append(rows, p+d.Value+" @!cn")
		}
		sort.Strings(rows)
		var b strings.Builder
		b.WriteString("# Domains of Chinese services that have no access point in\n")
		b.WriteString("# mainland China (overseas/CDN nodes), tagged @!cn.\n")
		b.WriteString("# Auto-generated from nekolsd/domain-list-community cn@!cn.\n")
		b.WriteString("# Included by data/cn so that sing-box tag cn@!cn can be built.\n\n")
		for _, r := range rows {
			b.WriteString(r)
			b.WriteString("\n")
		}
		if err := os.WriteFile(os.Args[3], []byte(b.String()), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %d domains to %s\n", len(rows), os.Args[3])
	default:
		usage()
	}
}
