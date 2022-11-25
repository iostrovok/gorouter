// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// HTTP file system request handler

package static

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// A Dir implements FileSystem using the native file system restricted to a
// specific directory tree.
//
// While the FileSystem.Open method takes '/'-separated paths, a Directory's string
// value is a filename on the native file system, not a URL, so it is separated
// by filepath.Separator, which isn't necessarily '/'.
//
// Note that Dir could expose sensitive files and directories. Dir will follow
// symlinks pointing out of the directory tree, which can be especially dangerous
// if serving from a directory in which users are able to create arbitrary symlinks.
// Dir will also allow access to files and directories starting with a period,
// which could expose sensitive directories like .git or sensitive files like
// .htpasswd. To exclude files with a leading period, remove the files/directories
// from the server or create a custom FileSystem implementation.
//
// An empty Dir is treated as ".".
type Dir string

// mapOpenError maps the provided non-nil error from opening name
// to a possibly better non-nil error. In particular, it turns OS-specific errors
// about opening files in non-directories into fs.ErrNotExist. See Issues 18984 and 49552.
func mapOpenError(originalErr error, name string, sep rune, stat func(string) (fs.FileInfo, error)) error {
	if errors.Is(originalErr, fs.ErrNotExist) || errors.Is(originalErr, fs.ErrPermission) {
		return originalErr
	}

	parts := strings.Split(name, string(sep))
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		fi, err := stat(strings.Join(parts[:i+1], string(sep)))
		if err != nil {
			return originalErr
		}
		if !fi.IsDir() {
			return fs.ErrNotExist
		}
	}
	return originalErr
}

// Open implements FileSystem using os.Open, opening files for reading rooted
// and relative to the directory d.
func (d Dir) Open(name string) (File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}
	dir := string(d)
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, mapOpenError(err, fullName, filepath.Separator, os.Stat)
	}
	return f, nil
}

// A FileSystem implements access to a collection of named files.
// The elements in a file path are separated by slash ('/', U+002F)
// characters, regardless of host operating system convention.
// See the FileServer function to convert a FileSystem to a Handler.
//
// This interface predates the fs.FS interface, which can be used instead:
// the FS adapter function converts a fs.FS to a FileSystem.
type FileSystem interface {
	Open(name string) (File, error)
}

// A File is returned by a FileSystem's Open method and can be
// served by the FileServer implementation.
//
// The methods should behave the same as those on an *os.File.
type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]fs.FileInfo, error)
	Stat() (fs.FileInfo, error)
}

type anyDirs interface {
	len() int
	name(i int) string
	isDir(i int) bool
}

type fileInfoDirs []fs.FileInfo

func (d fileInfoDirs) len() int          { return len(d) }
func (d fileInfoDirs) isDir(i int) bool  { return d[i].IsDir() }
func (d fileInfoDirs) name(i int) string { return d[i].Name() }

type dirEntryDirs []fs.DirEntry

func (d dirEntryDirs) len() int          { return len(d) }
func (d dirEntryDirs) isDir(i int) bool  { return d[i].IsDir() }
func (d dirEntryDirs) name(i int) string { return d[i].Name() }

func dirList(ctx *fasthttp.RequestCtx, f File) error {
	// Prefer to use ReadDir instead of Readdir,
	// because the former doesn't require calling
	// Stat on every entry of a directory on Unix.
	var dirs anyDirs
	var err error
	if d, ok := f.(fs.ReadDirFile); ok {
		var list dirEntryDirs
		list, err = d.ReadDir(-1)
		dirs = list
	} else {
		var list fileInfoDirs
		list, err = f.Readdir(-1)
		dirs = list
	}

	if err != nil {
		logf(ctx, "http: error reading directory: %v", err)
		Error(ctx, "Error reading directory", fasthttp.StatusInternalServerError)
		return err
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs.name(i) < dirs.name(j) })

	ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
	if _, err := ctx.WriteString("<pre>\n"); err != nil {
		return err
	}
	for i, n := 0, dirs.len(); i < n; i++ {
		name := dirs.name(i)
		if dirs.isDir(i) {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		dirUrl := url.URL{Path: name}
		str := `<a href="` + dirUrl.String() + `">` + htmlReplacer.Replace(name) + `</a>` + "\n"
		if _, err := ctx.WriteString(str); err != nil {
			return err
		}

	}
	if _, err := ctx.WriteString("</pre>\n"); err != nil {
		return err
	}

	return nil
}

