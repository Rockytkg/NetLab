package portal

import (
	"strconv"

	"github.com/gin-gonic/gin"
	request "netlab-backend/internal/dto/request"
	service "netlab-backend/internal/service/portal"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

type Handler struct{ svc *service.Service }

func NewHandler(svc *service.Service) *Handler { return &Handler{svc: svc} }
func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return page, size
}
func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid id"))
		return 0, false
	}
	return id, true
}
func input(req request.PortalNasUpsertRequest) service.NasInput {
	return service.NasInput{Name: req.Name, Identifier: req.Identifier, Vendor: req.Vendor, ProtocolProfile: req.ProtocolProfile, SourceIP: req.SourceIP, ACPort: req.ACPort, SharedSecret: req.SharedSecret, RadiusNasID: req.RadiusNasID, CoAEnabled: req.CoAEnabled, Status: req.Status, Remark: req.Remark}
}
func bind(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid request parameters: "+err.Error()))
		return false
	}
	return true
}
func (h *Handler) ListNas(c *gin.Context) {
	page, size := pageParams(c)
	items, total, e := h.svc.ListNas(c, page, size, c.Query("keyword"))
	if e != nil {
		response.Error(c, e)
		return
	}
	response.SuccessOK(c, gin.H{"items": items, "total": total, "page": page, "size": size})
}
func (h *Handler) CreateNas(c *gin.Context) {
	var req request.PortalNasUpsertRequest
	if !bind(c, &req) {
		return
	}
	item, e := h.svc.CreateNas(c, input(req))
	if e != nil {
		response.Error(c, e)
		return
	}
	response.SuccessOK(c, item)
}
func (h *Handler) UpdateNas(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req request.PortalNasUpsertRequest
	if !bind(c, &req) {
		return
	}
	item, e := h.svc.UpdateNas(c, id, input(req))
	if e != nil {
		response.Error(c, e)
		return
	}
	response.SuccessOK(c, item)
}
func (h *Handler) DeleteNas(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if e := h.svc.DeleteNas(c, id); e != nil {
		response.Error(c, e)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": 1})
}
func (h *Handler) ListSessions(c *gin.Context) {
	page, size := pageParams(c)
	items, total, e := h.svc.ListSessions(c, page, size, c.Query("username"), c.Query("nasId"))
	if e != nil {
		response.Error(c, e)
		return
	}
	response.SuccessOK(c, gin.H{"items": items, "total": total, "page": page, "size": size})
}
func (h *Handler) TerminateSession(c *gin.Context) {
	if e := h.svc.TerminateSession(c, c.Param("id")); e != nil {
		response.Error(c, e)
		return
	}
	response.SuccessOK(c, gin.H{"terminated": true})
}
func (h *Handler) Authenticate(c *gin.Context) {
	var req request.PortalAuthenticationRequest
	if !bind(c, &req) {
		return
	}
	authType := byte(0)
	if req.AuthType == "pap" {
		authType = 1
	}
	session, appErr := h.svc.Authenticate(c.Request.Context(), service.AuthenticateInput{NASIdentifier: req.NASIdentifier, ClientIP: req.ClientIP, Username: req.Username, Password: req.Password, AuthType: authType})
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, session)
}

// GetSettings returns the effective Portal listener configuration.
func (h *Handler) GetSettings(c *gin.Context) {
	response.SuccessOK(c, h.svc.EffectiveConfig(c.Request.Context()))
}

// UpdateSettings persists and hot-applies Portal listener settings.
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req request.PortalSystemSettingsRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.UpdateSettings(c.Request.Context(), req.Enabled, req.BindHost, req.NotifyPort); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, h.svc.EffectiveConfig(c.Request.Context()))
}
