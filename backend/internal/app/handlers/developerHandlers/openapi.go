package developerHandlers

import (
	"net/http"

	response "github.com/J0es1ick/shortli/internal/app/httputils"
)

func (h *Handler) OpenAPI(w http.ResponseWriter, r *http.Request) {
	shortCode := map[string]interface{}{
		"name": "shortCode", "in": "path", "required": true,
		"schema": map[string]interface{}{"type": "string", "minLength": 3, "maxLength": 32},
	}
	errorResponse := map[string]interface{}{
		"description": "Ошибка запроса",
		"content": map[string]interface{}{"application/json": map[string]interface{}{
			"schema": map[string]string{"$ref": "#/components/schemas/Error"},
		}},
	}
	linkResponse := map[string]interface{}{
		"description": "Ссылка",
		"content": map[string]interface{}{"application/json": map[string]interface{}{
			"schema": map[string]string{"$ref": "#/components/schemas/Link"},
		}},
	}
	pathResponses := func(success map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"200": success, "400": errorResponse, "401": errorResponse,
			"403": errorResponse, "404": errorResponse, "429": errorResponse,
			"500": errorResponse,
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]string{
			"title": "Публичный API Shortli", "version": "1.1.0",
			"description": "Создание и управление короткими ссылками с неизменяемым адресом назначения.",
		},
		"servers":  []map[string]string{{"url": "/api/v1"}},
		"security": []map[string][]string{{"ApiKeyAuth": {}}},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"ApiKeyAuth": map[string]string{"type": "apiKey", "in": "header", "name": "X-API-Key"},
			},
			"schemas": map[string]interface{}{
				"Error": map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{
						"error": map[string]string{"type": "string"},
					},
				},
				"Link": map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{
						"original_url": map[string]string{"type": "string", "format": "uri"},
						"short_code":   map[string]string{"type": "string"},
						"short_url":    map[string]string{"type": "string", "format": "uri"},
						"expires_at":   map[string]interface{}{"type": "string", "format": "date-time", "nullable": true},
						"is_active":    map[string]string{"type": "boolean"},
					},
				},
				"CreateLink": map[string]interface{}{
					"type": "object", "required": []string{"original_url"},
					"properties": map[string]interface{}{
						"original_url": map[string]string{"type": "string", "format": "uri"},
						"custom_alias": map[string]interface{}{"type": "string", "minLength": 3, "maxLength": 32},
						"expires_at":   map[string]interface{}{"type": "string", "format": "date-time", "nullable": true},
					},
				},
				"UpdateLink": map[string]interface{}{
					"type": "object", "properties": map[string]interface{}{
						"is_active":        map[string]string{"type": "boolean"},
						"expires_at":       map[string]interface{}{"type": "string", "format": "date-time", "nullable": true},
						"clear_expiration": map[string]string{"type": "boolean"},
					},
				},
			},
		},
		"paths": map[string]interface{}{
			"/links": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Создать короткую ссылку",
					"parameters": []map[string]interface{}{{
						"name": "include_qr", "in": "query", "required": false,
						"schema": map[string]string{"type": "boolean"},
					}},
					"requestBody": map[string]interface{}{
						"required": true, "content": map[string]interface{}{
							"application/json": map[string]interface{}{"schema": map[string]string{"$ref": "#/components/schemas/CreateLink"}},
						},
					},
					"responses": map[string]interface{}{
						"201": linkResponse, "400": errorResponse, "401": errorResponse,
						"409": errorResponse, "429": errorResponse, "500": errorResponse,
					},
				},
			},
			"/links/{shortCode}": map[string]interface{}{
				"parameters": []map[string]interface{}{shortCode},
				"get": map[string]interface{}{
					"summary": "Получить собственную ссылку", "responses": pathResponses(linkResponse),
				},
				"patch": map[string]interface{}{
					"summary": "Изменить состояние или срок действия",
					"requestBody": map[string]interface{}{
						"required": true, "content": map[string]interface{}{
							"application/json": map[string]interface{}{"schema": map[string]string{"$ref": "#/components/schemas/UpdateLink"}},
						},
					},
					"responses": pathResponses(linkResponse),
				},
				"delete": map[string]interface{}{
					"summary": "Удалить собственную ссылку", "responses": pathResponses(map[string]interface{}{"description": "Ссылка удалена"}),
				},
			},
			"/links/{shortCode}/analytics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Получить аналитику переходов",
					"parameters": []map[string]interface{}{
						shortCode,
						{"name": "days", "in": "query", "schema": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 365, "default": 30}},
					},
					"responses": pathResponses(map[string]interface{}{"description": "Аналитика ссылки"}),
				},
			},
		},
	})
}
