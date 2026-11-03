package deployments

import "github.com/ude-p/shipl/internal/model"

type CreateRequest struct {
	Application string            `json:"application"`
	Environment string            `json:"environment"`
	Spec        model.Application `json:"spec"`
}
