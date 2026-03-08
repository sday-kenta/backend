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
// @Description Если точка находится вне разрешённой области работы проекта, возвращается ошибка.
// @Description Если точка находится внутри разрешённой области, backend пытается определить адрес через локальный кеш и/или Nominatim/OpenStreetMap.
// @Tags maps
// @Accept json
// @Produce json
// @Param lat query number true "Широта точки" example(53.2051714)
// @Param lon query number true "Долгота точки" example(50.1334676)
// @Success 200 {object} response.ReverseGeocodeResponse "Найденный адрес по координатам"
// @Failure 400 {object} response.Error "Некорректные координаты или отсутствующие query-параметры"
// @Failure 422 {object} response.Error "Точка находится вне зоны работы проекта"
// @Failure 500 {object} response.Error "Внутренняя ошибка сервиса карт"
// @Router /v1/maps/reverse [get]
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
// @Description Поиск ограничивается адресами, релевантными для зоны работы проекта. Если найдены только адреса вне разрешённой области, возвращается ошибка.
// @Description Если по запросу не найдено ни одного подходящего адреса, возвращается успешный ответ с пустым списком.
// @Description
// @Description Примеры запросов:
// @Description - "Самара проспект Ленина 1"
// @Description - "Самара Ленина 1"
// @Description - "Ленина 1"
// @Tags maps
// @Accept json
// @Produce json
// @Param q query string true "Строка поиска адреса" example("Самара проспект Ленина 1")
// @Success 200 {object} response.SearchAddressResponse "Список найденных адресов или пустой список"
// @Failure 400 {object} response.Error "Некорректный запрос, например пустой или слишком короткий q"
// @Failure 422 {object} response.Error "Найдены только адреса вне зоны работы проекта"
// @Failure 500 {object} response.Error "Внутренняя ошибка сервиса карт"
// @Router /v1/maps/search [get]
func (r *V1) searchAddresses(ctx *fiber.Ctx) error {
	query := strings.TrimSpace(ctx.Query("q"))
	addresses, err := r.g.Search(ctx.UserContext(), query)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidSearchQuery):
			return errorResponse(ctx, http.StatusBadRequest, err.Error())
		case errors.Is(err, entity.ErrOutOfAllowedZone):
			return errorResponse(ctx, http.StatusUnprocessableEntity, err.Error())
		default:
			r.l.Error(err, "restapi - v1 - searchAddresses")
			return errorResponse(ctx, http.StatusInternalServerError, "map service problems")
		}
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": addresses})
}
