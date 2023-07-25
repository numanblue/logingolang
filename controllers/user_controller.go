// controllers/user_controller.go

package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func ShowLoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", nil)
}
