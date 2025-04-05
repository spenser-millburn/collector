package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/observability-collector/internal/api/docs"
	"github.com/observability-collector/internal/core"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// API represents the REST API for the observability collector
type API struct {
	app    *core.App
	router *gin.Engine
	server *http.Server
	port   int
	host   string
}

// NewAPI creates a new API instance
// @title           Observability Collector API
// @version         1.0
// @description     API for controlling the observability collector
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.basic  BasicAuth
func NewAPI(app *core.App, port int, host string) *API {
	// Set up Swagger info
	docs.SwaggerInfo.Title = "Observability Collector API"
	docs.SwaggerInfo.Description = "API for controlling the observability collector"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", host, port)
	docs.SwaggerInfo.BasePath = "" // Empty to match routes as they appear in Swagger UI
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Create router
	router := gin.Default()

	// Create API instance
	api := &API{
		app:    app,
		router: router,
		port:   port,
		host:   host,
	}

	// Set up routes
	api.setupRoutes()

	return api
}

// setupRoutes configures all the API routes
func (a *API) setupRoutes() {
	// Use root router for simplicity and to match Swagger docs
	// Health check
	a.router.GET("/health", a.healthCheck)

	// Status endpoints
	a.router.GET("/status", a.getStatus)
	
	// Plugin management
	plugins := a.router.Group("/plugins")
	{
		plugins.GET("", a.getPlugins)
		plugins.GET("/:type", a.getPluginsByType)
		plugins.GET("/:type/:name", a.getPluginByName)
		plugins.POST("/:type/:name/start", a.startPlugin)
		plugins.POST("/:type/:name/stop", a.stopPlugin)
		plugins.POST("/:type/:name/restart", a.restartPlugin)
	}

	// Buffer management
	buffers := a.router.Group("/buffers")
	{
		buffers.GET("", a.getBuffers)
		buffers.GET("/:name", a.getBufferByName)
		buffers.POST("/:name/flush", a.flushBuffer)
	}

	// Configuration
	a.router.GET("/config", a.getConfig)
	a.router.PUT("/config", a.updateConfig)
	
	// Controls
	a.router.POST("/start", a.startCollector)
	a.router.POST("/stop", a.stopCollector)
	a.router.POST("/restart", a.restartCollector)

	// Swagger documentation - serve at root URL for better discoverability
	url := ginSwagger.URL("/swagger.json") // The URL pointing to API definition
	a.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
	// Also serve the swagger.json directly
	a.router.GET("/swagger.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
}

// Start starts the API server
func (a *API) Start() error {
	addr := fmt.Sprintf("%s:%d", a.host, a.port)
	a.server = &http.Server{
		Addr:    addr,
		Handler: a.router,
	}

	return a.server.ListenAndServe()
}

// Stop stops the API server
func (a *API) Stop(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

// healthCheck handles GET /api/v1/health
// @Summary      Health check
// @Description  Check if the API is running
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func (a *API) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
	})
}

// getStatus handles GET /api/v1/status
// @Summary      Get system status
// @Description  Get the status of the collector and all its components
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /status [get]
func (a *API) getStatus(c *gin.Context) {
	status := a.app.GetAppStatus()
	c.JSON(http.StatusOK, status)
}

// getPlugins handles GET /api/v1/plugins
// @Summary      Get all plugins
// @Description  Get information about all registered plugins
// @Tags         plugins
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /plugins [get]
func (a *API) getPlugins(c *gin.Context) {
	status := a.app.GetAppStatus()
	plugins := status["plugins"]
	c.JSON(http.StatusOK, plugins)
}

// getPluginsByType handles GET /api/v1/plugins/:type
// @Summary      Get plugins by type
// @Description  Get information about plugins of a specific type
// @Tags         plugins
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Plugin type (input, processor, output)"
// @Success      200  {object}  map[string]interface{}
// @Router       /plugins/{type} [get]
func (a *API) getPluginsByType(c *gin.Context) {
	pluginType := c.Param("type")
	status := a.app.GetAppStatus()
	plugins := status["plugins"].(map[string]interface{})
	
	result := make(map[string]interface{})
	for name, plugin := range plugins {
		p := plugin.(map[string]interface{})
		if p["type"] == pluginType {
			result[name] = plugin
		}
	}
	
	c.JSON(http.StatusOK, result)
}

