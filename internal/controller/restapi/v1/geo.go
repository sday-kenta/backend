package v1

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/gofiber/fiber/v2"
)

// ReverseGeocode godoc
// @Summary Получение адреса по координатам
// @Description Выполняет reverse geocoding: по координатам точки на карте возвращает человекочитаемый адрес.
// @Description Используется, когда пользователь выбирает место на карте, а frontend должен показать адрес выбранной точки.
// @Description
// @Description Сначала backend локально проверяет, попадает ли точка хотя бы в одну поддерживаемую зону проекта в PostGIS.
// @Description Если точка не попадает ни в одну зону, возвращается ошибка. Если попадает, backend определяет адрес через локальный кеш и/или Nominatim/OpenStreetMap.
// @Tags maps
// @Accept json
// @Produce json
// @Param lat query number true "Широта точки" example(53.2051714)
// @Param lon query number true "Долгота точки" example(50.1334676)
// @Success 200 {object} response.ReverseGeocodeResponse "Найденный адрес по координатам"
// @Failure 400 {object} response.Error "Некорректные координаты или отсутствующие query-параметры"
// @Failure 404 {object} response.Error "Адрес по координатам не найден"
// @Failure 422 {object} response.Error "Точка находится вне зоны работы проекта"
// @Failure 500 {object} response.Error "Внутренняя ошибка сервиса карт"
// @Router /maps/reverse [get]
func (r *V1) reverseGeocode(ctx *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(strings.TrimSpace(ctx.Query("lat")), 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid lat query parameter")
	}

	lon, err := strconv.ParseFloat(strings.TrimSpace(ctx.Query("lon")), 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid lon query parameter")
	}

	address, err := r.g.ReverseGeocode(ctx.UserContext(), lat, lon)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCoordinates):
			return errorResponse(ctx, http.StatusBadRequest, err.Error())
		case errors.Is(err, entity.ErrOutOfAllowedZone):
			return errorResponse(ctx, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, entity.ErrAddressNotFound):
			return errorResponse(ctx, http.StatusNotFound, err.Error())
		default:
			r.l.Error(err, "restapi - v1 - reverseGeocode")
			return errorResponse(ctx, http.StatusInternalServerError, "map service problems")
		}
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": address})
}

// SearchAddresses godoc
// @Summary Поиск адреса по строке
// @Description Выполняет поиск адреса по текстовому запросу пользователя через Nominatim/OpenStreetMap. Используется, когда пользователь вводит адрес вручную в строке поиска на карте.
// @Description
// @Description Сервис может вернуть до 4 наиболее подходящих вариантов адреса. Если frontend передаёт query-параметр city, он трактуется как город пользователя и должен входить в список поддерживаемых городов проекта.
// @Description Если в строке запроса уже указан поддерживаемый город, он имеет приоритет над query-параметром city. Если указан неподдерживаемый город или найдены только адреса вне поддерживаемых зон проекта, возвращается ошибка.
// @Description Если по запросу не найдено ни одного подходящего адреса, возвращается ошибка 404.
// @Description
// @Description Примеры запросов:
// @Description - "Самара проспект Ленина 1"
// @Description - "Самара Ленина 1"
// @Description - "Ленина 1"
// @Tags maps
// @Accept json
// @Produce json
// @Param q query string true "Строка поиска адреса" default(Ленина 1) example(Ленина 1)
// @Param city query string false "Город-подсказка пользователя. Для Swagger по умолчанию подставляется samara." default(samara) example(samara)
// @Success 200 {object} response.SearchAddressResponse "До 4 наиболее подходящих адресов"
// @Failure 400 {object} response.Error "Некорректный запрос, например пустой или слишком короткий q"
// @Failure 404 {object} response.Error "Подходящих адресов не найдено"
// @Failure 422 {object} response.Error "Указан неподдерживаемый город или найдены только адреса вне зоны работы проекта"
// @Failure 500 {object} response.Error "Внутренняя ошибка сервиса карт"
// @Router /maps/search [get]
func (r *V1) searchAddresses(ctx *fiber.Ctx) error {
	query := strings.TrimSpace(ctx.Query("q"))
	city := strings.TrimSpace(ctx.Query("city"))
	addresses, err := r.g.Search(ctx.UserContext(), query, city)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidSearchQuery):
			return errorResponse(ctx, http.StatusBadRequest, err.Error())
		case errors.Is(err, entity.ErrAddressNotFound):
			return errorResponse(ctx, http.StatusNotFound, err.Error())
		case errors.Is(err, entity.ErrOutOfAllowedZone):
			return errorResponse(ctx, http.StatusUnprocessableEntity, err.Error())
		default:
			r.l.Error(err, "restapi - v1 - searchAddresses")
			return errorResponse(ctx, http.StatusInternalServerError, "map service problems")
		}
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": addresses})
}

// ReloadCities godoc
// @Summary Reload supported cities cache
// @Description Reloads in-memory supported cities from zones table. Admin only.
// @Tags maps
// @Accept json
// @Produce json
// @Param X-User-Role header string true "User role (must be 'admin')" default(admin)
// @Success 200 {object} map[string]string "Reload successful"
// @Failure 403 {object} response.Error "Access denied"
// @Failure 500 {object} response.Error "Internal error"
// @Router /maps/reload-cities [post]
func (r *V1) reloadCities(ctx *fiber.Ctx) error {
	if err := r.g.ReloadCities(ctx.UserContext()); err != nil {
		r.l.Error(err, "restapi - v1 - reloadCities")
		return errorResponse(ctx, http.StatusInternalServerError, "map service problems")
	}
	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "message": "cities cache reloaded"})
}
