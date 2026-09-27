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

// Reproduces a real Kabutan page shape (ticker 3246) where the PTS (after-hours)
// section further down #kobetsu_left has its own 始値/高値/安値/現在値 table filled
// with "－" placeholders for stocks with no PTS trading. That nested table must not
// clobber the real OHLC values captured earlier in the document.
const kobetsuFixtureWithPtsHTML = `<!doctype html>
<html><body>
<div id="kobetsu_left">
<dl>
  <dt>前日終値</dt>
  <dd class="floatr">695&nbsp;(<time datetime="2026-09-24">09/24</time>)</dd>
</dl>
<h2><time datetime="2026-09-25">09月25日</time></h2>
<table>
  <tbody>
    <tr><th scope='row'>始値</th><td>695</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T09:00+09:00">09:00</time>)</td></tr>
    <tr><th scope='row'>高値</th><td>696</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T09:17+09:00">09:17</time>)</td></tr>
    <tr><th scope='row'>安値</th><td>689</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T10:18+09:00">10:18</time>)</td></tr>
    <tr><th scope='row'>終値</th><td>691</td><td class="mark">&nbsp;</td><td>(<time datetime="2026-09-25T15:30+09:00">15:30</time>)</td></tr>
  </tbody>
</table>
<table>
  <tbody>
    <tr><th scope='row'>出来高</th><td>4,900&nbsp;株</td></tr>
  </tbody>
</table>
<div class="stock_pts_div" data-mode="1">
<table>
  <tbody>
    <tr><th scope='row'>始値</th><td>－</td><td>(<time datetime=""></time>)</td></tr>
    <tr><th scope='row'>高値</th><td>－</td><td>(<time datetime=""></time>)</td></tr>
    <tr><th scope='row'>安値</th><td>－</td><td>(<time datetime=""></time>)</td></tr>
    <tr><th scope='row'>現在値</th><td>－</td><td>(<time datetime=""></time>)</td></tr>
  </tbody>
</table>
</div>
</div>
</body></html>`

func TestOhlcFromKobetsuTable_IgnoresPtsPlaceholderTable(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(kobetsuFixtureWithPtsHTML))
	if err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	open, high, low, close := ohlcFromKobetsuTable(doc)

	if open != "695" {
		t.Errorf("open = %q, want %q", open, "695")
	}
	if high != "696" {
		t.Errorf("high = %q, want %q", high, "696")
	}
	if low != "689" {
		t.Errorf("low = %q, want %q", low, "689")
	}
	if close != "691" {
		t.Errorf("close = %q, want %q", close, "691")
	}
}

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
