// Package radius 提供 RADIUS 认证计费管理端点的 HTTP 处理。
package radius

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	dtorequest "netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd"
	radiussvc "netlab-backend/internal/service/radius"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// Handler 处理 RADIUS 管理相关端点的 HTTP 请求。
type Handler struct {
	svc *radiussvc.Service
}

// NewHandler 创建一个新的 RADIUS 管理 Handler。
func NewHandler(svc *radiussvc.Service) *Handler {
	return &Handler{svc: svc}
}

// bindJSON 解析请求体，失败时统一返回参数错误。
func bindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid request parameters: "+err.Error()))
		return false
	}
	return true
}

// parseID 解析路径参数 ID。
func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid id"))
		return 0, false
	}
	return id, true
}

// pageParams 解析分页参数。
func pageParams(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
	return page, size
}

// —— 认证用户 ——

// ListUsers 处理 GET /api/radius/users
// @Summary 获取 RADIUS 认证用户列表
// @Tags    Radius
// @Security BearerAuth
// @Param   page    query  int     false "页码"   default(1)
// @Param   size    query  int     false "每页数量" default(20)
// @Param   keyword query  string  false "搜索关键词（用户名/姓名/手机）"
// @Param   status  query  string  false "状态（enabled/disabled）"
// @Success 200     {object} response.ApiResponse{data=dtoresponse.RadiusUserListResult}
// @Router  /api/radius/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	page, size := pageParams(c)
	users, total, appErr := h.svc.ListUsers(c.Request.Context(), page, size, c.Query("keyword"), c.Query("status"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	items := make([]dtoresponse.RadiusUserItem, len(users))
	for i, u := range users {
		items[i] = userToDTO(&u)
	}
	response.SuccessOK(c, dtoresponse.RadiusUserListResult{Items: items, Total: total, Page: page, Size: size})
}

// CreateUser 处理 POST /api/radius/users
// @Summary 创建 RADIUS 认证用户
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusUserUpsertRequest true "用户信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusUserItem}
// @Router  /api/radius/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req dtorequest.RadiusUserUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	user, appErr := h.svc.CreateUser(c.Request.Context(), &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, userToDTO(user))
}

// UpdateUser 处理 PUT /api/radius/users/:id
// @Summary 更新 RADIUS 认证用户
// @Tags    Radius
// @Security BearerAuth
// @Param   id   path uint64 true "用户 ID"
// @Param   body body dtorequest.RadiusUserUpsertRequest true "用户信息（密码留空表示不变）"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusUserItem}
// @Router  /api/radius/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dtorequest.RadiusUserUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	user, appErr := h.svc.UpdateUser(c.Request.Context(), id, &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, userToDTO(user))
}

// DeleteUser 处理 DELETE /api/radius/users/:id
// @Summary 删除 RADIUS 认证用户
// @Tags    Radius
// @Security BearerAuth
// @Param   id path uint64 true "用户 ID"
// @Success 200 {object} response.ApiResponse
// @Router  /api/radius/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if appErr := h.svc.DeleteUser(c.Request.Context(), id); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": 1})
}

func userToDTO(u *model.RadiusUser) dtoresponse.RadiusUserItem {
	item := dtoresponse.RadiusUserItem{
		ID:                      u.ID,
		Username:                u.Username,
		ProfileID:               u.ProfileID,
		ProfileLinkMode:         u.ProfileLinkMode,
		Realname:                u.Realname,
		Email:                   u.Email,
		Mobile:                  u.Mobile,
		Address:                 u.Address,
		MacAddr:                 u.MacAddr,
		Vlanid1:                 u.Vlanid1,
		Vlanid2:                 u.Vlanid2,
		BindMac:                 u.BindMac,
		BindVlan:                u.BindVlan,
		IpAddr:                  u.IpAddr,
		IpV6Addr:                u.IpV6Addr,
		AddrPool:                u.AddrPool,
		IPv6PrefixPool:          u.IPv6PrefixPool,
		DelegatedIpv6Prefix:     u.DelegatedIpv6Prefix,
		DelegatedIpv6PrefixPool: u.DelegatedIpv6PrefixPool,
		ActiveNum:               u.ActiveNum,
		UpRate:                  u.UpRate,
		DownRate:                u.DownRate,
		Domain:                  u.Domain,
		ExpireTime:              u.ExpireTime,
		Status:                  u.Status,
		Remark:                  u.Remark,
		OnlineCount:             u.OnlineCount,
		LastOnline:              u.LastOnline,
		CreatedAt:               u.CreatedAt,
		UpdatedAt:               u.UpdatedAt,
	}
	if u.Profile != nil {
		item.ProfileName = u.Profile.Name
	}
	return item
}

