package gorouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func multyPeekArray(in [][]byte) []string {
	out := make([]string, 0)
	for _, v := range in {
		out = append(out, string(v))
	}
	return out
}

func TestTree(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/work", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/work/bork", set3)

	set4 := Set("4")
	tr.Add(MethodPost, "/work/cork", set4)

	set5 := Set("5")
	set5.ID = "5"
	tr.Add(MethodPost, "/walk/", set5)
	tr.Add(MethodGet, "/walk/", set5)

	assert.NotNil(t, 1)
}

func TestSplitPath(t *testing.T) {
	// t.Skip("AAA")
	data := map[string][]string{
		"":        {"/"},
		"/":       {"/"},
		"/work":   {"/", "work"},
		".":       {"/", "."},
		"/a/b/c/": {"/", "a", "b", "c"},
	}

	for path, d := range data {
		paths := splitPath(path)
		assert.EqualValues(t, d, paths, "for '"+path+"'")
	}
}

func TestTreeTop1(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/work", set2)

	t.Logf("TestTreeTop1:\n---------\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/'")
	assert.EqualValues(t, "1", res.HandlerSet.ID, "expected 1 for '/'")

	res = tr.Find(MethodPost, "", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/'")
	assert.EqualValues(t, "1", res.HandlerSet.ID, "expected 1 for '/'")
}

func TestTreeTop2(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/work", set2)

	t.Logf("TestTreeTop2:\n---------\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/'")
	assert.EqualValues(t, "1", res.HandlerSet.ID, "expected 1 for '/'")
}

func TestTreeLevel1_0(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/work", set2)
	tr.Add(MethodGet, "/work", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/work/bork", set3)

	set4 := Set("4")
	tr.Add(MethodPost, "/work/cork", set4)

	set5 := Set("5")
	tr.Add(MethodPost, "/walk/", set5)
	tr.Add(MethodGet, "/walk/", set5)

	t.Logf("TestTreeLevel1_0:\n---------\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/work", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/work'")
	assert.EqualValues(t, "2", res.HandlerSet.ID, "expected 2 for '/work'")

	res = tr.Find(MethodGet, "/work", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/work'")
	assert.EqualValues(t, "2", res.HandlerSet.ID, "expected 2 for '/work'")
}

func TestTreeLevel1_1(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/work/bork/pork", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/work/bork", set3)

	set4 := Set("4")
	tr.Add(MethodPost, "/work/cork", set4)

	set5 := Set("5")
	tr.Add(MethodPost, "/walk/", set5)
	tr.Add(MethodGet, "/walk/", set5)

	t.Logf("TestTreeLevel1_1:\n---------\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/work", args)
	assert.NotNil(t, res)
	assert.EqualValues(t, false, res.Find, "expected false for '/work'")

	res = tr.Find(MethodPost, "/work/cork", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/work/cork'")
	assert.EqualValues(t, "4", res.HandlerSet.ID, "expected 4 for '/work/cork'")
}

func TestTreeUrlIDs_0(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "/", set)

	set3 := Set("2")
	tr.Add(MethodPost, "/:id", set3)

	t.Logf("TestTreeUrlIDs_0:\n---------\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/vasya", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/b/vasya/user'")
	assert.EqualValues(t, "2", res.HandlerSet.ID, "expected 6 for '/b/vasya/user'")

	assert.EqualValues(t, "vasya", string(args.Peek("id")), "expected res.UrlIDs for '/b/vasya/user'")
	//assert.EqualValues(t, url.Values{"id": []string{"vasya"}}, res.UrlIDs, "expected res.UrlIDs for '/b/vasya/user'")
}

func TestTreeUrlIDs_1(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/:login", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/:login/user", set3)

	t.Logf("\n---------\nTestTreeUrlIDs_1:\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/vasya/user", args)
	assert.NotNil(t, res)
	assert.NotNil(t, res.HandlerSet)
	assert.EqualValues(t, true, res.Find, "expected true for '/b/vasya/user'")
	assert.EqualValues(t, "3", res.HandlerSet.ID, "expected 3 for '/b/vasya/user'")
	assert.EqualValues(t, "vasya", string(args.Peek("login")), "expected res.UrlIDs for '/b/vasya/user'")
	//assert.EqualValues(t, url.Values{"login": []string{"vasya"}}, res.UrlIDs, "expected res.UrlIDs for '/b/vasya/user'")
}

func TestTreeUrlIDs_2(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	set.ID = "1"
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/a/:id/borrow", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/a/:id", set3)

	set4 := Set("4")
	tr.Add(MethodPost, "/b/:login", set4)

	set5 := Set("5")
	tr.Add(MethodPost, "/b/:login/admin", set5)

	set6 := Set("6")
	tr.Add(MethodPost, "/b/:login/user", set6)

	set7 := Set("7")
	tr.Add(MethodPost, "/b/:login/:login/user", set7)

	t.Logf("\n---------\nTestTreeUrlIDs_2:\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/b/vasya/user", args)
	assert.NotNil(t, res, "expected not nil for '/b/vasya/user'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/b/vasya/user'")
	assert.EqualValues(t, true, res.Find, "expected true for '/b/vasya/user'")
	assert.EqualValues(t, "6", res.HandlerSet.ID, "expected 6 for '/b/vasya/user'")
	assert.EqualValues(t, "vasya", string(args.Peek("login")), "expected res.UrlIDs for '/b/vasya/user'")
	//assert.EqualValues(t, url.Values{"login": []string{"vasya"}}, res.UrlIDs, "expected res.UrlIDs for '/b/vasya/user'")

	args.Reset()
	res = tr.Find(MethodPost, "/b/vasya/petrov/user", args)
	assert.NotNil(t, res, "expected not nil for '/b/vasya/petrov/user'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/b/vasya/petrov/use'")
	assert.EqualValues(t, true, res.Find, "expected true for '/b/vasya/petrov/use'")
	assert.EqualValues(t, "7", res.HandlerSet.ID, "expected 6 for '/b/vasya/petrov/use'")

	tmp := multyPeekArray(args.PeekMulti("login"))
	assert.EqualValues(t, []string{"vasya", "petrov"}, tmp, "expected res.UrlIDs for '/b/vasya/petrov/use'")
}

func TestTreeUrlIDs_3(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	set.ID = "1"
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/a/:id/borrow", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/a/:id", set3)

	set4 := Set("4")
	tr.Add(MethodPost, "/b/:login", set4)

	set5 := Set("5")
	tr.Add(MethodPost, "/b/:login/admin", set5)

	set6 := Set("6")
	tr.Add(MethodPost, "/b/:login/user", set6)

	set7 := Set("7")
	tr.Add(MethodPost, "/b/:login/:id/user", set7)

	t.Logf("\n---------\nTestTreeUrlIDs_3:\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/b/vasya/123/user", args)
	assert.NotNil(t, res, "expected not nil for '/b/vasya/123/user'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/b/vasya/123/user'")
	assert.EqualValues(t, true, res.Find, "expected true for '/b/vasya/123/user'")
	assert.EqualValues(t, "7", res.HandlerSet.ID, "expected 6 for '/b/vasya/123/user'")

	assert.EqualValues(t, "123", string(args.Peek("id")), "expected res.UrlIDs for '/b/vasya/user'")
	assert.EqualValues(t, "vasya", string(args.Peek("login")), "expected res.UrlIDs for '/b/vasya/user'")

	//assert.EqualValues(t, url.Values{"login": []string{"vasya"}, "id": []string{"123"}}, res.UrlIDs, "expected res.UrlIDs for '/b/vasya/user'")

	res = tr.Find(MethodGet, "/", args)
	assert.NotNil(t, res, "expected not nil for '/'")
	assert.EqualValues(t, false, res.Find, "expected false for '/'")
}

func TestTreeUrlID_4(t *testing.T) {
	// t.Skip("AAA")
	tr := newTree()
	assert.NotNil(t, tr)

	set := Set("1")
	set.ID = "1"
	tr.Add(MethodPost, "/", set)

	set2 := Set("2")
	tr.Add(MethodPost, "/a/:id/borrow", set2)

	set3 := Set("3")
	tr.Add(MethodPost, "/a/:id", set3)

	set4 := Set("4")
	tr.Add(MethodPost, "/b/:login", set4)

	set5 := Set("5")
	tr.Add(MethodPost, "/b/:login/admin", set5)

	set6 := Set("6")
	tr.Add(MethodPost, "/b/:login/user", set6)

	set7 := Set("7")
	tr.Add(MethodPost, "/b/:login/:id/user", set7)

	set8 := Set("8")
	tr.Add(MethodPost, "/a/:id/:login/admin", set8)

	tr.Add(MethodPost, "/a/:id/:login/user", Set("9"))
	tr.Add(MethodPost, "/a/:id/:login/user/account", Set("10"))
	tr.Add(MethodGetPost, "/a/:id/user/account", Set("11"))

	t.Logf("\n---------\nTestTreeUrlIDs_3:\n%s\n---------\n", tr.String())

	args := &fasthttp.Args{}
	defer args.Reset()

	res := tr.Find(MethodPost, "/b/vasya/123/user", args)
	assert.NotNil(t, res, "expected not nil for '/b/vasya/123/user'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/b/vasya/123/user'")
	assert.EqualValues(t, true, res.Find, "expected true for '/b/vasya/123/user'")
	assert.EqualValues(t, "7", res.HandlerSet.ID, "expected 6 for '/b/vasya/123/user'")
	assert.EqualValues(t, "123", string(args.Peek("id")), "expected res.UrlIDs for '/b/vasya/123/user'")
	assert.EqualValues(t, "vasya", string(args.Peek("login")), "expected res.UrlIDs for '/b/vasya/123/user'")

	res = tr.Find(MethodPost, "/a/tennis player/Jean-Julien Rojer/user", args)
	assert.NotNil(t, res, "expected not nil for '/a/tennis player/Jean-Julien Rojer/user'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/a/tennis player/Jean-Julien Rojer/user'")
	assert.EqualValues(t, true, res.Find, "expected true for '/a/tennis player/Jean-Julien Rojer/user'")
	assert.EqualValues(t, "9", res.HandlerSet.ID, "expected 9 for '/a/tennis player/Jean-Julien Rojer/user'")
	assert.EqualValues(t, "tennis player", string(args.Peek("id")), "expected res.UrlIDs for '/a/tennis player/Jean-Julien Rojer/user'")
	assert.EqualValues(t, "Jean-Julien Rojer", string(args.Peek("login")), "expected res.UrlIDs for '/a/tennis player/Jean-Julien Rojer/user'")

	res = tr.Find(MethodPost, "/a/tennis player/Jean-Julien Rojer/user/account", args)
	assert.NotNil(t, res, "expected not nil for '/a/tennis player/Jean-Julien Rojer/user/account'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/a/tennis player/Jean-Julien Rojer/user/account'")
	assert.EqualValues(t, true, res.Find, "expected true for '/a/tennis player/Jean-Julien Rojer/user/account'")
	assert.EqualValues(t, "10", res.HandlerSet.ID, "expected 10 for '/a/tennis player/Jean-Julien Rojer/user/account'")
	assert.EqualValues(t, "tennis player", string(args.Peek("id")), "expected res.UrlIDs for '/a/tennis player/Jean-Julien Rojer/user/account'")
	assert.EqualValues(t, "Jean-Julien Rojer", string(args.Peek("login")), "expected res.UrlIDs for '/a/tennis player/Jean-Julien Rojer/user/account'")

	res = tr.Find(MethodPost, "/a/tennis player/user/account", args)
	assert.NotNil(t, res, "expected not nil for '/a/tennis player/user/account'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/a/tennis player/user/account'")
	assert.EqualValues(t, true, res.Find, "expected true for '/a/tennis player/user/account'")
	assert.EqualValues(t, "11", res.HandlerSet.ID, "expected 10 for '/a/tennis player/user/account'")
	assert.EqualValues(t, "tennis player", string(args.Peek("id")), "expected res.UrlIDs for '/a/tennis player/user/account'")

	res = tr.Find(MethodGet, "/a/tennis player/user/account", args)
	assert.NotNil(t, res, "expected not nil for '/a/tennis player/user/account'")
	assert.EqualValues(t, true, res.Find, "expected true for '/a/tennis player/user/account'")
	assert.NotNil(t, res.HandlerSet, "expected not nil for '/a/tennis player/user/account'")
	assert.EqualValues(t, "11", res.HandlerSet.ID, "expected 10 for '/a/tennis player/user/account'")
	assert.EqualValues(t, "tennis player", string(args.Peek("id")), "expected res.UrlIDs for '/a/tennis player/user/account'")

	res = tr.Find(MethodDelete, "/a/tennis player/user/account", args)
	assert.NotNil(t, res, "expected not nil for '/a/tennis player/user/account'")
	assert.EqualValues(t, false, res.Find, "expected false for '/a/tennis player/user/account'")

	res = tr.Find(MethodPost, "/a", args)
	assert.NotNil(t, res, "expected not nil for '/a'")
	assert.EqualValues(t, false, res.Find, "expected false for '/a'")

	res = tr.Find(MethodGet, "/a", args)
	assert.NotNil(t, res, "expected not nil for '/a'")
	assert.EqualValues(t, false, res.Find, "expected false for '/a'")

	res = tr.Find(MethodGet, "/", args)
	assert.NotNil(t, res, "expected not nil for '/'")
	assert.EqualValues(t, false, res.Find, "expected false for '/'")
}
