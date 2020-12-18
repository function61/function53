package main

import (
	"strings"
	"testing"

	"github.com/function61/gokit/testing/assert"
)

func TestFoo(t *testing.T) {
	testBlocklistContent := `
########## Blacklist from https://easylist-downloads.adblockplus.org/antiadblockfilters.txt ##########

ads.com
ads.example.co.uk


########## Blacklist from https://pgl.yoyo.org/adservers/serverlist.php ##########

# Ignored duplicates: 219

# Ignored entries due to the whitelist: 2

# whole TLD blocked
addomain

`
	listRef, err := blocklistParse(strings.NewReader(testBlocklistContent))
	assert.Assert(t, err == nil)

	list := *listRef
	assert.Assert(t, len(list) == 3)

	assert.Assert(t, list.Has("ads.com"))

	assert.Assert(t, list.Has("www.ads.example.co.uk"))
	assert.Assert(t, list.Has("ads.example.co.uk"))
	assert.Assert(t, !list.Has("example.co.uk"))

	assert.Assert(t, list.Has("blocked.everything.addomain"))
	assert.Assert(t, list.Has("www.addomain"))
	assert.Assert(t, list.Has("addomain"))

	assert.Assert(t, !list.Has("joonas.fi"))
}
