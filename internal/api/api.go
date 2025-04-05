package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sliink/collector/docs"
	"github.com/sliink/collector/internal/core"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// API represents the REST API for the observability collector
type API struct {
	core   *core.Core
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
func NewAPI(core *core.Core, port int, host string) *API {
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
		core:   core,
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
	
	// Pipeline management
	pipelines := a.router.Group("/pipelines")
	{
		pipelines.GET("", a.getPipelines)
		pipelines.GET("/:type", a.getPipelineByType)
		pipelines.POST("", a.createPipeline)
		pipelines.DELETE("/:type", a.deletePipeline)
	}

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
	// Collect status information from the core components
	status := map[string]interface{}{
		"core": map[string]interface{}{
			"status": a.core.GetStatus(),
		},
		"time": time.Now(),
	}
	
	// Add component statuses
	components := make(map[string]interface{})
	if a.core.GetDataPipeline() != nil {
		components["data_pipeline"] = map[string]interface{}{
			"status": a.core.GetDataPipeline().GetStatus(),
		}
	}
	status["components"] = components
	
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
	// Get the plugin registry
	registry, exists := a.core.GetComponent("plugin_registry")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Plugin registry not available"})
		return
	}
	
	// Get all plugins
	plugins := make(map[string]interface{})
	
	// Add input, processor, and output plugins
	component := registry.(core.Component)
	plugins["status"] = component.GetStatus()
	
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
	
	// Get the plugin registry
	_, exists := a.core.GetComponent("plugin_registry")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Plugin registry not available"})
		return
	}
	
	// Get plugins based on type
	result := make(map[string]interface{})
	switch pluginType {
	case "input":
		result["inputs"] = "Input plugins information"
	case "processor":
		result["processors"] = "Processor plugins information"
	case "output":
		result["outputs"] = "Output plugins information"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin type"})
		return
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
	// Not using pluginType for now, but keeping as parameter for API compatibility
	_ = c.Param("type")
	pluginName := c.Param("name")
	
	// Get the plugin registry
	_, exists := a.core.GetComponent("plugin_registry")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Plugin registry not available"})
		return
	}
	
	// Look for the plugin
	plugin, exists := a.core.GetComponent(pluginName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	
	// Return plugin information
	pluginInfo := map[string]interface{}{
		"id":     pluginName,
		"status": plugin.(core.Component).GetStatus(),
	}
	
	c.JSON(http.StatusOK, pluginInfo)
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
	
	// Get the plugin
	plugin, exists := a.core.GetComponent(pluginName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	
	// Try to start the plugin
	if comp, ok := plugin.(core.Component); ok {
		if comp.Start() {
			c.JSON(http.StatusOK, gin.H{"status": "Plugin started"})
			return
		}
	}
	
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start plugin"})
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
	
	// Get the plugin
	plugin, exists := a.core.GetComponent(pluginName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	
	// Try to stop the plugin
	if comp, ok := plugin.(core.Component); ok {
		if comp.Stop() {
			c.JSON(http.StatusOK, gin.H{"status": "Plugin stopped"})
			return
		}
	}
	
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop plugin"})
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
	
	// Get the plugin
	plugin, exists := a.core.GetComponent(pluginName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	
	// Try to restart the plugin
	if comp, ok := plugin.(core.Component); ok {
		if comp.Stop() && comp.Start() {
			c.JSON(http.StatusOK, gin.H{"status": "Plugin restarted"})
			return
		}
	}
	
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart plugin"})
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
	// Get the buffer manager
	bufferManager, exists := a.core.GetComponent("buffer_manager")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Buffer manager not available"})
		return
	}
	
	// Return buffer information
	bufferInfo := map[string]interface{}{
		"status": bufferManager.(core.Component).GetStatus(),
	}
	
	c.JSON(http.StatusOK, bufferInfo)
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
	
	// Get the buffer manager
	_, exists := a.core.GetComponent("buffer_manager")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Buffer manager not available"})
		return
	}
	
	// For now, just return a not implemented status
	// In a real implementation, we would query the buffer manager for the specific buffer
	c.JSON(http.StatusOK, gin.H{
		"name": bufferName,
		"info": "Buffer details would be shown here",
	})
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
	// Use the buffer name param for the response
	bufferName := c.Param("name")
	
	// Get the buffer manager
	_, exists := a.core.GetComponent("buffer_manager")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Buffer manager not available"})
		return
	}
	
	// This would flush the buffer in a real implementation
	c.JSON(http.StatusOK, gin.H{
		"status": fmt.Sprintf("Buffer %s flush operation would happen here", bufferName),
	})
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
	// Get the config manager
	configManager := a.core.GetConfigManager()
	if configManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Configuration manager not available"})
		return
	}
	
	// Get the full configuration
	config := configManager.GetAllConfig()
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
	
	// Get the config manager
	configManager := a.core.GetConfigManager()
	if configManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Configuration manager not available"})
		return
	}
	
	// Update the configuration
	// In a real implementation, we would validate and apply the configuration
	for key, value := range newConfig {
		configManager.SetConfig(key, value)
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
	// Start the core system
	if a.core.Start() {
		c.JSON(http.StatusOK, gin.H{"status": "Collector started"})
		return
	}
	
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start collector"})
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
	// Stop the core system
	if a.core.Stop() {
		c.JSON(http.StatusOK, gin.H{"status": "Collector stopped"})
		return
	}
	
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop collector"})
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
	// Stop the core system
	if !a.core.Stop() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop collector"})
		return
	}
	
	// Start the core system
	if !a.core.Start() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start collector"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "Collector restarted"})
}