// getPluginByName handles GET /api/v1/plugins/:type/:name
// @Summary      Get plugin by name
// @Description  Get information about a specific plugin
// @Tags         plugins
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Plugin type (input, processor, output)"
// @Param        name    path    string  true  "Plugin name"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Router       /plugins/{type}/{name} [get]
func (a *API) getPluginByName(c *gin.Context) {
	pluginType := c.Param("type")
	pluginName := c.Param("name")
	
	status := a.app.GetAppStatus()
	plugins := status["plugins"].(map[string]interface{})
	
	if plugin, exists := plugins[pluginName]; exists {
		p := plugin.(map[string]interface{})
		if p["type"] == pluginType {
			c.JSON(http.StatusOK, plugin)
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
}

// startPlugin handles POST /api/v1/plugins/:type/:name/start
// @Summary      Start a plugin
// @Description  Start a specific plugin
// @Tags         plugins
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Plugin type (input, processor, output)"
// @Param        name    path    string  true  "Plugin name"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /plugins/{type}/{name}/start [post]
func (a *API) startPlugin(c *gin.Context) {
	pluginName := c.Param("name")
	
	err := a.app.StartPlugin(context.Background(), pluginName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Plugin started"})
}

// stopPlugin handles POST /api/v1/plugins/:type/:name/stop
// @Summary      Stop a plugin
// @Description  Stop a specific plugin
// @Tags         plugins
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Plugin type (input, processor, output)"
// @Param        name    path    string  true  "Plugin name"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /plugins/{type}/{name}/stop [post]
func (a *API) stopPlugin(c *gin.Context) {
	pluginName := c.Param("name")
	
	err := a.app.StopPlugin(context.Background(), pluginName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Plugin stopped"})
}

// restartPlugin handles POST /api/v1/plugins/:type/:name/restart
// @Summary      Restart a plugin
// @Description  Restart a specific plugin
// @Tags         plugins
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Plugin type (input, processor, output)"
// @Param        name    path    string  true  "Plugin name"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /plugins/{type}/{name}/restart [post]
func (a *API) restartPlugin(c *gin.Context) {
	pluginName := c.Param("name")
	
	err := a.app.StopPlugin(context.Background(), pluginName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop plugin: " + err.Error()})
		return
	}
	
	err = a.app.StartPlugin(context.Background(), pluginName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start plugin: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Plugin restarted"})
}

// getBuffers handles GET /api/v1/buffers
// @Summary      Get all buffers
// @Description  Get information about all buffers
// @Tags         buffers
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /buffers [get]
func (a *API) getBuffers(c *gin.Context) {
	status := a.app.GetAppStatus()
	buffers := status["buffers"]
	c.JSON(http.StatusOK, buffers)
}

// getBufferByName handles GET /api/v1/buffers/:name
// @Summary      Get buffer by name
// @Description  Get information about a specific buffer
// @Tags         buffers
// @Accept       json
// @Produce      json
// @Param        name    path    string  true  "Buffer name"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Router       /buffers/{name} [get]
func (a *API) getBufferByName(c *gin.Context) {
	bufferName := c.Param("name")
	
	status := a.app.GetAppStatus()
	buffers := status["buffers"].(map[string]interface{})
	
	if buffer, exists := buffers[bufferName]; exists {
		c.JSON(http.StatusOK, buffer)
		return
	}
	
	c.JSON(http.StatusNotFound, gin.H{"error": "Buffer not found"})
}

// flushBuffer handles POST /api/v1/buffers/:name/flush
// @Summary      Flush a buffer
// @Description  Flush a specific buffer
// @Tags         buffers
// @Accept       json
// @Produce      json
// @Param        name    path    string  true  "Buffer name"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buffers/{name}/flush [post]
func (a *API) flushBuffer(c *gin.Context) {
	bufferName := c.Param("name")
	
	err := a.app.FlushBuffer(bufferName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Buffer flushed"})
}

// getConfig handles GET /api/v1/config
// @Summary      Get configuration
// @Description  Get the current collector configuration
// @Tags         config
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /config [get]
func (a *API) getConfig(c *gin.Context) {
	config := a.app.GetConfig()
	c.JSON(http.StatusOK, config)
}

// updateConfig handles PUT /api/v1/config
// @Summary      Update configuration
// @Description  Update the collector configuration
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        config  body    map[string]interface{}  true  "New configuration"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /config [put]
func (a *API) updateConfig(c *gin.Context) {
	var newConfig map[string]interface{}
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration format"})
		return
	}
	
	err := a.app.UpdateConfig(newConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Configuration updated"})
}

// startCollector handles POST /api/v1/start
// @Summary      Start collector
// @Description  Start the observability collector
// @Tags         control
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /start [post]
func (a *API) startCollector(c *gin.Context) {
	err := a.app.Start(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Collector started"})
}

// stopCollector handles POST /api/v1/stop
// @Summary      Stop collector
// @Description  Stop the observability collector
// @Tags         control
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /stop [post]
func (a *API) stopCollector(c *gin.Context) {
	err := a.app.Stop(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Collector stopped"})
}

// restartCollector handles POST /api/v1/restart
// @Summary      Restart collector
// @Description  Restart the observability collector
// @Tags         control
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /restart [post]
func (a *API) restartCollector(c *gin.Context) {
	err := a.app.Stop(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop collector: " + err.Error()})
		return
	}
	
	err = a.app.Start(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start collector: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Collector restarted"})
}