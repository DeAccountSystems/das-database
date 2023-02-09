package handle

import (
	"das_database/http_server/api_code"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqVersion struct {
}

type RespVersion struct {
}

func (h *HttpHandle) JsonRpcVersion(p json.RawMessage, apiResp *api_code.ApiResp) {
	var req []ReqVersion
	err := json.Unmarshal(p, &req)
	if err != nil {
		log.Error("json.Unmarshal err:", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}
	if len(req) != 1 {
		log.Error("len(req) is :", len(req))
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}

	if err = h.doVersion(&req[0], apiResp); err != nil {
		log.Error("doVersion err:", err.Error())
	}
}

func (h *HttpHandle) Version(ctx *gin.Context) {
	var (
		funcName = "Version"
		req      ReqVersion
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, toolib.JsonString(req))

	if err = h.doVersion(&req, &apiResp); err != nil {
		log.Error("doVersion err:", err.Error(), funcName)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doVersion(req *ReqVersion, apiResp *api_code.ApiResp) error {
	var resp RespVersion

	apiResp.ApiRespOK(resp)
	return nil
}
