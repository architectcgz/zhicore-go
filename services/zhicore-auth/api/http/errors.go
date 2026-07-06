package httpapi

import (
	"errors"
	"net/http"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
)

var errInvalidRequest = errors.New("invalid request")

func writeValidationError(w http.ResponseWriter) {
	sharedhttp.WriteErrorCode(w, http.StatusBadRequest, 1001, "参数校验失败")
}

func writeAccepted(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusAccepted, sharedhttp.CodeSuccess, sharedhttp.MessageSuccess, data)
}

func writeMappedError(w http.ResponseWriter, err error) {
	status, code, message := errorMapping(err)
	sharedhttp.WriteErrorCode(w, status, code, message)
}

func errorMapping(err error) (int, int, string) {
	switch {
	case errors.Is(err, ErrEmailInvalid):
		return http.StatusBadRequest, 2010, "邮箱格式错误"
	case errors.Is(err, ErrPasswordInvalid):
		return http.StatusBadRequest, 2011, "密码不符合要求"
	case errors.Is(err, ErrEmailExists):
		return http.StatusConflict, 2009, "邮箱已被注册"
	case errors.Is(err, ErrRegisterPendingRetryable):
		return http.StatusServiceUnavailable, 2012, "注册暂时未完成，请稍后重试"
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, 2003, "登录失败"
	case errors.Is(err, ErrAccountDisabled):
		return http.StatusForbidden, 2004, "账号已禁用"
	case errors.Is(err, ErrAccountBanned):
		return http.StatusForbidden, 2019, "账号已被封禁"
	case errors.Is(err, ErrAccountLocked):
		return http.StatusForbidden, 2014, "账号已临时锁定"
	case errors.Is(err, ErrCSRFInvalid):
		return http.StatusForbidden, 2013, "CSRF 校验失败"
	case errors.Is(err, ErrTokenInvalid):
		return http.StatusUnauthorized, 2001, "Token无效"
	case errors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized, 2002, "Token已过期"
	case errors.Is(err, ErrTokenReplayed):
		return http.StatusUnauthorized, 2017, "登录凭证已被重复使用"
	case errors.Is(err, ErrSessionRevoked):
		return http.StatusUnauthorized, 2018, "会话已失效"
	case errors.Is(err, ErrLoginRequired):
		return http.StatusUnauthorized, 2006, "请先登录"
	case errors.Is(err, ErrPermissionDenied):
		return http.StatusForbidden, 2005, "权限不足"
	case errors.Is(err, ErrRoleRequired):
		return http.StatusForbidden, 2007, "需要特定角色"
	case errors.Is(err, ErrResourceAccessDenied):
		return http.StatusForbidden, 2008, "无权访问该资源"
	case errors.Is(err, ErrDataNotFound):
		return http.StatusNotFound, 1005, "数据不存在"
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests, 2015, "请求过于频繁"
	// Principal unavailable is an Auth-specific contract and must not collapse
	// into the generic degraded code used by other dependency failures.
	case errors.Is(err, ErrPrincipalUnavailable):
		return http.StatusServiceUnavailable, 2016, "登录状态暂时无法确认"
	case errors.Is(err, ErrServiceDegraded):
		return http.StatusServiceUnavailable, 1004, "服务暂时不可用"
	default:
		return http.StatusInternalServerError, 1000, "服务器内部错误"
	}
}
