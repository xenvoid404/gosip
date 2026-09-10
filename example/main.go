package main

import (
	"log"

	"github.com/xenvoid404/gosip"
)

func main() {
	r := gosip.New()
	r.Get("/carmen", func(c *gosip.Context) error {
		return c.Status(gosip.StatusOK).JSON(gosip.Map{
			"success": true,
			"message": "cantik banget",
		})
	})
	r.Get("/carmen/{id}", func(c *gosip.Context) error {
		id := c.Params("id")
		return c.Status(gosip.StatusOK).JSON(gosip.Map{
			"success": true,
			"message": "carmen" + id,
		})
	})

	if err := r.Listen(":8000"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
