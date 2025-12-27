package handlers

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/CaffeinatedTech/domain-minder/internal/services"
	"github.com/labstack/echo/v4"
)

type DomainHandler struct {
	whoisService *services.WHOISService
}

func NewDomainHandler(whoisService *services.WHOISService) *DomainHandler {
	return &DomainHandler{whoisService: whoisService}
}

func (h *DomainHandler) ListDomains(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domains, err := database.GetDomainsByUserID(c.Request().Context(), user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domains")
	}

	return c.Render(http.StatusOK, "domains", map[string]interface{}{
		"TotalDomains": len(domains),
	})
}

func (h *DomainHandler) ShowAddDomain(c echo.Context) error {
	return c.Render(http.StatusOK, "add_domain", nil)
}

func (h *DomainHandler) ListDomainsPartial(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "not authenticated")
	}

	domains, err := database.GetDomainsByUserID(c.Request().Context(), user.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to get domains")
	}

	hideActions := c.QueryParam("hide_actions") == "true"

	domainRows := make([]map[string]interface{}, 0, len(domains))
	for _, d := range domains {
		daysLeft := int(time.Until(d.ExpiryDate).Hours() / 24)
		colorClass := getColorClass(daysLeft)
		percent := getPercent(daysLeft, 365)

		domainRows = append(domainRows, map[string]interface{}{
			"Domain":   d,
			"DaysLeft": daysLeft,
			"Color":    colorClass,
			"Percent":  percent,
		})
	}

	return c.Render(http.StatusOK, "domain_row", map[string]interface{}{
		"Domains":     domainRows,
		"HideActions": hideActions,
	})
}

func (h *DomainHandler) AddDomain(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domainName := c.FormValue("name")
	if !isValidDomain(domainName) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain name")
	}

	result, err := h.whoisService.Lookup(c.Request().Context(), domainName)
	if err != nil {
		result = &services.WHOISResult{
			DomainName:    domainName,
			ExpiryMissing: true,
			Error:         err,
		}
	}

	registrar := result.Registrar
	if registrar == "" {
		registrar = "Unknown"
	}

	// If expiry date couldn't be determined, redirect to confirmation page
	if result.ExpiryMissing || result.ExpiryDate.IsZero() {
		whoisRaw := ""
		if result.WHOISRaw != "" {
			whoisRaw = result.WHOISRaw
		}
		return c.Render(http.StatusOK, "add_domain_confirm", map[string]interface{}{
			"DomainName": domainName,
			"Registrar":  registrar,
			"WHOISRaw":   whoisRaw,
		})
	}

	expiryDate := result.ExpiryDate
	whoisRaw := ""
	if result.WHOISRaw != "" {
		whoisRaw = result.WHOISRaw
	}

	domain := &models.Domain{
		UserID:       user.ID,
		Name:         domainName,
		Registrar:    &registrar,
		ExpiryDate:   expiryDate,
		ManualExpiry: false,
		WHOISRaw:     &whoisRaw,
		Status:       "active",
	}

	_, err = database.CreateDomain(c.Request().Context(), domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create domain")
	}

	return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape("Domain added successfully"))
}

// AddDomainConfirm handles the confirmation form when WHOIS didn't return an expiry date
func (h *DomainHandler) AddDomainConfirm(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domainName := c.FormValue("name")
	if !isValidDomain(domainName) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain name")
	}

	registrar := c.FormValue("registrar")
	if registrar == "" {
		registrar = "Unknown"
	}

	expiryDateStr := c.FormValue("expiry_date")
	expiryDate, err := time.Parse("2006-01-02", expiryDateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid expiry date format")
	}

	whoisRaw := c.FormValue("whois_raw")

	domain := &models.Domain{
		UserID:       user.ID,
		Name:         domainName,
		Registrar:    &registrar,
		ExpiryDate:   expiryDate,
		ManualExpiry: true, // User provided this date manually
		WHOISRaw:     &whoisRaw,
		Status:       "active",
	}

	_, err = database.CreateDomain(c.Request().Context(), domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create domain")
	}

	return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape("Domain added successfully"))
}

