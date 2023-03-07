package gorouter

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// MyAds           = "/my-ads"
// MyAdsUpload     = "/my-ads/upload"
// MyAdsList       = "/my-ads/list"
// MyAdsTake       = "/my-ads/take/:id"
// MyAdsTakeJson   = "/my-ads/take/:id/json"
// MyAdsThumbnails = "/my-ads/take/:id/thumbnails"
//
// MyAdsPropertyUpdate = "/my-ads/:id/property/update"
//
// MyAdsConditionAdd      = "/my-ads/:id/condition/:condition/add"
// MyAdsConditionDel      = "/my-ads/:id/condition/:condition/del"
// MyVideoSearchByAdCount = "/my-ads/:id/video_search/count"
// MyAdAttachedVideo      = "/my-ads/attached_video/:id"
//
// AdvertiserProfile   = "/advertiser/profile"
// AdvertiserData      = "/advertiser/profile/data"
// AdvertiserBudgetAdd = "/advertiser/profile/budget/add"

func AddSet(tr *Tree, id, path string) {
	set7 := Set(id)
	tr.Add(MethodGetPost, path, set7)
}

func TestTreeOneMore(t *testing.T) {
	tr := newTree()
	assert.NotNil(t, tr)

	paths := []string{
		"/my-ads",
		"/my-ads/upload",
		"/my-ads/list",
		"/my-ads/take/:id",
		"/my-ads/take/:id/json",
		"/my-ads/take/:id/thumbnails",

		"/my-ads/:id/property/update",

		"/my-ads/:id/condition/:condition/add",
		"/my-ads/:id/condition/:condition/del",
		"/my-ads/:id/video_search/count",
		"/my-ads/attached_video/:id",

		"/advertiser/profile",
		"/advertiser/profile/data",
		"/advertiser/profile/budget/add",
	}

	for i := range paths {
		t.Logf("paths: %d => %s\n", i, paths[i])
		AddSet(tr, strconv.Itoa(i), paths[i])
	}

	t.Logf("\n---------\nTestTreeUrlIDs_3:\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/my-ads/attached_video/51", args)
	assert.NotNil(t, res, "expected not nil for '/my-ads/attached_video/51'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/my-ads/attached_video/51'")
	assert.EqualValues(t, true, res.Find, "expected true for '/my-ads/attached_video/51'")
	assert.EqualValues(t, "10", res.HandlerSet.ID, "expected 6 for '/my-ads/attached_video/51'")
	assert.EqualValues(t, "51", string(args.Peek("id")), "expected res.UrlIDs for '/my-ads/attached_video/51'")
}
