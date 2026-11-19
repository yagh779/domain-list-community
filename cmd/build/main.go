// Command build repackages a minimal dlc.dat for sing-geosite.
//
// It takes the latest v2fly/domain-list-community dlc.dat and produces a new
// dlc.dat containing only the two entries sing-geosite needs, trimmed to just
// the domains that actually feed its two tags:
//
//   - "cn":             v2fly's cn entry plus the cn-overseas overlay, keeping
//                       only the domains tagged @!cn (=> tag cn@!cn)
//   - "geolocation-!cn": v2fly's geolocation-!cn entry, keeping only the
//                       domains tagged @cn (=> tag geolocation-!cn@cn)
//
// Every other domain is dropped, so the published dlc.dat stays tiny while
// the two tags sing-geosite builds are byte-for-byte identical.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

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

func key(t routercommon.Domain_Type, value string) string {
	return fmt.Sprintf("%d\x00%s", int(t), value)
}

func dslType(name string) (routercommon.Domain_Type, bool) {
	switch name {
	case "domain":
		return routercommon.Domain_RootDomain, true
	case "full":
		return routercommon.Domain_Full, true
	case "keyword":
		return routercommon.Domain_Plain, true
	case "regexp":
		return routercommon.Domain_Regex, true
	default:
		return 0, false
	}
}

// parseOverseas reads data/cn-overseas in the v2fly DSL, e.g.
//
//	domain:example.com @!cn
//	full:www.example.com @!cn
func parseOverseas(path string) []*routercommon.Domain {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	var out []*routercommon.Domain
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		i := strings.Index(fields[0], ":")
		if i < 0 {
			continue
		}
		typ, ok := dslType(fields[0][:i])
		if !ok {
			continue
		}
		d := &routercommon.Domain{Type: typ, Value: fields[0][i+1:]}
		for _, a := range fields[1:] {
			a = strings.TrimPrefix(a, "@")
			if a != "" {
				d.Attribute = append(d.Attribute, &routercommon.Domain_Attribute{Key: a})
			}
		}
		if len(d.Attribute) == 0 {
			d.Attribute = []*routercommon.Domain_Attribute{{Key: "!cn"}}
		}
		out = append(out, d)
	}
	return out
}

// trimByAttr keeps only the domains of entry that carry attr.
func trimByAttr(entry *routercommon.GeoSite, attr string) *routercommon.GeoSite {
	trimmed := &routercommon.GeoSite{CountryCode: entry.CountryCode}
	for _, d := range entry.Domain {
		if hasAttr(d, attr) {
			trimmed.Domain = append(trimmed.Domain, d)
		}
	}
	return trimmed
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: build <v2fly-dlc.dat> <cn-overseas-file> <out-dlc.dat>")
		os.Exit(2)
	}
	v2 := load(os.Args[1])
	overseas := parseOverseas(os.Args[2])

	cn := findEntry(v2, "cn")
	if cn == nil {
		fmt.Fprintln(os.Stderr, "v2fly dlc.dat has no cn entry")
		os.Exit(1)
	}
	geo := findEntry(v2, "geolocation-!cn")
	if geo == nil {
		fmt.Fprintln(os.Stderr, "v2fly dlc.dat has no geolocation-!cn entry")
		os.Exit(1)
	}

	cnClone := proto.Clone(cn).(*routercommon.GeoSite)
	index := make(map[string]*routercommon.Domain, len(cnClone.Domain))
	for _, d := range cnClone.Domain {
		index[key(d.Type, d.Value)] = d
	}
	for _, od := range overseas {
		if ex, ok := index[key(od.Type, od.Value)]; ok {
			if !hasAttr(ex, "!cn") {
				ex.Attribute = append(ex.Attribute, &routercommon.Domain_Attribute{Key: "!cn"})
			}
			continue
		}
		cnClone.Domain = append(cnClone.Domain, od)
		index[key(od.Type, od.Value)] = od
	}

	// Keep only the domains that actually feed the two sing-geosite tags.
	cnTrimmed := trimByAttr(cnClone, "!cn")
	geoTrimmed := trimByAttr(proto.Clone(geo).(*routercommon.GeoSite), "cn")

	out := &routercommon.GeoSiteList{
		Entry: []*routercommon.GeoSite{
			cnTrimmed,
			geoTrimmed,
		},
	}
	b, err := proto.Marshal(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(1)
	}
	if err = os.WriteFile(os.Args[3], b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s: cn(!cn)=%d domains, geolocation-!cn(cn)=%d domains\n",
		os.Args[3], len(cnTrimmed.Domain), len(geoTrimmed.Domain))
}
