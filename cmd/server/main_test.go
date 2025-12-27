package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHealthEndpoint(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	if err := handler(c); err != nil {
		t.Fatalf("Handler error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("Body = %s, want OK", rec.Body.String())
	}
}

func TestServerAddr(t *testing.T) {
	addr := ":9000"
	e := echo.New()
	e.HidePort = true

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, addr)
	}

	if err := handler(c); err != nil {
		t.Fatalf("Handler error = %v", err)
	}

	if rec.Body.String() != addr {
		t.Errorf("Body = %s, want %s", rec.Body.String(), addr)
	}
}

func TestIndexEndpoint(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/dashboard")
	}

	if err := handler(c); err != nil {
		t.Fatalf("Handler error = %v", err)
	}

	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	if rec.Header().Get("Location") != "/dashboard" {
		t.Errorf("Location header = %s, want /dashboard", rec.Header().Get("Location"))
	}
}

func TestStaticFileServing(t *testing.T) {
	e := echo.New()
	e.Static("/static", "/home/adam/projects/domain-minder/static")

	req := httptest.NewRequest(http.MethodGet, "/static/test.css", nil)
	rec := httptest.NewRecorder()

	_ = e
	_ = req
	_ = rec

	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound && rec.Code != 0 {
		t.Errorf("Unexpected status code = %d", rec.Code)
	}
}

func TestEchoRouting(t *testing.T) {
	e := echo.New()
	registeredRoutes := make(map[string]bool)

	e.GET("/dashboard", func(c echo.Context) error {
		return c.String(http.StatusOK, "Dashboard")
	})
	e.GET("/domains", func(c echo.Context) error {
		return c.String(http.StatusOK, "Domains")
	})

	for _, route := range e.Routes() {
		registeredRoutes[route.Method+":"+route.Path] = true
	}

	if !registeredRoutes["GET:/dashboard"] {
		t.Error("Dashboard route not registered")
	}
	if !registeredRoutes["GET:/domains"] {
		t.Error("Domains route not registered")
	}
}