// ServeContent replies to the request using the content in the
// provided ReadSeeker. The main benefit of ServeContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
//
// If the response's Content-Type header is not set, ServeContent
// first tries to deduce the type from name's file extension and,
// if that fails, falls back to reading the first block of the content
// and passing it to DetectContentType.
// The name is otherwise unused; in particular it can be empty and is
// never sent in the response.
//
// If modTime is not the zero time or Unix epoch, ServeContent
// includes it in a Last-Modified header in the response. If the
// request includes an If-Modified-Since header, ServeContent uses
// modTime to decide whether the content needs to be sent at all.
//
// The content's Seek method must work: ServeContent uses
// a seek to the end of the content to determine its size.
//
// If the caller has set w's ETag header formatted per RFC 7232, section 2.3,
// ServeContent uses it to handle requests using If-Match, If-None-Match, or If-Range.
//
// Note that *os.File implements the io.ReadSeeker interface.
func ServeContent(ctx *fasthttp.RequestCtx, name string, modTime time.Time, content io.ReadSeeker) error {
	sizeFunc := func() (int64, error) {
		size, err := content.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, errSeeker
		}
		_, err = content.Seek(0, io.SeekStart)
		if err != nil {
			return 0, errSeeker
		}
		return size, nil
	}

	return serveContent(ctx, name, modTime, sizeFunc, content)
}

// errSeeker is returned by ServeContent's sizeFunc when the content doesn't seek properly.
// The underlying Seeker's error text isn't included in the sizeFunc reply, so it's not sent over HTTP to end users.
var errSeeker = errors.New("seeker can't seek")

// errNoOverlap is returned by serveContent's parseRange
// if first-byte-pos of all byte-range-spec values is greater than the content size.
var errNoOverlap = errors.New("invalid range: failed to overlap")

