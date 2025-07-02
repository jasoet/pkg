module ecommerce-api

go 1.23.0

toolchain go1.24.2

require (
	github.com/jasoet/pkg v1.1.0
	github.com/labstack/echo/v4 v4.13.4
	github.com/rs/zerolog v1.34.0
	gorm.io/gorm v1.30.0
)

// For local development, replace with local path
// replace github.com/jasoet/pkg => ../../