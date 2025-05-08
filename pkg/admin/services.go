package admin

import (
	"fmt"
	"net/http"
	"odin/pkg/config"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *AdminHandler) handleListServices(c echo.Context) error {
	if len(h.config.Services) == 0 {
		return c.HTML(http.StatusOK, `<div class="alert alert-info">No services configured. Add a new service to get started.</div>`)
	}

	html := `<table class="table table-striped mt-4">
		<thead>
			<tr>
				<th>Name</th>
				<th>Base Path</th>
				<th>Targets</th>
				<th>Authentication</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
	`

	for _, svc := range h.config.Services {
		targets := svc.Targets[0]
		if len(svc.Targets) > 1 {
			targets += " +" + fmt.Sprint(len(svc.Targets)-1)
		}

		authStatus := "Required"
		if !svc.Authentication {
			authStatus = "Not Required"
		}

		html += fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>
					<a href="/admin/services/%s" class="btn btn-sm btn-outline-primary">Edit</a>
					<button class="btn btn-sm btn-outline-danger" 
							hx-delete="/admin/services/%s" 
							hx-confirm="Are you sure you want to delete this service?"
							hx-target="#service-list"
							hx-swap="innerHTML">Delete</button>
				</td>
			</tr>
		`, svc.Name, svc.BasePath, targets, authStatus, svc.Name, svc.Name)
	}

	html += `
		</tbody>
	</table>
	`

	return c.HTML(http.StatusOK, html)
}

func (h *AdminHandler) handleNewService(c echo.Context) error {
	data := map[string]interface{}{
		"AvailableServices": h.config.Services,
	}
	return h.renderTemplate(c, "add_service.html", data)
}

func (h *AdminHandler) handleEditService(c echo.Context) error {
	name := c.Param("name")

	var svc *config.ServiceConfig
	for _, s := range h.config.Services {
		if s.Name == name {
			svc = &s
			break
		}
	}

	if svc == nil {
		return c.HTML(http.StatusNotFound, `<div class="alert alert-danger">Service not found</div>`)
	}

	availableServices := make([]config.ServiceConfig, 0)
	for _, s := range h.config.Services {
		if s.Name != name {
			availableServices = append(availableServices, s)
		}
	}

	data := map[string]interface{}{
		"Service":           svc,
		"AvailableServices": availableServices,
		"StripBasePathYes":  svc.StripBasePath,
		"StripBasePathNo":   !svc.StripBasePath,
		"AuthYes":           svc.Authentication,
		"AuthNo":            !svc.Authentication,
		"LoadBalancingRR":   svc.LoadBalancing == "round-robin" || svc.LoadBalancing == "",
		"LoadBalancingRand": svc.LoadBalancing == "random",
	}

	return h.renderTemplate(c, "edit_service.html", data)
}

func (h *AdminHandler) handleAddService(c echo.Context) error {
	name := c.FormValue("name")
	basePath := c.FormValue("basePath")

	for _, svc := range h.config.Services {
		if svc.Name == name {
			return c.HTML(http.StatusBadRequest, `<div class="alert alert-danger">A service with this name already exists</div>`)
		}
	}

	targets := parseMultilineInput(c.FormValue("targets"))
	if len(targets) == 0 {
		return c.HTML(http.StatusBadRequest, `<div class="alert alert-danger">At least one target URL is required</div>`)
	}

	enableAggregation := c.FormValue("enableAggregation") == "on"
	aggregationConfig := parseAggregationConfig(c, enableAggregation)

	newSvc := config.ServiceConfig{
		Name:           name,
		BasePath:       basePath,
		Targets:        targets,
		StripBasePath:  c.FormValue("stripBasePath") == "true",
		Authentication: c.FormValue("authentication") == "true",
		LoadBalancing:  c.FormValue("loadBalancing"),
		Aggregation:    aggregationConfig,
		RetryCount: 1,
		RetryDelay: 100 * time.Millisecond,
		Timeout:    5 * time.Second,
	}

	h.config.Services = append(h.config.Services, newSvc)

	if err := h.saveConfig(); err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="alert alert-danger">Failed to save configuration: `+err.Error()+`</div>`)
	}

	return c.HTML(http.StatusOK, `
		<div class="alert alert-success">
			Service added successfully! 
			<a href="/admin/dashboard">Return to dashboard</a>
		</div>
	`)
}

