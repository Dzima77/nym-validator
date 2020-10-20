// Copyright 2020 Nym Technologies SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"github.com/nymtech/nym/validator/nym/directory/presence"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nymtech/nym/validator/nym/directory/healthcheck"
	"github.com/nymtech/nym/validator/nym/directory/mixmining"
	"github.com/nymtech/nym/validator/nym/directory/server/html"
	"github.com/nymtech/nym/validator/nym/directory/server/websocket"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// New returns a new REST API server
// @title Nym Directory API
// @version 0.9.0-dev
// @description A directory API allowing Nym nodes and clients to connect to each other.
// @termsOfService http://swagger.io/terms/
// @license.name Apache 2.0
// @license.url https://github.com/nymtech/nym-validator/license
func New() *gin.Engine {
	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Add cors middleware
	router.Use(cors.Default())

	// Serve Swagger frontend static files using gin-swagger middleware
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Add HTML templates to the router
	t, err := html.LoadTemplate()
	if err != nil {
		panic(err)
	}
	router.SetHTMLTemplate(t)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "/server/html/index.html", nil)
	})

	// Set up websocket handlers
	hub := websocket.NewHub()
	go hub.Run()
	router.GET("/ws", func(c *gin.Context) {
		websocket.Serve(hub, c.Writer, c.Request)
	})

	// Sanitize controller input against XSS attacks using bluemonday.Policy
	policy := bluemonday.UGCPolicy()

	// Measurements: wire up dependency injection
	measurementsCfg := injectMeasurements(policy)

	// Presence: wire up dependency injection
	presenceCfg := injectPresence(policy)

	// Register all HTTP controller routes
	healthcheck.New().RegisterRoutes(router)
	mixmining.New(measurementsCfg).RegisterRoutes(router)
	presence.New(presenceCfg).RegisterRoutes(router)

	return router
}

func injectPresence(policy *bluemonday.Policy) presence.Config {
	sanitizer := presence.NewSanitizer(policy)
	db := presence.NewDb()
	presenceService := *presence.NewService(db)

	return presence.Config{
		Service:   &presenceService,
		Sanitizer: sanitizer,
	}
}

func injectMeasurements(policy *bluemonday.Policy) mixmining.Config {
	sanitizer := mixmining.NewSanitizer(policy)
	db := mixmining.NewDb()
	mixminingService := *mixmining.NewService(db)

	return mixmining.Config{
		Service:   &mixminingService,
		Sanitizer: sanitizer,
	}
}
