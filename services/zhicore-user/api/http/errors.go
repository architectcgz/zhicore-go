package httpapi

import (
	"errors"
	"net/http"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

var (
	errLoginRequired  = errors.New("login required")
	errInvalidRequest = errors.New("invalid request")
)

func writeValidationError(c *gin.Context) {
	sharedhttp.WriteErrorCode(c.Writer, http.StatusBadRequest, 1001, "参数校验失败")
}

func writeMappedError(c *gin.Context, err error) {
	status, code, message := errorMapping(err)
	sharedhttp.WriteErrorCode(c.Writer, status, code, message)
}

func errorMapping(err error) (int, int, string) {
	switch {
	case errors.Is(err, errLoginRequired):
		return http.StatusUnauthorized, 2006, "请先登录"
	case errors.Is(err, application.ErrPublicIDInvalid), errors.Is(err, errInvalidRequest):
		return http.StatusBadRequest, 1001, "参数校验失败"
	case errors.Is(err, application.ErrProfileNotFound):
		return http.StatusNotFound, 3001, "用户不存在"
	case errors.Is(err, application.ErrNicknameTaken):
		return http.StatusConflict, 3005, "昵称已被使用"
	case errors.Is(err, application.ErrUserNotActive):
		return http.StatusForbidden, 3006, "用户不可用"
	case errors.Is(err, application.ErrNicknameInvalid):
		return http.StatusBadRequest, 3013, "昵称不合法"
	case errors.Is(err, application.ErrBioInvalid):
		return http.StatusBadRequest, 3014, "简介不合法"
	case errors.Is(err, application.ErrAvatarInvalid):
		return http.StatusBadRequest, 3015, "头像文件不可引用"
	case errors.Is(err, application.ErrCannotFollowSelf):
		return http.StatusBadRequest, 3007, "不能关注自己"
	case errors.Is(err, application.ErrInteractionBlocked):
		return http.StatusForbidden, 3010, "互动被拉黑阻止"
	case errors.Is(err, application.ErrCannotBlockSelf):
		return http.StatusBadRequest, 3011, "不能拉黑自己"
	case errors.Is(err, application.ErrCursorInvalid):
		return http.StatusBadRequest, 1001, "参数校验失败"
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "服务暂时不可用"
	default:
		return http.StatusInternalServerError, 1000, "服务器内部错误"
	}
}
