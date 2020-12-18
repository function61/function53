package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/function61/gokit/net/http/ezhttp"
	"github.com/function61/gokit/os/atomicfilewrite"
	"github.com/function61/gokit/os/osutil"
	"github.com/miekg/dns"
)

const (
	blocklistFilename = "blocklist.txt"
)

type Blocklist map[string]bool

func (b Blocklist) Has(hostname string) bool {
	parts := dns.SplitDomainName(hostname)
	partLen := len(parts)

	/*	ads.adsprovider.co.uk =>

		is "ads.adsprovider.co.uk" on the list?
		is "adsprovider.co.uk" on the list?
		is "co.uk" on the list?
		is "uk" on the list?
	*/
	for i := 0; i < partLen; i++ {
		test := strings.Join(parts[i:partLen], ".")

		if _, has := b[test]; has {
			return true
		}
	}

	return false
}

var blocklistItemParseRe = regexp.MustCompile("^[^#]+")

func blocklistParse(content io.Reader) (*Blocklist, error) {
	list := Blocklist{}

	lineScanner := bufio.NewScanner(content)
	for lineScanner.Scan() {
		line := lineScanner.Text()
		if !blocklistItemParseRe.MatchString(line) {
			continue
		}

		list[line] = true
	}
	if err := lineScanner.Err(); err != nil {
		return nil, err
	}

	return &list, nil
}

func blocklistLoadFromDisk() (*Blocklist, error) {
	file, err := os.Open(blocklistFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return blocklistParse(file)
}

func blocklistExists() (bool, error) {
	return osutil.Exists(blocklistFilename)
}

// atomically updates a blocklist to disk
// source: https://github.com/jedisct1/dnscrypt-proxy/wiki/Public-blacklists
// "Updated daily"
func blocklistUpdate() error {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()
	res, err := ezhttp.Get(
		ctx,
		"https://download.dnscrypt.info/blacklists/domains/mybase.txt",
		ezhttp.Header("User-Agent", "github.com/function61/function53"))
	if err != nil {
		return err
	}

	return atomicfilewrite.Write(blocklistFilename, func(blocklist io.Writer) error {
		_, err := io.Copy(blocklist, res.Body)
		return err
	})
}