// —— 策略套餐 ——

// ListProfiles 处理 GET /api/radius/profiles
// @Summary 获取 RADIUS 策略套餐列表
// @Tags    Radius
// @Security BearerAuth
// @Param   page    query int    false "页码"   default(1)
// @Param   size    query int    false "每页数量" default(20)
// @Param   keyword query string false "名称关键词"
// @Success 200     {object} response.ApiResponse{data=dtoresponse.RadiusProfileListResult}
// @Router  /api/radius/profiles [get]
func (h *Handler) ListProfiles(c *gin.Context) {
	page, size := pageParams(c)
	profiles, total, appErr := h.svc.ListProfiles(c.Request.Context(), page, size, c.Query("keyword"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	items := make([]dtoresponse.RadiusProfileItem, len(profiles))
	for i, p := range profiles {
		items[i] = profileToDTO(&p)
	}
	response.SuccessOK(c, dtoresponse.RadiusProfileListResult{Items: items, Total: total, Page: page, Size: size})
}

// ListProfileOptions 处理 GET /api/radius/profiles/options
// @Summary 获取全部套餐（下拉选项）
// @Tags    Radius
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse{data=[]dtoresponse.RadiusProfileOption}
// @Router  /api/radius/profiles/options [get]
func (h *Handler) ListProfileOptions(c *gin.Context) {
	profiles, appErr := h.svc.ListAllProfiles(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	items := make([]dtoresponse.RadiusProfileOption, len(profiles))
	for i, p := range profiles {
		items[i] = dtoresponse.RadiusProfileOption{ID: p.ID, Name: p.Name}
	}
	response.SuccessOK(c, items)
}

// CreateProfile 处理 POST /api/radius/profiles
// @Summary 创建 RADIUS 策略套餐
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusProfileUpsertRequest true "套餐信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusProfileItem}
// @Router  /api/radius/profiles [post]
func (h *Handler) CreateProfile(c *gin.Context) {
	var req dtorequest.RadiusProfileUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	profile, appErr := h.svc.CreateProfile(c.Request.Context(), &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, profileToDTO(profile))
}

// UpdateProfile 处理 PUT /api/radius/profiles/:id
// @Summary 更新 RADIUS 策略套餐
// @Tags    Radius
// @Security BearerAuth
// @Param   id   path uint64 true "套餐 ID"
// @Param   body body dtorequest.RadiusProfileUpsertRequest true "套餐信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusProfileItem}
// @Router  /api/radius/profiles/{id} [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dtorequest.RadiusProfileUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	profile, appErr := h.svc.UpdateProfile(c.Request.Context(), id, &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, profileToDTO(profile))
}

// DeleteProfile 处理 DELETE /api/radius/profiles/:id
// @Summary 删除 RADIUS 策略套餐（被引用时禁止）
// @Tags    Radius
// @Security BearerAuth
// @Param   id path uint64 true "套餐 ID"
// @Success 200 {object} response.ApiResponse
// @Router  /api/radius/profiles/{id} [delete]
func (h *Handler) DeleteProfile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if appErr := h.svc.DeleteProfile(c.Request.Context(), id); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": 1})
}

func profileToDTO(p *model.RadiusProfile) dtoresponse.RadiusProfileItem {
	return dtoresponse.RadiusProfileItem{
		ID:                      p.ID,
		Name:                    p.Name,
		Status:                  p.Status,
		AddrPool:                p.AddrPool,
		IPv6PrefixPool:          p.IPv6PrefixPool,
		DelegatedIpv6PrefixPool: p.DelegatedIpv6PrefixPool,
		ActiveNum:               p.ActiveNum,
		UpRate:                  p.UpRate,
		DownRate:                p.DownRate,
		Domain:                  p.Domain,
		BindMac:                 p.BindMac,
		BindVlan:                p.BindVlan,
		Remark:                  p.Remark,
		UserCount:               p.UserCount,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

// —— NAS 设备 ——

// ListNas 处理 GET /api/radius/nas
// @Summary 获取 NAS 设备列表
// @Tags    Radius
// @Security BearerAuth
// @Param   page    query int    false "页码"   default(1)
// @Param   size    query int    false "每页数量" default(20)
// @Param   keyword query string false "关键词（名称/IP/Identifier）"
// @Success 200     {object} response.ApiResponse{data=dtoresponse.RadiusNasListResult}
// @Router  /api/radius/nas [get]
func (h *Handler) ListNas(c *gin.Context) {
	page, size := pageParams(c)
	items, total, appErr := h.svc.ListNas(c.Request.Context(), page, size, c.Query("keyword"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	dtos := make([]dtoresponse.RadiusNasItem, len(items))
	for i, n := range items {
		dtos[i] = nasToDTO(&n)
	}
	response.SuccessOK(c, dtoresponse.RadiusNasListResult{Items: dtos, Total: total, Page: page, Size: size})
}

// CreateNas 处理 POST /api/radius/nas
// @Summary 创建 NAS 设备
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusNasUpsertRequest true "NAS 信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusNasItem}
// @Router  /api/radius/nas [post]
func (h *Handler) CreateNas(c *gin.Context) {
	var req dtorequest.RadiusNasUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	nas, appErr := h.svc.CreateNas(c.Request.Context(), &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, nasToDTO(nas))
}

// UpdateNas 处理 PUT /api/radius/nas/:id
// @Summary 更新 NAS 设备
// @Tags    Radius
// @Security BearerAuth
// @Param   id   path uint64 true "NAS ID"
// @Param   body body dtorequest.RadiusNasUpsertRequest true "NAS 信息（密钥留空表示不变）"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusNasItem}
// @Router  /api/radius/nas/{id} [put]
func (h *Handler) UpdateNas(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dtorequest.RadiusNasUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	nas, appErr := h.svc.UpdateNas(c.Request.Context(), id, &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, nasToDTO(nas))
}

// DeleteNas 处理 DELETE /api/radius/nas/:id
// @Summary 删除 NAS 设备
// @Tags    Radius
// @Security BearerAuth
// @Param   id path uint64 true "NAS ID"
// @Success 200 {object} response.ApiResponse
// @Router  /api/radius/nas/{id} [delete]
func (h *Handler) DeleteNas(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if appErr := h.svc.DeleteNas(c.Request.Context(), id); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": 1})
}

func nasToDTO(n *model.RadiusNas) dtoresponse.RadiusNasItem {
	return dtoresponse.RadiusNasItem{
		ID:         n.ID,
		Name:       n.Name,
		Identifier: n.Identifier,
		Hostname:   n.Hostname,
		Ipaddr:     n.Ipaddr,
		CoaPort:    n.CoaPort,
		Model:      n.Model,
		VendorCode: n.VendorCode,
		Status:     n.Status,
		Tags:       n.Tags,
		Remark:     n.Remark,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
	}
}

// —— 在线会话 ——

// ListSessions 处理 GET /api/radius/sessions
// @Summary 获取在线会话列表
// @Tags    Radius
// @Security BearerAuth
// @Param   page     query int    false "页码"   default(1)
// @Param   size     query int    false "每页数量" default(20)
// @Param   username query string false "用户名"
// @Param   nasAddr  query string false "NAS IP"
// @Param   macAddr  query string false "MAC 地址"
// @Success 200      {object} response.ApiResponse{data=dtoresponse.RadiusSessionListResult}
// @Router  /api/radius/sessions [get]
func (h *Handler) ListSessions(c *gin.Context) {
	page, size := pageParams(c)
	items, total, appErr := h.svc.ListSessions(c.Request.Context(), page, size,
		c.Query("username"), c.Query("nasAddr"), c.Query("macAddr"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	dtos := make([]dtoresponse.RadiusSessionItem, len(items))
	for i, s := range items {
		dtos[i] = sessionToDTO(&s)
	}
	response.SuccessOK(c, dtoresponse.RadiusSessionListResult{Items: dtos, Total: total, Page: page, Size: size})
}

// KickSession 处理 DELETE /api/radius/sessions/:id
// @Summary 踢下线（向 NAS 发送 Disconnect-Request）
// @Tags    Radius
// @Security BearerAuth
// @Param   id path uint64 true "会话 ID"
// @Success 200 {object} response.ApiResponse{data=dtoresponse.RadiusKickResult}
// @Router  /api/radius/sessions/{id} [delete]
func (h *Handler) KickSession(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result, appErr := h.svc.KickSession(c.Request.Context(), id)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, kickResultToDTO(result))
}

// kickResultToDTO 把 CoA/Disconnect 交换结果转换为响应视图。
func kickResultToDTO(result *radiusd.CoAResult) dtoresponse.RadiusKickResult {
	return dtoresponse.RadiusKickResult{
		Success:        result.Success,
		ResponseCode:   result.ResponseCode,
		Target:         result.Target,
		Message:        result.Err,
		ErrorCause:     result.ErrorCause,
		ErrorCauseText: result.ErrorCauseText,
		RttMs:          result.RTT.Milliseconds(),
	}
}

// CoASession 处理 POST /api/radius/sessions/:id/coa
// @Summary 会话 CoA 动态授权变更（下发 Session-Timeout/Filter-Id）
// @Tags    Radius
// @Security BearerAuth
// @Param   id   path uint64 true "会话 ID"
// @Param   body body dtorequest.RadiusCoARequest true "授权变更属性（sessionTimeout 与 filterId 至少一项）"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusKickResult}
// @Router  /api/radius/sessions/{id}/coa [post]
func (h *Handler) CoASession(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dtorequest.RadiusCoARequest
	if !bindJSON(c, &req) {
		return
	}
	if (req.SessionTimeout == nil || *req.SessionTimeout <= 0) && req.FilterID == "" {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "sessionTimeout 与 filterId 至少一项有效"))
		return
	}
	result, appErr := h.svc.ModifySession(c.Request.Context(), id, req.SessionTimeout, req.FilterID)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, kickResultToDTO(result))
}

func sessionToDTO(s *model.RadiusOnline) dtoresponse.RadiusSessionItem {
	return dtoresponse.RadiusSessionItem{
		ID:                  s.ID,
		Username:            s.Username,
		NasId:               s.NasId,
		NasAddr:             s.NasAddr,
		NasPaddr:            s.NasPaddr,
		NasClass:            s.NasClass,
		NasPort:             s.NasPort,
		NasPortId:           s.NasPortId,
		NasPortType:         s.NasPortType,
		ServiceType:         s.ServiceType,
		FramedIpaddr:        s.FramedIpaddr,
		FramedNetmask:       s.FramedNetmask,
		FramedIpv6Prefix:    s.FramedIpv6Prefix,
		FramedIpv6Address:   s.FramedIpv6Address,
		DelegatedIpv6Prefix: s.DelegatedIpv6Prefix,
		MacAddr:             s.MacAddr,
		AcctSessionId:       s.AcctSessionId,
		AcctSessionTime:     s.AcctSessionTime,
		AcctInputTotal:      s.AcctInputTotal,
		AcctOutputTotal:     s.AcctOutputTotal,
		AcctInputPackets:    s.AcctInputPackets,
		AcctOutputPackets:   s.AcctOutputPackets,
		AcctStartTime:       s.AcctStartTime,
		LastUpdate:          s.LastUpdate,
	}
}

// —— 记账记录 ——

// ListAccounting 处理 GET /api/radius/accounting
// @Summary 获取记账记录
// @Tags    Radius
// @Security BearerAuth
// @Param   page      query int    false "页码"   default(1)
// @Param   size      query int    false "每页数量" default(20)
// @Param   username  query string false "用户名"
// @Param   startTime query string false "开始时间（RFC3339）"
// @Param   endTime   query string false "结束时间（RFC3339）"
// @Success 200       {object} response.ApiResponse{data=dtoresponse.RadiusAccountingListResult}
// @Router  /api/radius/accounting [get]
func (h *Handler) ListAccounting(c *gin.Context) {
	page, size := pageParams(c)
	startTime := parseTimeQuery(c, "startTime")
	endTime := parseTimeQuery(c, "endTime")
	items, total, appErr := h.svc.ListAccounting(c.Request.Context(), page, size, c.Query("username"), startTime, endTime)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	dtos := make([]dtoresponse.RadiusAccountingItem, len(items))
	for i, a := range items {
		dtos[i] = accountingToDTO(&a)
	}
	response.SuccessOK(c, dtoresponse.RadiusAccountingListResult{Items: dtos, Total: total, Page: page, Size: size})
}

// parseTimeQuery 解析 RFC3339 时间查询参数，无效时返回 nil。
func parseTimeQuery(c *gin.Context, key string) *time.Time {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t
	}
	return nil
}

func accountingToDTO(a *model.RadiusAccounting) dtoresponse.RadiusAccountingItem {
	return dtoresponse.RadiusAccountingItem{
		ID:                  a.ID,
		Username:            a.Username,
		NasId:               a.NasId,
		NasAddr:             a.NasAddr,
		NasPaddr:            a.NasPaddr,
		NasClass:            a.NasClass,
		NasPort:             a.NasPort,
		NasPortId:           a.NasPortId,
		NasPortType:         a.NasPortType,
		ServiceType:         a.ServiceType,
		FramedIpaddr:        a.FramedIpaddr,
		FramedNetmask:       a.FramedNetmask,
		FramedIpv6Prefix:    a.FramedIpv6Prefix,
		FramedIpv6Address:   a.FramedIpv6Address,
		DelegatedIpv6Prefix: a.DelegatedIpv6Prefix,
		MacAddr:             a.MacAddr,
		AcctSessionId:       a.AcctSessionId,
		AcctSessionTime:     a.AcctSessionTime,
		AcctInputTotal:      a.AcctInputTotal,
		AcctOutputTotal:     a.AcctOutputTotal,
		AcctInputPackets:    a.AcctInputPackets,
		AcctOutputPackets:   a.AcctOutputPackets,
		AcctTerminateCause:  a.AcctTerminateCause,
		AcctStartTime:       a.AcctStartTime,
		AcctStopTime:        a.AcctStopTime,
	}
}

// —— 认证日志 ——

// ListAuthLogs 处理 GET /api/radius/auth-logs
// @Summary 获取 RADIUS 认证日志
// @Tags    Radius
// @Security BearerAuth
// @Param   page    query  int    false "页码"   default(1)
// @Param   size    query  int    false "每页数量" default(20)
// @Param   keyword query  string false "关键词（用户名/MAC）"
// @Param   result  query  string false "结果（accept/reject）"
// @Success 200     {object} response.ApiResponse{data=dtoresponse.RadiusAuthLogListResult}
// @Router  /api/radius/auth-logs [get]
func (h *Handler) ListAuthLogs(c *gin.Context) {
	page, size := pageParams(c)
	items, total, appErr := h.svc.ListAuthLogs(c.Request.Context(), page, size, c.Query("keyword"), c.Query("result"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	dtos := make([]dtoresponse.RadiusAuthLogItem, len(items))
	for i, l := range items {
		dtos[i] = authLogToDTO(&l)
	}
	response.SuccessOK(c, dtoresponse.RadiusAuthLogListResult{Items: dtos, Total: total, Page: page, Size: size})
}

// DeleteAuthLogs 处理 DELETE /api/radius/auth-logs
// @Summary 批量删除认证日志
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusIDsRequest true "待删除的日志 ID 列表"
// @Success 200  {object} response.ApiResponse
// @Router  /api/radius/auth-logs [delete]
func (h *Handler) DeleteAuthLogs(c *gin.Context) {
	var req dtorequest.RadiusIDsRequest
	if !bindJSON(c, &req) {
		return
	}
	deleted, appErr := h.svc.DeleteAuthLogs(c.Request.Context(), req.IDs)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": deleted})
}

func authLogToDTO(l *model.RadiusAuthLog) dtoresponse.RadiusAuthLogItem {
	return dtoresponse.RadiusAuthLogItem{
		ID:        l.ID,
		Username:  l.Username,
		NasAddr:   l.NasAddr,
		NasPaddr:  l.NasPaddr,
		MacAddr:   l.MacAddr,
		AuthType:  l.AuthType,
		Result:    l.Result,
		Reason:    l.Reason,
		CreatedAt: l.CreatedAt,
	}
}

// —— 免认证规则 ——

// ListBypassRules 处理 GET /api/radius/bypass
// @Summary 获取免认证规则列表
// @Tags    Radius
// @Security BearerAuth
// @Param   page    query  int     false "页码"   default(1)
// @Param   size    query  int     false "每页数量" default(20)
// @Param   keyword query  string  false "关键词（取值/备注）"
// @Success 200     {object} response.ApiResponse{data=dtoresponse.RadiusBypassListResult}
// @Router  /api/radius/bypass [get]
func (h *Handler) ListBypassRules(c *gin.Context) {
	page, size := pageParams(c)
	items, total, appErr := h.svc.ListBypassRules(c.Request.Context(), page, size, c.Query("keyword"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	dtos := make([]dtoresponse.RadiusBypassItem, len(items))
	for i, r := range items {
		dtos[i] = bypassToDTO(&r)
	}
	response.SuccessOK(c, dtoresponse.RadiusBypassListResult{Items: dtos, Total: total, Page: page, Size: size})
}

// CreateBypassRule 处理 POST /api/radius/bypass
// @Summary 创建免认证规则（type=mac 时 value 为单个 MAC；type=ip 时为 IP/CIDR）
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusBypassUpsertRequest true "规则信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusBypassItem}
// @Router  /api/radius/bypass [post]
func (h *Handler) CreateBypassRule(c *gin.Context) {
	var req dtorequest.RadiusBypassUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	rule, appErr := h.svc.CreateBypassRule(c.Request.Context(), &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, bypassToDTO(rule))
}

// UpdateBypassRule 处理 PUT /api/radius/bypass/:id
// @Summary 更新免认证规则
// @Tags    Radius
// @Security BearerAuth
// @Param   id   path uint64 true "规则 ID"
// @Param   body body dtorequest.RadiusBypassUpsertRequest true "规则信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusBypassItem}
// @Router  /api/radius/bypass/{id} [put]
func (h *Handler) UpdateBypassRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dtorequest.RadiusBypassUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	rule, appErr := h.svc.UpdateBypassRule(c.Request.Context(), id, &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, bypassToDTO(rule))
}

// DeleteBypassRule 处理 DELETE /api/radius/bypass/:id
// @Summary 删除免认证规则
// @Tags    Radius
// @Security BearerAuth
// @Param   id path uint64 true "规则 ID"
// @Success 200 {object} response.ApiResponse
// @Router  /api/radius/bypass/{id} [delete]
func (h *Handler) DeleteBypassRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if appErr := h.svc.DeleteBypassRule(c.Request.Context(), id); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": 1})
}

func bypassToDTO(r *model.RadiusBypass) dtoresponse.RadiusBypassItem {
	return dtoresponse.RadiusBypassItem{
		ID:         r.ID,
		Type:       r.Type,
		Value:      r.Value,
		ProfileID:  r.ProfileID,
		NasID:      r.NasID,
		ExpireTime: r.ExpireTime,
		Status:     r.Status,
		Remark:     r.Remark,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

// —— 系统设置 ——

// GetSettings 处理 GET /api/radius/settings
// @Summary 获取 RADIUS 设置（env 与 DB 配置合并后的生效值）
// @Tags    Radius
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse{data=dtoresponse.RadiusSettingsResponse}
// @Router  /api/radius/settings [get]
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius settings", err))
		return
	}
	response.SuccessOK(c, settings)
}

// UpdateSystemSettings 处理 PUT /api/radius/settings/system
// @Summary 更新 RADIUS 系统（监听器）设置并热生效
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusSystemSettingsRequest true "系统设置"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusSettingsResponse}
// @Router  /api/radius/settings/system [put]
func (h *Handler) UpdateSystemSettings(c *gin.Context) {
	var req dtorequest.RadiusSystemSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	if appErr := h.svc.UpdateSystemSettings(c.Request.Context(), &req); appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.GetSettings(c)
}

// UpdateServerSettings 处理 PUT /api/radius/settings/server
// @Summary 更新 RADIUS 服务器策略设置并热生效
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusServerSettingsRequest true "服务器策略设置"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusSettingsResponse}
// @Router  /api/radius/settings/server [put]
func (h *Handler) UpdateServerSettings(c *gin.Context) {
	var req dtorequest.RadiusServerSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	if appErr := h.svc.UpdateServerSettings(c.Request.Context(), &req); appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.GetSettings(c)
}

// UpdateEapSettings 处理 PUT /api/radius/settings/eap
// @Summary 更新 RADIUS EAP（802.1X）设置并热生效
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusEapSettingsRequest true "EAP 设置"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusSettingsResponse}
// @Router  /api/radius/settings/eap [put]
func (h *Handler) UpdateEapSettings(c *gin.Context) {
	var req dtorequest.RadiusEapSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	if appErr := h.svc.UpdateEapSettings(c.Request.Context(), &req); appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.GetSettings(c)
}

// —— TLS 证书 ——

// ListCerts 处理 GET /api/radius/certs
// @Summary 获取 RADIUS TLS 证书列表
// @Tags    Radius
// @Security BearerAuth
// @Param   page     query int    false "页码"   default(1)
// @Param   size     query int    false "每页数量" default(20)
// @Param   keyword  query string false "关键词（名称/主题）"
// @Param   certType query string false "类型（server/ca）"
// @Success 200      {object} response.ApiResponse{data=dtoresponse.RadiusCertListResult}
// @Router  /api/radius/certs [get]
func (h *Handler) ListCerts(c *gin.Context) {
	page, size := pageParams(c)
	items, total, appErr := h.svc.ListCerts(c.Request.Context(), page, size, c.Query("keyword"), c.Query("certType"))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	dtos := make([]dtoresponse.RadiusCertItem, len(items))
	for i, cert := range items {
		dtos[i] = certToDTO(&cert)
	}
	response.SuccessOK(c, dtoresponse.RadiusCertListResult{Items: dtos, Total: total, Page: page, Size: size})
}

// CreateCert 处理 POST /api/radius/certs
// @Summary 导入 RADIUS TLS 证书（服务器证书必须提供私钥）
// @Tags    Radius
// @Security BearerAuth
// @Param   body body dtorequest.RadiusCertUpsertRequest true "证书信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusCertItem}
// @Router  /api/radius/certs [post]
func (h *Handler) CreateCert(c *gin.Context) {
	var req dtorequest.RadiusCertUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	cert, appErr := h.svc.CreateCert(c.Request.Context(), &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, certToDTO(cert))
}

// UpdateCert 处理 PUT /api/radius/certs/:id
// @Summary 更新 RADIUS TLS 证书（类型不允许修改；私钥留空表示不变）
// @Tags    Radius
// @Security BearerAuth
// @Param   id   path uint64 true "证书 ID"
// @Param   body body dtorequest.RadiusCertUpsertRequest true "证书信息"
// @Success 200  {object} response.ApiResponse{data=dtoresponse.RadiusCertItem}
// @Router  /api/radius/certs/{id} [put]
func (h *Handler) UpdateCert(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dtorequest.RadiusCertUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	cert, appErr := h.svc.UpdateCert(c.Request.Context(), id, &req)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, certToDTO(cert))
}

