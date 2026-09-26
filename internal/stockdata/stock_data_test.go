package stockdata

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const kobetsuFixtureHTML = `<!doctype html>
<html><body>
<div id="kobetsu_left">
<dl>
  <dt>前日終値</dt>
  <dd class="floatr">3,573.0&nbsp;(<time datetime="2026-09-24">09/24</time>)</dd>
</dl>
<h2><time datetime="2026-09-25">09月25日</time></h2>
<table>
  <tbody>
    <tr><th scope='row'>始値</th><td>3,610.0</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T09:00+09:00">09:00</time>)</td></tr>
    <tr><th scope='row'>高値</th><td>3,719.0</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T13:41+09:00">13:41</time>)</td></tr>
    <tr><th scope='row'>安値</th><td>3,608.0</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T09:00+09:00">09:00</time>)</td></tr>
    <tr><th scope='row'>終値</th><td>3,714.0</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T15:30+09:00">15:30</time>)</td></tr>
  </tbody>
</table>
</div>
</body></html>`

func TestOhlcFromKobetsuTable(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(kobetsuFixtureHTML))
	if err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	open, high, low, close := ohlcFromKobetsuTable(doc)

	if open != "3,610.0" {
		t.Errorf("open = %q, want %q", open, "3,610.0")
	}
	if high != "3,719.0" {
		t.Errorf("high = %q, want %q", high, "3,719.0")
	}
	if low != "3,608.0" {
		t.Errorf("low = %q, want %q", low, "3,608.0")
	}
	if close != "3,714.0" {
		t.Errorf("close = %q, want %q", close, "3,714.0")
	}
}
