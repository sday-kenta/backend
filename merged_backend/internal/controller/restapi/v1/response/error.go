// backend/internal/controller/restapi/v1/response/error.go

package response

type Error struct {
	Error string `json:"error" example:"some informative error message"`
}