// if name is empty, filename is unknown. (used for mime type, before sniffing)
// if modTime.IsZero(), modTime is unknown.
// content must be seeked to the beginning of the file.
// The sizeFunc is called at most once. Its error, if any, is sent in the HTTP response.
func serveContent(ctx *fasthttp.RequestCtx, name string, modTime time.Time, sizeFunc func() (int64, error), content io.ReadSeeker) error {
	setLastModified(ctx, modTime)
	done, rangeReq := checkPreconditions(ctx, modTime)
	if done {
		return nil
	}

	code := fasthttp.StatusOK

	// If Content-Type isn't set, use the file's extension to find it, but
	// if the Content-Type is unset explicitly, do not sniff the type.
	cType := string(ctx.Request.Header.Peek("Content-Type"))
	if cType == "" {
		cType = mime.TypeByExtension(filepath.Ext(name))
		if cType == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(content, buf[:])
			cType = DetectContentType(buf[:n])
			_, err := content.Seek(0, io.SeekStart) // rewind to output whole file
			if err != nil {
				Error(ctx, "seeker can't seek", fasthttp.StatusInternalServerError)
				return err
			}
		}
		ctx.Response.Header.Set("Content-Type", cType)
	}

	size, err := sizeFunc()
	if err != nil {
		Error(ctx, err.Error(), fasthttp.StatusInternalServerError)
		return err
	}

	// handle Content-Range header.
	sendSize := size
	var sendContent io.Reader = content
	if size >= 0 {
		ranges, err := parseRange(rangeReq, size)
		if err != nil {
			if err == errNoOverlap {
				ctx.Response.Header.Set("Content-Range", fmt.Sprintf("bytes */%d", size))
			}
			Error(ctx, err.Error(), fasthttp.StatusRequestedRangeNotSatisfiable)
			return err
		}
		if sumRangesSize(ranges) > size {
			// The total number of bytes in all the ranges
			// is larger than the size of the file by
			// itself, so this is probably an attack, or a
			// dumb client. Ignore the range request.
			ranges = nil
		}
		switch {
		case len(ranges) == 1:
			// RFC 7233, Section 4.1:
			// "If a single part is being transferred, the server
			// generating the 206 response MUST generate a
			// Content-Range header field, describing what range
			// of the selected representation is enclosed, and a
			// payload consisting of the range.
			// ...
			// A server MUST NOT generate a multipart response to
			// a request for a single range, since a client that
			// does not request multiple parts might not support
			// multipart responses."
			ra := ranges[0]
			if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
				Error(ctx, err.Error(), fasthttp.StatusRequestedRangeNotSatisfiable)
				return err
			}
			sendSize = ra.length
			code = fasthttp.StatusPartialContent
			ctx.Response.Header.Set("Content-Range", ra.contentRange(size))
		case len(ranges) > 1:
			sendSize, err = rangesMIMESize(ranges, cType, size)
			if err != nil {
				Error(ctx, err.Error(), fasthttp.StatusRequestedRangeNotSatisfiable)
				return err
			}

			code = fasthttp.StatusPartialContent
			pr, pw := io.Pipe()
			mw := multipart.NewWriter(pw)
			ctx.Response.Header.Set("Content-Type", "multipart/byteranges; boundary="+mw.Boundary())
			sendContent = pr

			defer func() {
				_ = pr.Close() // cause writing goroutine to fail and exit if CopyN doesn't finish.
			}()

			go func() {
				for _, ra := range ranges {
					part, err := mw.CreatePart(ra.mimeHeader(cType, size))
					if err != nil {
						_ = pw.CloseWithError(err)
						return
					}

					if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
						_ = pw.CloseWithError(err)
						return
					}

					if _, err := io.CopyN(part, content, ra.length); err != nil {
						_ = pw.CloseWithError(err)
						return
					}
				}
				_ = mw.Close()
				_ = pw.Close()
			}()
		}

		ctx.Response.Header.Set("Accept-Ranges", "bytes")
		if len(ctx.Response.Header.Peek("Content-Encoding")) == 0 {
			ctx.Response.Header.Set("Content-Length", strconv.FormatInt(sendSize, 10))
		}
	}

	ctx.Response.SetStatusCode(code)
	if string(ctx.Method()) != "HEAD" {
		if _, err := io.CopyN(ctx.Response.BodyWriter(), sendContent, sendSize); err != nil {
			logf(ctx, "http-static: CopyN: %v", err)
		}
	} else {
		if _, err := ctx.Write([]byte{}); err != nil {
			logf(ctx, "http-static: HEAD: %v", err)
		}
	}

	return nil
}

// scanETag determines if a syntactically valid ETag is present at s. If so,
// the ETag and remaining text after consuming ETag is returned. Otherwise,
// it returns "", "".
func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	// ETag is either W/"text" or "text".
	// See RFC 7232 2.3.
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		// Character values allowed in ETags.
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return s[:i+1], s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

// etagStrongMatch reports whether a and b match using strong ETag comparison.
// Assumes a and b are valid ETags.
func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

// etagWeakMatch reports whether a and b match using weak ETag comparison.
// Assumes a and b are valid ETags.
func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

// condResult is the result of an HTTP request precondition check.
// See https://tools.ietf.org/html/rfc7232 section 3.
type condResult int

const (
	condNone condResult = iota
	condTrue
	condFalse
)

func checkIfMatch(ctx *fasthttp.RequestCtx) condResult {
	im := string(ctx.Request.Header.Peek("If-Match"))
	if im == "" {
		return condNone
	}
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return condTrue
		}
		etag, remain := scanETag(im)
		if etag == "" {
			break
		}
		if etagStrongMatch(etag, string(ctx.Response.Header.Peek("Etag"))) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

func checkIfUnmodifiedSince(ctx *fasthttp.RequestCtx, modTime time.Time) condResult {
	ius := string(ctx.Request.Header.Peek("If-Unmodified-Since"))
	if ius == "" || isZeroTime(modTime) {
		return condNone
	}
	t, err := ParseTime(ius)
	if err != nil {
		return condNone
	}

	// The Last-Modified header truncates sub-second precision so
	// the modTime needs to be truncated too.
	modTime = modTime.Truncate(time.Second)
	if modTime.Before(t) || modTime.Equal(t) {
		return condTrue
	}
	return condFalse
}

