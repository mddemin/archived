package html

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strconv"

	sprig "github.com/Masterminds/sprig/v3"
	echo "github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"

	"github.com/teran/archived/models"
	"github.com/teran/archived/service"
)

const (
	notFoundTemplateFilename    = "404.html"
	serverErrorTemplateFilename = "5xx.html"
)

type Handlers interface {
	ContainerIndex(c echo.Context) error
	VersionIndex(c echo.Context) error

	Register(e *echo.Echo)
}

type handlers struct {
	svc         service.Publisher
	staticDir   string
	templateDir string
}

func New(svc service.Publisher, templateDir, staticDir string) Handlers {
	return &handlers{
		svc:         svc,
		staticDir:   staticDir,
		templateDir: templateDir,
	}
}

func (h *handlers) ContainerIndex(c echo.Context) error {
	containers, err := h.svc.ListContainers(c.Request().Context())
	if err != nil {
		return err
	}

	type data struct {
		Title      string
		Containers []string
	}

	return c.Render(http.StatusOK, "container-list.html", &data{
		Title:      "Container index",
		Containers: containers,
	})
}

func (h *handlers) VersionIndex(c echo.Context) error {
	container := c.Param("container")

	pageParam := c.QueryParam("page")
	var page uint64 = 1

	var err error
	if pageParam != "" {
		page, err = strconv.ParseUint(pageParam, 10, 64)
		if err != nil {
			log.Warnf("malformed page parameter: `%s`", pageParam)
			page = 1
		}
	}

	pagesCount, versions, err := h.svc.ListPublishedVersionsByPage(c.Request().Context(), container, page)
	if err != nil {
		if err == service.ErrNotFound {
			return c.Render(http.StatusNotFound, notFoundTemplateFilename, nil)
		}
		return err
	}

	type data struct {
		Title       string
		CurrentPage uint64
		PagesCount  uint64
		Container   string
		Versions    []models.Version
	}

	return c.Render(http.StatusOK, "version-list.html", &data{
		Title:       fmt.Sprintf("Version index (%s)", container),
		CurrentPage: page,
		PagesCount:  pagesCount,
		Container:   container,
		Versions:    versions,
	})
}

func (h *handlers) ObjectIndex(c echo.Context) error {
	container := c.Param("container")
	version := c.Param("version")

	pageParam := c.QueryParam("page")
	var page uint64 = 1

	var err error
	if pageParam != "" {
		page, err = strconv.ParseUint(pageParam, 10, 64)
		if err != nil {
			log.Warnf("malformed page parameter: `%s`", pageParam)
			page = 1
		}
	}

	pagesCount, objects, err := h.svc.ListObjectsByPage(c.Request().Context(), container, version, page)
	if err != nil {
		if err == service.ErrNotFound {
			return c.Render(http.StatusNotFound, notFoundTemplateFilename, nil)
		}
		return err
	}

	type data struct {
		Title       string
		CurrentPage uint64
		PagesCount  uint64
		Container   string
		Version     string
		Objects     []string
	}
	return c.Render(http.StatusOK, "object-list.html", &data{
		Title:       fmt.Sprintf("Object index (%s/%s)", container, version),
		CurrentPage: page,
		PagesCount:  pagesCount,
		Container:   container,
		Version:     version,
		Objects:     objects,
	})
}

func (h *handlers) GetObject(c echo.Context) error {
	container := c.Param("container")
	version := c.Param("version")

	objectParam := c.Param("object")
	key, err := url.PathUnescape(objectParam)
	if err != nil {
		return err
	}

	url, err := h.svc.GetObjectURL(c.Request().Context(), container, version, key)
	if err != nil {
		if err == service.ErrNotFound {
			return c.Render(http.StatusNotFound, notFoundTemplateFilename, nil)
		}
		return err
	}

	return c.Redirect(http.StatusFound, url)
}

func (h *handlers) ErrorHandler(err error, c echo.Context) {
	code := 500
	templateFilename := serverErrorTemplateFilename

	v, ok := err.(*echo.HTTPError)
	if ok {
		code = v.Code

		switch v.Code {
		case http.StatusNotFound:
			code = http.StatusNotFound
			templateFilename = notFoundTemplateFilename
		}
	}

	type data struct {
		Code    int
		Message string
	}

	if err := c.Render(code, templateFilename, &data{
		Code:    code,
		Message: http.StatusText(code),
	}); err != nil {
		c.Logger().Error(err)
	}
}

func (h *handlers) Register(e *echo.Echo) {
	e.Renderer = &renderer{
		templates: template.Must(
			template.New("base").Funcs(sprig.FuncMap()).ParseGlob(path.Join(h.templateDir, "*.html")),
		),
	}

	e.HTTPErrorHandler = h.ErrorHandler

	e.GET("/", h.ContainerIndex)
	e.GET("/:container/", h.VersionIndex)
	e.GET("/:container/:version/", h.ObjectIndex)
	e.GET("/:container/:version/:object", h.GetObject)

	e.Static(h.staticDir, "static")
}
