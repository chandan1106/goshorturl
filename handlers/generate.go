package handlers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"shorturl/db"

	"github.com/gofiber/fiber/v2"
)

type GenerateRequest struct {
	CampaignName string `json:"campignName"`
	MainURL      string `json:"mainUrl"`
	APIKey       string `json:"apikey"`
	Type         string `json:"type"`
	ShortCode    string `json:"shortcode,omitempty"`
	Expiry       string `json:"expiry"`
	Count        string `json:"count"`
	SenderID     string `json:"senderId,omitempty"`
	Domain       string `json:"domain,omitempty"`
}

var validAPIKeys = []string{
	"abcjdakfdsnkndskvn",
}

func isValidAPIKey(key string) bool {
	for _, k := range validAPIKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Build short URL
func buildShortURL(domain, sender, shortcode string) string {
	if sender != "" {
		sender = sender + "/"
	}
	return fmt.Sprintf("%s%s%s", domain, sender, shortcode)
}

// Insert into mainurl table
func insertMainURL(ctx context.Context, longURL, shortcode string, expiryTime time.Time, domain, senderID, createdBy, campaignName string) (int, error) {
	var insertID int
	// Use the parameters directly, not req/sc
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO mainurl
		(longurl, shortcode, expirytime, domain, senderid, createdby, campaignname)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id
	`, longURL, shortcode, expiryTime, domain, senderID, createdBy, campaignName).Scan(&insertID)

	if err != nil {
		fmt.Println("DB insert error:", err)
		return 0, err
	}

	return insertID, nil
}

func GenerateShortURL(c *fiber.Ctx) error {
	var req GenerateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON payload", "status": "error"})
	}

	// Validate required fields
	if req.CampaignName == "" || req.MainURL == "" || req.APIKey == "" || req.Type == "" || req.Expiry == "" || req.Count == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Missing required fields", "status": "error"})
	}

	if _, err := url.ParseRequestURI(req.MainURL); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid mainUrl", "status": "error"})
	}

	if !isValidAPIKey(req.APIKey) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid API key", "status": "error"})
	}

	countInt, err := strconv.Atoi(req.Count)
	if err != nil || countInt < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Count must be a valid number", "status": "error"})
	}

	expiryDays, err := strconv.Atoi(req.Expiry)
	if err != nil || expiryDays < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Expiry Time must be a valid number", "status": "error"})
	}
	expiryTime := time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour)

	// Default domain and createdBy
	domain := "https://velocity.veup.io/shorturl/"
	if req.Domain != "" {
		domain = req.Domain
	}

	createdBy := "chandan"

	responseMap := make(map[string]string)
	ctx := context.Background()

	switch strings.ToLower(req.Type) {
	case "custom":
		if req.ShortCode == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Shortcode required for custom type", "status": "error"})
		}
		shortURL := buildShortURL(domain, req.SenderID, req.ShortCode)
		insertID, err := insertMainURL(ctx, req.MainURL, req.ShortCode, expiryTime, domain, req.SenderID, createdBy, req.CampaignName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to insert custom shorturl", "status": "error"})
		}
		responseMap[strconv.Itoa(insertID)] = shortURL

	case "unique":
		rows, err := db.Pool.Query(ctx, `SELECT id, shortcode FROM shortcodes WHERE status = 0 ORDER BY id LIMIT $1`, countInt)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to fetch shortcodes", "status": "error"})
		}
		defer rows.Close()

		var shortcodeIDs []int
		var shortcodes []string
		for rows.Next() {
			var id int
			var sc string
			if err := rows.Scan(&id, &sc); err != nil {
				continue
			}
			shortcodeIDs = append(shortcodeIDs, id)
			shortcodes = append(shortcodes, sc)
		}

		if len(shortcodes) == 0 {
			return c.JSON(fiber.Map{"message": "No available shortcodes", "status": "empty"})
		}

		if _, err := db.Pool.Exec(ctx, `UPDATE shortcodes SET status = 1, taken_timestamp = NOW() WHERE id = ANY($1::int[])`, shortcodeIDs); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update shortcodes", "status": "error"})
		}

		for _, sc := range shortcodes {
			shortURL := buildShortURL(domain, req.SenderID, sc)
			insertID, err := insertMainURL(ctx, req.MainURL, sc, expiryTime, domain, req.SenderID, createdBy, req.CampaignName)
			if err != nil {
				continue
			}
			responseMap[strconv.Itoa(insertID)] = shortURL
		}

	default: // generic
		if countInt != 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Generic type can only have count = 1", "status": "error"})
		}

		var id int
		var sc string
		if err := db.Pool.QueryRow(ctx, `SELECT id, shortcode FROM shortcodes WHERE status = 0 ORDER BY id LIMIT 1`).Scan(&id, &sc); err != nil {
			return c.JSON(fiber.Map{"message": "No available shortcodes", "status": "empty"})
		}

		if _, err := db.Pool.Exec(ctx, `UPDATE shortcodes SET status = 1, taken_timestamp = NOW() WHERE id=$1`, id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update shortcode", "status": "error"})
		}

		shortURL := buildShortURL(domain, req.SenderID, sc)
		insertID, err := insertMainURL(ctx, req.MainURL, sc, expiryTime, domain, req.SenderID, createdBy, req.CampaignName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to insert generic shorturl", "status": "error"})
		}
		responseMap[strconv.Itoa(insertID)] = shortURL
	}

	return c.JSON(responseMap)
}
