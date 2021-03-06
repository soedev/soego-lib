package soeerror

import (
	"fmt"
	"github.com/pkg/errors"
	soego_lib "github.com/soedev/soego-lib"
	"strings"
)

//GenerateError 生成错误返回
func GenerateError(err error, msg string) error {
	if err != nil {
		err = errors.Wrap(err, msg)
		return err
	}
	return errors.New(msg)
}

//GenerateErrMsg 返回错误字符串
func GenerateErrMsg(err error) string {
	if err != nil {
		errMsg := strings.Split(err.Error(), ": ")
		if len(errMsg) > 1 {
			soego_lib.Logger.Info(fmt.Sprintf("stack trace:\n%+v\n", err))
		}
		return errMsg[0]
	}
	return ""
}