// DeleteCert 处理 DELETE /api/radius/certs/:id
// @Summary 删除 RADIUS TLS 证书（被设置引用时禁止）
// @Tags    Radius
// @Security BearerAuth
// @Param   id path uint64 true "证书 ID"
// @Success 200 {object} response.ApiResponse
// @Router  /api/radius/certs/{id} [delete]
func (h *Handler) DeleteCert(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if appErr := h.svc.DeleteCert(c.Request.Context(), id); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": 1})
}

// ExportCert 处理 GET /api/radius/certs/:id/export
// @Summary 导出证书 PEM（includeKey=true 时附带私钥）
// @Tags    Radius
// @Security BearerAuth
// @Param   id         path uint64 true "证书 ID"
// @Param   includeKey query bool   false "是否附带私钥"
// @Success 200 {file} file "PEM 文件下载"
// @Router  /api/radius/certs/{id}/export [get]
func (h *Handler) ExportCert(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	includeKey := c.Query("includeKey") == "true"
	filename, content, appErr := h.svc.ExportCert(c.Request.Context(), id, includeKey)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(200, "application/x-pem-file", content)
}

func certToDTO(cert *model.RadiusCert) dtoresponse.RadiusCertItem {
	return dtoresponse.RadiusCertItem{
		ID:          cert.ID,
		Name:        cert.Name,
		CertType:    cert.CertType,
		CertPEM:     cert.CertPEM,
		Subject:     cert.Subject,
		Issuer:      cert.Issuer,
		Serial:      cert.Serial,
		Fingerprint: cert.Fingerprint,
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
		HasKey:      cert.HasKey,
		Remark:      cert.Remark,
		CreatedAt:   cert.CreatedAt,
		UpdatedAt:   cert.UpdatedAt,
	}
}
