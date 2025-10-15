package handlers

import (
	"context"
	"fmt"
	"time"

	"shorturl/db"

	"github.com/gofiber/fiber/v2"
)

// ReportRequest defines the request structure
type ReportRequest struct {
	CampaignName string `json:"campaignName,omitempty"`
	ShortCode    string `json:"shortcode,omitempty"`
	ReportType   string `json:"reportType"` // "summary" or "detailed"
}

// Helper functions for nullable fields
func safeString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func safeFloat(f *float64) float64 {
	if f != nil {
		return *f
	}
	return 0
}

// GetClickReport handles fetching reports
func GetClickReport(c *fiber.Ctx) error {
	ctx := context.Background()

	var req ReportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid JSON payload",
		})
	}

	if req.ReportType != "summary" && req.ReportType != "detailed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "reportType must be 'summary' or 'detailed'",
		})
	}

	args := []interface{}{}
	argID := 1
	query := ""

	if req.ReportType == "summary" {
		query = `
		SELECT cr.shortcode, mu.campaignname, SUM(cr.clicks) as clicks
		FROM creport cr
		JOIN mainurl mu ON cr.shortcode = mu.shortcode
		WHERE 1=1
		`
		if req.CampaignName != "" {
			query += fmt.Sprintf(" AND mu.campaignname ILIKE $%d", argID)
			args = append(args, "%"+req.CampaignName+"%")
			argID++
		}
		if req.ShortCode != "" {
			query += fmt.Sprintf(" AND cr.shortcode ILIKE $%d", argID)
			args = append(args, "%"+req.ShortCode+"%")
			argID++
		}
		query += " GROUP BY cr.shortcode, mu.campaignname ORDER BY cr.shortcode"
	} else { // detailed
		query = `
		SELECT cr.shortcode, mu.campaignname, cr.clicks, cr.ip, cr.device, cr.country_code, cr.country,
		       cr.region, cr.city, cr.postal_code, cr.latitude, cr.longitude, cr.organization,
		       cr.timezone, cr.time
		FROM creport cr
		JOIN mainurl mu ON cr.shortcode = mu.shortcode
		WHERE 1=1
		`
		if req.CampaignName != "" {
			query += fmt.Sprintf(" AND mu.campaignname ILIKE $%d", argID)
			args = append(args, "%"+req.CampaignName+"%")
			argID++
		}
		if req.ShortCode != "" {
			query += fmt.Sprintf(" AND cr.shortcode ILIKE $%d", argID)
			args = append(args, "%"+req.ShortCode+"%")
			argID++
		}
		query += " ORDER BY cr.time DESC"
	}

	//fmt.Println("Query:", query)
	//fmt.Println("Args:", args)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		//fmt.Println("Query error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to fetch report",
		})
	}
	defer rows.Close()

	reports := []map[string]interface{}{}
	rowCount := 0

	for rows.Next() {
		rowCount++

		if req.ReportType == "summary" {
			var shortcode, campaignName string
			var clicks int
			if err := rows.Scan(&shortcode, &campaignName, &clicks); err != nil {
				//fmt.Println("Scan error:", err)
				continue
			}
			reports = append(reports, map[string]interface{}{
				"shortcode":    shortcode,
				"campaignName": campaignName,
				"clicks":       clicks,
			})
		} else { // detailed
			var shortcode, campaignName string
			var clicks int
			var ip, device, countryCode, country, region, city, postalCode, organization, timezone *string
			var latitude, longitude *float64
			var timeStamp *time.Time

			if err := rows.Scan(&shortcode, &campaignName, &clicks, &ip, &device, &countryCode, &country,
				&region, &city, &postalCode, &latitude, &longitude, &organization, &timezone, &timeStamp); err != nil {
				//fmt.Println("Scan error:", err)
				continue
			}

			reports = append(reports, map[string]interface{}{
				"shortcode":    shortcode,
				"campaignName": campaignName,
				"clicks":       clicks,
				"ip":           safeString(ip),
				"device":       safeString(device),
				"country_code": safeString(countryCode),
				"country":      safeString(country),
				"region":       safeString(region),
				"city":         safeString(city),
				"postal_code":  safeString(postalCode),
				"latitude":     safeFloat(latitude),
				"longitude":    safeFloat(longitude),
				"organization": safeString(organization),
				"timezone":     safeString(timezone),
				"time": func() string {
					if timeStamp != nil {
						return timeStamp.Format("2006-01-02 15:04:05")
					}
					return ""
				}(),
			})
		}
	}

	//	fmt.Println("Rows returned:", rowCount)
	return c.JSON(fiber.Map{
		"status":  "success",
		"reports": reports,
	})
}
