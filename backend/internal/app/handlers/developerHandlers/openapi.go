package developerHandlers

import (
	response "github.com/J0es1ick/shortli/internal/app/httputils"
	"net/http"
)

func (h *Handler) OpenAPI(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"openapi":    "3.0.3",
		"info":       map[string]string{"title": "Публичный API Shortli", "version": "1.0.0", "description": "Создание коротких ссылок с неизменяемым адресом назначения и управление ими."},
		"servers":    []map[string]string{{"url": "/api/v1"}},
		"security":   []map[string][]string{{"ApiKeyAuth": {}}},
		"components": map[string]interface{}{"securitySchemes": map[string]interface{}{"ApiKeyAuth": map[string]string{"type": "apiKey", "in": "header", "name": "X-API-Key"}}},
		"paths": map[string]interface{}{
			"/links": map[string]interface{}{"post": map[string]interface{}{"summary": "Создать короткую ссылку", "responses": map[string]interface{}{"201": map[string]string{"description": "Ссылка создана"}}}},
			"/links/{shortCode}": map[string]interface{}{
				"get":    map[string]string{"summary": "Получить собственную ссылку"},
				"patch":  map[string]string{"summary": "Приостановить, возобновить или изменить срок действия"},
				"delete": map[string]string{"summary": "Удалить собственную ссылку"},
			},
			"/links/{shortCode}/analytics": map[string]interface{}{"get": map[string]string{"summary": "Получить аналитику переходов, доступную владельцу"}},
		},
	})
}
