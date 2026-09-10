package main

import (
	"log"

	"github.com/xenvoid404/gosip"
)

func main() {
	r := gosip.New()
	r.Use(gosip.Recover())

	r.SetNotFoundHandler(func(c *gosip.Context) error {
		return c.Status(gosip.StatusNotFound).JSON(gosip.Map{
			"success": false,
			"message": "not found",
		})
	})

	r.SetMethodNotAllowedHandler(func(c *gosip.Context) error {
		return c.Status(gosip.StatusMethodNotAllowed).JSON(gosip.Map{
			"success": false,
			"message": "method not allowed",
		})
	})

	r.SetErrorHandler(func(err error, c *gosip.Context) error {
		return c.Status(gosip.StatusInternalServerError).JSON(gosip.Map{
			"success": false,
			"message": "internal server error",
		})
	})

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

	r.Get("/panic", func(c *gosip.Context) error {
		panic("boom")
	})

	if err := r.Listen(":8000"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