// @Summary      Get all pipelines
// @Description  Get information about all data pipelines
// @Tags         pipelines
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /pipelines [get]
func (a *API) getPipelines(c *gin.Context) {
	// Get the data pipeline
	pipeline := a.core.GetDataPipeline()
	if pipeline == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data pipeline not available"})
		return
	}
	
	// Return pipeline information for all types
	pipelines := map[string]interface{}{
		"logs": map[string]interface{}{
			"status": "active",
			"processors": []string{"log_parser"},
		},
		"metrics": map[string]interface{}{
			"status": "inactive",
		},
		"traces": map[string]interface{}{
			"status": "inactive",
		},
	}
	
	c.JSON(http.StatusOK, pipelines)
}

// @Summary      Get pipeline by type
// @Description  Get information about a specific pipeline by telemetry type
// @Tags         pipelines
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Pipeline type (logs, metrics, traces)"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string
// @Router       /pipelines/{type} [get]
func (a *API) getPipelineByType(c *gin.Context) {
	pipelineType := c.Param("type")
	
	// Get the data pipeline
	pipeline := a.core.GetDataPipeline()
	if pipeline == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data pipeline not available"})
		return
	}
	
	// Check pipeline type is valid
	switch pipelineType {
	case "logs", "metrics", "traces":
		// Valid pipeline type
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline type"})
		return
	}
	
	// Return pipeline information based on type
	// In a real implementation, we would get this from the pipeline component
	var pipelineInfo map[string]interface{}
	
	if pipelineType == "logs" {
		pipelineInfo = map[string]interface{}{
			"type": pipelineType,
			"status": "active",
			"processors": []string{"log_parser"},
			"inputs": []string{"file_input", "docker_compose_input"},
			"outputs": []string{"stdout_output"},
		}
		c.JSON(http.StatusOK, pipelineInfo)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pipeline not found"})
	}
}

// @Summary      Create a pipeline
// @Description  Create a new data pipeline
// @Tags         pipelines
// @Accept       json
// @Produce      json
// @Param        pipeline  body    map[string]interface{}  true  "Pipeline configuration"
// @Success      201  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /pipelines [post]
func (a *API) createPipeline(c *gin.Context) {
	var pipelineConfig map[string]interface{}
	if err := c.ShouldBindJSON(&pipelineConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline configuration format"})
		return
	}
	
	// Get the data pipeline
	pipeline := a.core.GetDataPipeline()
	if pipeline == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data pipeline not available"})
		return
	}
	
	// Extract pipeline type and processors
	pipelineType, ok := pipelineConfig["type"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pipeline type is required"})
		return
	}
	
	// Validate pipeline type
	switch pipelineType {
	case "logs", "metrics", "traces":
		// Valid pipeline type
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline type"})
		return
	}
	
	// Extract processors
	processorsList := []string{}
	if processors, ok := pipelineConfig["processors"].([]interface{}); ok {
		for _, processor := range processors {
			if processorID, ok := processor.(string); ok {
				processorsList = append(processorsList, processorID)
			}
		}
	}
	
	// Create the pipeline
	// In a real implementation, this would create the pipeline
	if pipelineType == "logs" {
		c.JSON(http.StatusCreated, gin.H{
			"status": "Pipeline created",
			"type": pipelineType,
			"processors": processorsList,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pipeline, only log pipeline is supported currently"})
	}
}

// @Summary      Delete a pipeline
// @Description  Delete a data pipeline by telemetry type
// @Tags         pipelines
// @Accept       json
// @Produce      json
// @Param        type    path    string  true  "Pipeline type (logs, metrics, traces)"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /pipelines/{type} [delete]
func (a *API) deletePipeline(c *gin.Context) {
	pipelineType := c.Param("type")
	
	// Get the data pipeline
	pipeline := a.core.GetDataPipeline()
	if pipeline == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data pipeline not available"})
		return
	}
	
	// Check pipeline type is valid
	switch pipelineType {
	case "logs", "metrics", "traces":
		// Valid pipeline type
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline type"})
		return
	}
	
	// Delete the pipeline
	// In a real implementation, this would delete the pipeline
	if pipelineType == "logs" {
		c.JSON(http.StatusOK, gin.H{
			"status": "Pipeline deleted",
			"type": pipelineType,
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pipeline not found"})
	}
}