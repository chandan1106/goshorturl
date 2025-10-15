package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"shorturl/db"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
)

type GeoIPResponse struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	PostalCode  string  `json:"postal_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Org         string  `json:"organization"`
	Timezone    string  `json:"timezone"`
}

// RedirectShortURL handles redirecting short URLs and logging clicks
func RedirectShortURL(c *fiber.Ctx) error {
	ctx := context.Background()

	path := c.Params("*")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid short URL")
	}
	shortcode := parts[len(parts)-1]

	var longURL string
	err := db.Pool.QueryRow(ctx, `
		SELECT longurl
		FROM mainurl
		WHERE shortcode=$1 AND status=0
		LIMIT 1
	`, shortcode).Scan(&longURL)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Short URL not found or expired")
	}

	// Collect visitor info
	ip := c.IP()
	device := c.Get("User-Agent")

	// Initialize GeoIP fields
	var geo GeoIPResponse

	// Fetch geolocation from Fortnic API
	geoURL := fmt.Sprintf("https://geoip.fortnic.com/%s?format=json", ip)
	client := resty.New()
	resp, err := client.R().Get(geoURL)
	if err == nil && resp.StatusCode() == 200 {
		_ = json.Unmarshal(resp.Body(), &geo)
	}

	// Insert click info into creport table
	_, err = db.Pool.Exec(ctx, `
	INSERT INTO creport
	(shortcode, clicks, ip, location, device, country_code, country, region, city, postal_code, latitude, longitude, organization, timezone)
	VALUES
	($1, 1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, shortcode, ip, geo.City, device, geo.CountryCode, geo.Country, geo.Region, geo.City, geo.PostalCode, geo.Latitude, geo.Longitude, geo.Org, geo.Timezone)
	if err != nil {
		fmt.Println("Failed to insert click report:", err)
	}

	return c.Redirect(longURL, fiber.StatusTemporaryRedirect)
}
