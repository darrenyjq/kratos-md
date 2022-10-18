package negotiate

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func AbortStringError(c *gin.Context, err error, data ...string) (int, string) {
	return AbortStringErrorWithStatus(c, err, http.StatusOK, data...)
}

func AbortStringBadRequestError(c *gin.Context, err error, data ...string) (int, string) {
	return AbortStringErrorWithStatus(c, err, http.StatusBadRequest, data...)
}

func AbortStringForbiddenError(c *gin.Context, err error, data ...string) (int, string) {
	return AbortStringErrorWithStatus(c, err, http.StatusForbidden, data...)
}

func AbortStringInternalServerError(c *gin.Context, err error, data ...string) (int, string) {
	return AbortStringErrorWithStatus(c, err, http.StatusInternalServerError, data...)
}

func AbortStringErrorWithStatus(c *gin.Context, err error, status int, data ...string) (int, string) {
	c.Abort() //关键步骤，中止后续执行
	c.Error(errors.Errorf("%+v", err))
	if len(data) > 0 {
		return status, data[0]
	} else {
		return status, ""
	}
}
