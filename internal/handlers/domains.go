package handlers

import (
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/CaffeinatedTech/domain-minder/internal/services"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
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

	var html string
	if len(domains) == 0 {
		html = `<p>No domains added yet.</p>`
	} else {
		html = `<table border="1"><tr><th>Domain</th><th>Registrar</th><th>Expiry</th><th>Days Left</th><th>Status</th><th>Actions</th></tr>`
		for _, d := range domains {
			daysLeft := int(time.Until(d.ExpiryDate).Hours() / 24)
			status := d.Status
			if daysLeft < 0 {
				status = "expired"
			}
			html += `<tr>
				<td>` + d.Name + `</td>
				<td>` + nullString(d.Registrar) + `</td>
				<td>` + d.ExpiryDate.Format("2006-01-02") + `</td>
				<td>` + strconv.Itoa(daysLeft) + `</td>
				<td>` + status + `</td>
				<td>
					<form method="POST" action="/domains/` + strconv.Itoa(d.ID) + `/delete" style="display:inline;">
						<button type="submit" onclick="return confirm('Delete this domain?')">Delete</button>
					</form>
				</td>
			</tr>`
		}
		html += `</table>`
	}

	message := c.QueryParam("message")
	if message != "" {
		message = `<p style="color: green;">` + message + `</p>`
	}

	return c.String(http.StatusOK, `
    <html><body>
    <h1>Your Domains</h1>
    `+message+`
    `+html+`
    <h2>Add New Domain</h2>
    <form method="POST" action="/domains">
        <label>Domain Name: <input type="text" name="name" placeholder="example.com" required></label><br>
        <button type="submit">Add Domain</button>
    </form>
    <p><small>WHOIS lookup will be performed to find expiry date and registrar.</small></p>
    <a href="/dashboard">Back to Dashboard</a>
    </body></html>
    `)
}

func (h *DomainHandler) ShowAddDomain(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	return c.String(http.StatusOK, `
    <html><body>
    <h1>Add Domain</h1>
    <form method="POST" action="/domains">
        <label>Domain Name: <input type="text" name="name" required></label><br>
        <button type="submit">Add Domain</button>
    </form>
    <a href="/domains">Cancel</a>
    </body></html>
    `)
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

	result, err := h.whoisService.Lookup(domainName)
	if err != nil {
		result = &services.WHOISResult{
			DomainName: domainName,
			Error:      err,
		}
	}

	registrar := result.Registrar
	if registrar == "" {
		registrar = "Unknown"
	}

	expiryDate := result.ExpiryDate
	if expiryDate.IsZero() {
		expiryDate = time.Now().AddDate(1, 0, 0)
	}

	whoisRaw := ""
	if result.WHOISRaw != "" {
		whoisRaw = result.WHOISRaw
	}

	domain := &models.Domain{
		UserID:     user.ID,
		Name:       domainName,
		Registrar:  &registrar,
		ExpiryDate: expiryDate,
		WHOISRaw:   &whoisRaw,
		Status:     "active",
	}

	_, err = database.CreateDomain(c.Request().Context(), domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create domain")
	}

	message := "Domain added successfully"
	if result.Error != nil {
		message += " (WHOIS lookup failed, default expiry date set)"
	}

	return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape(message))
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

	return c.String(http.StatusOK, `
    <html><body>
    <h1>Edit Domain</h1>
    <form method="POST" action="/domains/`+strconv.Itoa(domain.ID)+`">
        <label>Domain Name: <input type="text" name="name" value="`+domain.Name+`" required></label><br>
        <label>Registrar: <input type="text" name="registrar" value="`+nullString(domain.Registrar)+`"></label><br>
        <label>Expiry Date: <input type="date" name="expiry_date" value="`+domain.ExpiryDate.Format("2006-01-02")+`" required></label><br>
        <label>Status:
            <select name="status">
                <option value="active"`+selected(domain.Status, "active")+`>Active</option>
                <option value="expired"`+selected(domain.Status, "expired")+`>Expired</option>
                <option value="pending"`+selected(domain.Status, "pending")+`>Pending</option>
            </select>
        </label><br>
        <label>Notes:<br><textarea name="notes">`+nullString(domain.Notes)+`</textarea></label><br>
        <button type="submit">Save</button>
    </form>
    <a href="/domains">Cancel</a>
    </body></html>
    `)
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

	result, err := h.whoisService.Lookup(domain.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "WHOIS lookup failed: "+err.Error())
	}

	if !result.ExpiryDate.IsZero() {
		if err := database.UpdateDomainWHOIS(c.Request().Context(), domainID, result.ExpiryDate, result.Registrar, result.WHOISRaw); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update domain")
		}
	}

	return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape("WHOIS check completed"))
}

func isValidDomain(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]*\.[a-zA-Z]{2,}$`, name)
	return matched
}

func selected(current, value string) string {
	if current == value {
		return " selected"
	}
	return ""
}