func (h *AdminHandler) handleUpdateService(c echo.Context) error {
	name := c.Param("name")

	var svcIndex = -1
	for i, svc := range h.config.Services {
		if svc.Name == name {
			svcIndex = i
			break
		}
	}

	if svcIndex == -1 {
		return c.HTML(http.StatusNotFound, `<div class="alert alert-danger">Service not found</div>`)
	}

	targets := parseMultilineInput(c.FormValue("targets"))
	if len(targets) == 0 {
		return c.HTML(http.StatusBadRequest, `<div class="alert alert-danger">At least one target URL is required</div>`)
	}

	h.config.Services[svcIndex].BasePath = c.FormValue("basePath")
	h.config.Services[svcIndex].Targets = targets
	h.config.Services[svcIndex].StripBasePath = c.FormValue("stripBasePath") == "true"
	h.config.Services[svcIndex].Authentication = c.FormValue("authentication") == "true"
	h.config.Services[svcIndex].LoadBalancing = c.FormValue("loadBalancing")

	enableAggregation := c.FormValue("enableAggregation") == "on"
	if enableAggregation {
		aggregationConfig := parseAggregationConfig(c, enableAggregation)
		h.config.Services[svcIndex].Aggregation = aggregationConfig
	} else {
		h.config.Services[svcIndex].Aggregation = nil
	}

	if err := h.saveConfig(); err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="alert alert-danger">Failed to save configuration: `+err.Error()+`</div>`)
	}

	return c.HTML(http.StatusOK, `<div class="alert alert-success">Service updated successfully!</div>`)
}

func (h *AdminHandler) handleDeleteService(c echo.Context) error {
	name := c.Param("name")

	var found bool
	for i, svc := range h.config.Services {
		if svc.Name == name {
			h.config.Services = append(h.config.Services[:i], h.config.Services[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return c.HTML(http.StatusNotFound, `<div class="alert alert-danger">Service not found</div>`)
	}

	if err := h.saveConfig(); err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="alert alert-danger">Failed to save configuration: `+err.Error()+`</div>`)
	}

	return h.handleListServices(c)
}


func parseMultilineInput(input string) []string {
	result := []string{}
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func parseAggregationConfig(c echo.Context, enabled bool) *config.AggregationConfig {
	if !enabled {
		return nil
	}

	aggregationConfig := &config.AggregationConfig{
		Dependencies: []config.DependencyConfig{},
	}

	depCount := 0
	for i := 0; ; i++ {
		depServiceName := c.FormValue(fmt.Sprintf("dependencies[%d].service", i))
		if depServiceName == "" {
			break
		}
		depCount++
	}

	dependencies := make([]config.DependencyConfig, depCount)
	for i := 0; i < depCount; i++ {
		depServiceName := c.FormValue(fmt.Sprintf("dependencies[%d].service", i))
		depPath := c.FormValue(fmt.Sprintf("dependencies[%d].path", i))

		dependency := config.DependencyConfig{
			Service:          depServiceName,
			Path:             depPath,
			ParameterMapping: []config.MappingConfig{},
			ResultMapping:    []config.MappingConfig{},
		}

		for j := 0; ; j++ {
			fromPath := c.FormValue(fmt.Sprintf("dependencies[%d].paramMapping[%d].from", i, j))
			toParam := c.FormValue(fmt.Sprintf("dependencies[%d].paramMapping[%d].to", i, j))

			if fromPath == "" || toParam == "" {
				break
			}

			dependency.ParameterMapping = append(dependency.ParameterMapping, config.MappingConfig{
				From: fromPath,
				To:   toParam,
			})
		}

		for j := 0; ; j++ {
			fromPath := c.FormValue(fmt.Sprintf("dependencies[%d].resultMapping[%d].from", i, j))
			toPath := c.FormValue(fmt.Sprintf("dependencies[%d].resultMapping[%d].to", i, j))

			if fromPath == "" || toPath == "" {
				break
			}

			dependency.ResultMapping = append(dependency.ResultMapping, config.MappingConfig{
				From: fromPath,
				To:   toPath,
			})
		}

		dependencies[i] = dependency
	}

	aggregationConfig.Dependencies = dependencies
	return aggregationConfig
}
