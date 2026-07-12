package handler

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/go-chi/chi/v5"
	"golang.org/x/image/draw"

	"menata.id/runtime/internal/auth"
	"menata.id/runtime/internal/model"
)

// CAP-F06: `file` fields now actually store what's uploaded (previously the
// picker rendered but Create/Update's plain ParseForm never read a
// multipart body, so any selected file was silently dropped). Files live
// on local disk under uploadsDir, keyed by an unguessable token -- a
// prototype-honest choice consistent with this codebase's existing
// single-process, single-host deployment (server-manager.sh, no object
// storage anywhere else in this stack either).
const uploadsDir = "uploads"

// processFileUploads reads every `file`-typed Field's uploaded part from an
// already-parsed multipart form, runs the declared image pipeline
// (Options.Accept implying an image MIME type, Options.Compress true) via
// stdlib image decode + golang.org/x/image/draw resize + JPEG/WebP
// re-encode, and returns fieldID -> storage key for every field that had a
// real upload in this request. A field absent from the returned map means
// "no new upload this request" -- Create leaves it unset, Update leaves the
// record's existing value untouched.
//
// Server-side re-encoding always runs when Compress is declared, whether or
// not the browser already compressed client-side -- the server never trusts
// that happened (same "client is advisory, server enforces" principle
// CAP-C09 already applies to Constraints).
func (h *Handler) processFileUploads(r *http.Request, machine *model.Machine) (map[string]string, error) {
	out := make(map[string]string)
	if r.MultipartForm == nil {
		return out, nil
	}
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeFile {
			continue
		}
		headers := r.MultipartForm.File[f.ID]
		if len(headers) == 0 {
			continue
		}
		fh := headers[0]
		file, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("open uploaded file for %s: %w", f.ID, err)
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("read uploaded file for %s: %w", f.ID, err)
		}
		if len(data) == 0 {
			continue
		}
		contentType := fh.Header.Get("Content-Type")
		if f.Options.Accept != "" && !acceptMatches(f.Options.Accept, contentType) {
			return nil, fmt.Errorf("%s: uploaded file type %q is not accepted (%s)", f.Name, contentType, f.Options.Accept)
		}
		if f.Options.Compress && isImageAccept(f.Options.Accept) {
			data, contentType, err = compressImage(data, f.Options)
			if err != nil {
				return nil, fmt.Errorf("%s: compress: %w", f.Name, err)
			}
		}
		key, err := storeFile(data, contentType)
		if err != nil {
			return nil, fmt.Errorf("%s: store: %w", f.Name, err)
		}
		out[f.ID] = key
	}
	return out, nil
}

func acceptMatches(accept, contentType string) bool {
	for _, want := range strings.Split(accept, ",") {
		want = strings.TrimSpace(want)
		if want == "" {
			continue
		}
		if strings.HasSuffix(want, "/*") {
			if strings.HasPrefix(contentType, strings.TrimSuffix(want, "*")) {
				return true
			}
		} else if want == contentType {
			return true
		}
	}
	return false
}

func isImageAccept(accept string) bool {
	return strings.Contains(accept, "image/")
}

// compressImage decodes a JPEG/PNG/GIF, resizes (longest edge down to
// MaxDimension, aspect-preserving, only ever shrinks, never upscales -- 0
// means no resize, compress-only), and re-encodes as JPEG (default) or
// WebP (Options.Format == "webp", via github.com/chai2010/webp + the
// system's libwebp -- confirmed available on this host, see CLAUDE.md).
func compressImage(data []byte, opts model.FieldOptions) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	if opts.MaxDimension > 0 {
		img = resizeToMax(img, opts.MaxDimension)
	}
	var buf bytes.Buffer
	if opts.Format == "webp" {
		if err := webp.Encode(&buf, img, &webp.Options{Quality: 80}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil
	}
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}

func resizeToMax(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}
	var nw, nh int
	if w >= h {
		nw = maxDim
		nh = h * maxDim / w
	} else {
		nh = maxDim
		nw = w * maxDim / h
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func storeFile(data []byte, contentType string) (string, error) {
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		return "", err
	}
	token, err := auth.NewToken() // 32 bytes of entropy, base64url -- ServeFile's own trust boundary
	if err != nil {
		return "", err
	}
	key := token + extensionFor(contentType)
	if err := os.WriteFile(filepath.Join(uploadsDir, key), data, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

func extensionFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// contentTypeForExtension is extensionFor's reverse, used by ServeFile --
// deliberately a fixed table, not the OS's mime.types (stdlib's
// mime.TypeByExtension falls back to reading /etc/mime.types, which isn't
// guaranteed present or identical across every host this might run on).
func contentTypeForExtension(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	default:
		return ""
	}
}

// ServeFile handles GET /files/{key} -- keys are unguessable (32 bytes of
// entropy from auth.NewToken), so this deliberately serves without a
// separate per-request permission check, the same trust boundary this
// codebase already gives a session or CSRF token. filepath.Base strips any
// path-traversal attempt before touching the filesystem.
func (h *Handler) ServeFile(w http.ResponseWriter, r *http.Request) {
	key := filepath.Base(chi.URLParam(r, "key"))
	path := filepath.Join(uploadsDir, key)
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ct := contentTypeForExtension(filepath.Ext(key)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Write(data)
}