func checkIfNoneMatch(ctx *fasthttp.RequestCtx) condResult {
	inm := string(ctx.Request.Header.Peek("If-None-Match"))
	if inm == "" {
		return condNone
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
			continue
		}
		if buf[0] == '*' {
			return condFalse
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, string(ctx.Response.Header.Peek("Etag"))) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

func checkIfModifiedSince(ctx *fasthttp.RequestCtx, modTime time.Time) condResult {
	if string(ctx.Method()) != "GET" && string(ctx.Method()) != "HEAD" {
		return condNone
	}
	ims := string(ctx.Request.Header.Peek("If-Modified-Since"))
	if ims == "" || isZeroTime(modTime) {
		return condNone
	}
	t, err := ParseTime(ims)
	if err != nil {
		return condNone
	}
	// The Last-Modified header truncates sub-second precision so
	// the modTime needs to be truncated too.
	modTime = modTime.Truncate(time.Second)
	if modTime.Before(t) || modTime.Equal(t) {
		return condFalse
	}
	return condTrue
}

func checkIfRange(ctx *fasthttp.RequestCtx, modTime time.Time) condResult {
	switch string(ctx.Method()) {
	case "GET", "HEAD":
	default:
		return condNone
	}

	ir := string(ctx.Request.Header.Peek("If-Range"))
	if ir == "" {
		return condNone
	}

	etag, _ := scanETag(ir)
	if etag != "" {
		if etagStrongMatch(etag, string(ctx.Response.Header.Peek("Etag"))) {
			return condTrue
		} else {
			return condFalse
		}
	}
	// The If-Range value is typically the ETag value, but it may also be
	// the modTime date. See golang.org/issue/8367.
	if modTime.IsZero() {
		return condFalse
	}
	t, err := ParseTime(ir)
	if err != nil {
		return condFalse
	}
	if t.Unix() == modTime.Unix() {
		return condTrue
	}
	return condFalse
}

var unixEpochTime = time.Unix(0, 0)

// isZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

func setLastModified(ctx *fasthttp.RequestCtx, modTime time.Time) {
	if !isZeroTime(modTime) {
		ctx.Response.Header.Set("Last-Modified", modTime.UTC().Format(TimeFormat))
	}
}

func writeNotModified(ctx *fasthttp.RequestCtx) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	ctx.Response.Header.Del("Content-Type")
	ctx.Response.Header.Del("Content-Length")
	ctx.Response.Header.Del("Content-Encoding")
	if string(ctx.Response.Header.Peek("Etag")) != "" {
		ctx.Response.Header.Del("Last-Modified")
	}
	ctx.Response.SetStatusCode(fasthttp.StatusNotModified)
}

// checkPreconditions evaluates request preconditions and reports whether a precondition
// resulted in sending StatusNotModified or StatusPreconditionFailed.
func checkPreconditions(ctx *fasthttp.RequestCtx, modTime time.Time) (done bool, rangeHeader string) {
	// This function carefully follows RFC 7232 section 6.
	ch := checkIfMatch(ctx)
	if ch == condNone {
		ch = checkIfUnmodifiedSince(ctx, modTime)
	}
	if ch == condFalse {
		ctx.Response.Header.SetStatusCode(fasthttp.StatusPreconditionFailed)
		return true, ""
	}
	switch checkIfNoneMatch(ctx) {
	case condFalse:
		if string(ctx.Method()) == "GET" || string(ctx.Method()) == "HEAD" {
			writeNotModified(ctx)
			return true, ""
		} else {
			ctx.Response.Header.SetStatusCode(fasthttp.StatusPreconditionFailed)
			return true, ""
		}
	case condNone:
		if checkIfModifiedSince(ctx, modTime) == condFalse {
			writeNotModified(ctx)
			return true, ""
		}
	}

	rangeHeader = string(ctx.Request.Header.Peek("Range"))
	if rangeHeader != "" && checkIfRange(ctx, modTime) == condFalse {
		rangeHeader = ""
	}
	return false, rangeHeader
}

