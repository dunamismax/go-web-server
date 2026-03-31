package handler

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const frontendBackendProxyBase = "/_backend"

// FrontendHandler serves the embedded Astro build output for shipped browser routes.
type FrontendHandler struct {
	distFS      fs.FS
	assetServer http.Handler
}

// NewFrontendHandler creates a frontend handler backed by the embedded dist filesystem.
func NewFrontendHandler(distFS fs.FS) *FrontendHandler {
	return &FrontendHandler{
		distFS:      distFS,
		assetServer: http.FileServer(http.FS(distFS)),
	}
}

// Page serves a built HTML page from the embedded frontend dist.
func (h *FrontendHandler) Page(name string) echo.HandlerFunc {
	return func(c echo.Context) error {
		body, err := fs.ReadFile(h.distFS, name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to serve embedded frontend page")
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		_, err = c.Response().Write(body)

		return err
	}
}

// Asset serves built frontend assets such as /_astro/* files.
func (h *FrontendHandler) Asset(c echo.Context) error {
	h.assetServer.ServeHTTP(c.Response(), c.Request())

	return nil
}

// FrontendBackendProxyRewrite strips the baked Astro backend proxy prefix so shipped assets can
// keep using the same /_backend/* contract they use in local frontend development.
func FrontendBackendProxyRewrite(prefix string) echo.MiddlewareFunc {
	normalized := strings.TrimSpace(prefix)
	if normalized == "" {
		normalized = frontendBackendProxyBase
	}

	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	normalized = strings.TrimRight(normalized, "/")
	if normalized == "" {
		normalized = "/"
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			if req.URL == nil {
				return next(c)
			}

			if rewritten, ok := stripFrontendBackendProxyPrefix(req.URL.Path, normalized); ok {
				req.URL.Path = rewritten
			}

			if req.URL.RawPath != "" {
				if rewritten, ok := stripFrontendBackendProxyPrefix(req.URL.RawPath, normalized); ok {
					req.URL.RawPath = rewritten
				}
			}

			return next(c)
		}
	}
}

func stripFrontendBackendProxyPrefix(path, prefix string) (string, bool) {
	if path != prefix && !strings.HasPrefix(path, prefix+"/") {
		return path, false
	}

	rewritten := strings.TrimPrefix(path, prefix)
	if rewritten == "" {
		return "/", true
	}

	return rewritten, true
}