func (h *DomainHandler) ShowEditDomain(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domainID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
	}

	domain, err := database.GetDomainByID(c.Request().Context(), domainID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
	}
	if domain == nil {
		return echo.NewHTTPError(http.StatusNotFound, "domain not found")
	}
	if domain.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	return c.Render(http.StatusOK, "edit_domain", map[string]interface{}{
		"Domain": domain,
	})
}

func (h *DomainHandler) UpdateDomain(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domainID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
	}

	domain, err := database.GetDomainByID(c.Request().Context(), domainID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
	}
	if domain == nil {
		return echo.NewHTTPError(http.StatusNotFound, "domain not found")
	}
	if domain.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	domain.Name = c.FormValue("name")
	registrar := c.FormValue("registrar")
	domain.Registrar = &registrar
	expiryDateStr := c.FormValue("expiry_date")
	expiryDate, err := time.Parse("2006-01-02", expiryDateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid expiry date format")
	}
	domain.ExpiryDate = expiryDate
	domain.ManualExpiry = true // User manually edited the expiry date
	domain.Status = c.FormValue("status")
	notes := c.FormValue("notes")
	domain.Notes = &notes

	if err := database.UpdateDomain(c.Request().Context(), domain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update domain")
	}

	return c.Redirect(http.StatusSeeOther, "/domains")
}

func (h *DomainHandler) DeleteDomain(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domainID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
	}

	domain, err := database.GetDomainByID(c.Request().Context(), domainID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
	}
	if domain == nil {
		return echo.NewHTTPError(http.StatusNotFound, "domain not found")
	}
	if domain.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	if err := database.DeleteDomain(c.Request().Context(), domainID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete domain")
	}

	isHTMX := c.Request().Header.Get("HX-Request") == "true"
	if isHTMX {
		return c.String(http.StatusOK, "")
	}

	return c.Redirect(http.StatusSeeOther, "/domains")
}

func (h *DomainHandler) CheckDomain(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	domainID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
	}

	domain, err := database.GetDomainByID(c.Request().Context(), domainID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
	}
	if domain == nil {
		return echo.NewHTTPError(http.StatusNotFound, "domain not found")
	}
	if domain.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	result, err := h.whoisService.Lookup(c.Request().Context(), domain.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "WHOIS lookup failed: "+err.Error())
	}

	message := "WHOIS check completed"

	// Only update expiry if NOT manually set
	if !domain.ManualExpiry && !result.ExpiryDate.IsZero() {
		if err := database.UpdateDomainWHOIS(c.Request().Context(), domainID, result.ExpiryDate, result.Registrar, result.WHOISRaw); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update domain")
		}
	} else if domain.ManualExpiry {
		message = "WHOIS check completed (expiry date kept, manually set)"
	}

	return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape(message))
}

func isValidDomain(name string) bool {
	if name == "" || len(name) < 4 || len(name) > 253 {
		return false
	}
	matched, err := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$`, name)
	if err != nil {
		return false
	}
	return matched
}

func selected(current, value string) string {
	if current == value {
		return " selected"
	}
	return ""
}

func getColorClass(daysLeft int) string {
	switch {
	case daysLeft < 0:
		return "progress-red"
	case daysLeft < 7:
		return "progress-critical"
	case daysLeft < 14:
		return "progress-red"
	case daysLeft < 30:
		return "progress-orange"
	case daysLeft < 60:
		return "progress-yellow"
	default:
		return "progress-green"
	}
}

func getPercent(daysLeft, maxDays int) float64 {
	if daysLeft <= 0 {
		return 100
	}
	if daysLeft > maxDays {
		return 5
	}
	return float64(daysLeft) / float64(maxDays) * 100
}