// name is '/'-separated, not filepath.Separator.
func serveFile(ctx *fasthttp.RequestCtx, fsm FileSystem, name string, showDir, redirect bool) error {
	const indexPage = "/index.html"

	// redirect .../index.html to .../
	// can't use Redirect() because that would make the path absolute,
	// which would be a problem running under StripPrefix
	if strings.HasSuffix(string(ctx.Path()), indexPage) {
		localRedirect(ctx, "./")
		return nil
	}

	file, err := fsm.Open(name)
	if err != nil {
		msg, code := toHTTPError(err)
		Error(ctx, msg, code)
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		msg, code := toHTTPError(err)
		Error(ctx, msg, code)
		return err
	}

	if !showDir && fileInfo.IsDir() {
		msg, code := toHTTPError(fs.ErrPermission)
		Error(ctx, msg, code)
		return err
	}

	if redirect {
		// redirect to canonical path: / at end of directory url
		// r.URL.Path always begins with /
		redirectUrl := string(ctx.Path())
		if fileInfo.IsDir() {
			if redirectUrl[len(redirectUrl)-1] != '/' {
				localRedirect(ctx, path.Base(redirectUrl)+"/")
				return nil
			}
		} else {
			if redirectUrl[len(redirectUrl)-1] == '/' {
				localRedirect(ctx, "../"+path.Base(redirectUrl))
				return nil
			}
		}
	}

	if fileInfo.IsDir() {
		//url := r.URL.Path
		dirUrl := string(ctx.Path())
		// redirect if the directory name doesn't end in a slash
		if dirUrl == "" || dirUrl[len(dirUrl)-1] != '/' {
			localRedirect(ctx, path.Base(dirUrl)+"/")
			return nil
		}

		// use contents of index.html for directory, if present
		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fsm.Open(index)
		if err == nil {
			dd, err := ff.Stat()
			if err == nil {
				//name = index
				fileInfo = dd
				file = ff
			}

			_ = ff.Close()
		}
	}

	// Still a directory? (we didn't find an index.html file)
	if fileInfo.IsDir() {
		if checkIfModifiedSince(ctx, fileInfo.ModTime()) == condFalse {
			writeNotModified(ctx)
			return nil
		}
		setLastModified(ctx, fileInfo.ModTime())
		return dirList(ctx, file)
	}

	// serveContent will check modification time
	sizeFunc := func() (int64, error) { return fileInfo.Size(), nil }

	// ignore error here
	return serveContent(ctx, fileInfo.Name(), fileInfo.ModTime(), sizeFunc, file)
}

// toHTTPError returns a non-specific HTTP error message and status code
// for a given non-nil error value. It's important that toHTTPError does not
// actually return err.Error(), since msg and httpStatus are returned to users,
// and historically Go's ServeContent always returned just "404 Not Found" for
// all errors. We don't want to start leaking information in error messages.
func toHTTPError(err error) (msg string, httpStatus int) {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "404 page not found", fasthttp.StatusNotFound
	case errors.Is(err, fs.ErrPermission):
		return "403 Forbidden", fasthttp.StatusForbidden
	}

	// Default:
	return "500 Internal Server Error", fasthttp.StatusInternalServerError
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(ctx *fasthttp.RequestCtx, newPath string) {
	//if q := r.URL.RawQuery; q != "" {
	if q := ctx.Request.RequestURI(); len(q) > 0 {
		newPath += "?" + string(q)
	}

	ctx.Response.Header.Set("Location", newPath)
	ctx.Response.Header.SetStatusCode(fasthttp.StatusMovedPermanently)
}

