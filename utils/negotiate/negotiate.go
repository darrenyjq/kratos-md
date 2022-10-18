package negotiate

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
)

func AbortError(c *gin.Context, err error, data ...interface{}) (int, gin.Negotiate) {
	return AbortErrorWithStatus(c, err, http.StatusOK, data...)
}

func AbortBadRequestError(c *gin.Context, err error, data ...interface{}) (int, gin.Negotiate) {
	return AbortErrorWithStatus(c, err, http.StatusBadRequest, data...)
}

func AbortForbiddenError(c *gin.Context, err error, data ...interface{}) (int, gin.Negotiate) {
	return AbortErrorWithStatus(c, err, http.StatusForbidden, data...)
}

func AbortInternalServerError(c *gin.Context, err error, data ...interface{}) (int, gin.Negotiate) {
	return AbortErrorWithStatus(c, err, http.StatusInternalServerError, data...)
}

func AbortErrorWithStatus(c *gin.Context, err error, status int, data ...interface{}) (int, gin.Negotiate) {
	c.Abort() //关键步骤，中止后续执行
	c.Error(errors.Errorf("%+v", err))
	if len(data) > 0 {
		return status, gin.Negotiate{
			Offered: []string{binding.MIMEJSON},
			Data:    data[0],
		}
	} else {
		return status, gin.Negotiate{
			Offered: []string{binding.MIMEJSON},
			Data: gin.H{
				"Status": http.StatusText(status),
			},
		}
	}
}

func JSON(code int, data interface{}) (int, gin.Negotiate) {
	return code, gin.Negotiate{
		Offered: []string{binding.MIMEJSON},
		Data:    data,
	}
}

func HTML(code int, name string, data interface{}) (int, gin.Negotiate) {
	return code, gin.Negotiate{
		Offered:  []string{binding.MIMEHTML},
		HTMLName: name,
		Data:     data,
	}
}