// ServeFile replies to the request with the contents of the named
// file or directory.
//
// If the provided file or directory name is a relative path, it is
// interpreted relative to the current directory and may ascend to
// parent directories. If the provided name is constructed from user
// input, it should be sanitized before calling ServeFile.
//
// As a precaution, ServeFile will reject requests where r.URL.Path
// contains a ".." path element; this protects against callers who
// might unsafely use filepath.Join on r.URL.Path without sanitizing
// it and then use that filepath.Join result as the name argument.
//
// As another special case, ServeFile redirects any request where r.URL.Path
// ends in "/index.html" to the same path, without the final
// "index.html". To avoid such redirects either modify the path or
// use ServeContent.
//
// Outside those two special cases, ServeFile does not use
// r.URL.Path for selecting the file or directory to serve; only the
// file or directory provided in the name argument is used.
func ServeFile(ctx *fasthttp.RequestCtx, name string) error {
	//if containsDotDot(r.URL.Path) {
	if containsDotDot(string(ctx.Path())) {
		// Too many programs use r.URL.Path to construct the argument to
		// serveFile. Reject the request under the assumption that happened
		// here and ".." may not be wanted.
		// Note that name might not contain "..", for example if code (still
		// incorrectly) used filepath.Join(myDir, r.URL.Path).
		Error(ctx, "invalid URL path", fasthttp.StatusBadRequest)
		return errors.New("invalid URL path")
	}

	dir, file := filepath.Split(name)
	return serveFile(ctx, Dir(dir), file, false, false)
}

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

type fileHandler struct {
	root FileSystem
}

type ioFS struct {
	fSys fs.FS
}

type ioFile struct {
	file fs.File
}

func (f ioFS) Open(name string) (File, error) {
	if name == "/" {
		name = "."
	} else {
		name = strings.TrimPrefix(name, "/")
	}
	file, err := f.fSys.Open(name)
	if err != nil {
		return nil, mapOpenError(err, name, '/', func(path string) (fs.FileInfo, error) {
			return fs.Stat(f.fSys, path)
		})
	}
	return ioFile{file}, nil
}

func (f ioFile) Close() error               { return f.file.Close() }
func (f ioFile) Read(b []byte) (int, error) { return f.file.Read(b) }
func (f ioFile) Stat() (fs.FileInfo, error) { return f.file.Stat() }

var errMissingSeek = errors.New("io.File missing Seek method")
var errMissingReadDir = errors.New("io.File directory missing ReadDir method")

func (f ioFile) Seek(offset int64, whence int) (int64, error) {
	s, ok := f.file.(io.Seeker)
	if !ok {
		return 0, errMissingSeek
	}
	return s.Seek(offset, whence)
}

func (f ioFile) ReadDir(count int) ([]fs.DirEntry, error) {
	d, ok := f.file.(fs.ReadDirFile)
	if !ok {
		return nil, errMissingReadDir
	}
	return d.ReadDir(count)
}

func (f ioFile) Readdir(count int) ([]fs.FileInfo, error) {
	d, ok := f.file.(fs.ReadDirFile)
	if !ok {
		return nil, errMissingReadDir
	}
	var list []fs.FileInfo
	for {
		dirs, err := d.ReadDir(count - len(list))
		for _, dir := range dirs {
			info, err := dir.Info()
			if err != nil {
				// Pretend it doesn't exist, like (*os.File).Readdir does.
				continue
			}
			list = append(list, info)
		}
		if err != nil {
			return list, err
		}
		if count < 0 || len(list) >= count {
			break
		}
	}
	return list, nil
}

// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

func (r httpRange) mimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		"Content-Type":  {contentType},
	}
}

// parseRange parses a Range header string as per RFC 7233.
// errNoOverlap is returned if none of the ranges overlap.
func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, errors.New("invalid range")
	}
	var ranges []httpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = textproto.TrimString(ra)
		if ra == "" {
			continue
		}
		start, end, ok := strings.Cut(ra, "-")
		if !ok {
			return nil, errors.New("invalid range")
		}
		start, end = textproto.TrimString(start), textproto.TrimString(end)
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file,
			// and we are dealing with <suffix-length>
			// which has to be a non-negative integer as per
			// RFC 7233 Section 2.1 "Byte-Ranges".
			if end == "" || end[0] == '-' {
				return nil, errors.New("invalid range")
			}
			i, err := strconv.ParseInt(end, 10, 64)
			if i < 0 || err != nil {
				return nil, errors.New("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, errors.New("invalid range")
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, errors.New("invalid range")
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, errNoOverlap
	}
	return ranges, nil
}
