package soego_lib

import "github.com/soedev/soego/core/elog"

var Logger *elog.Component

const PackageName = "soego-lib"

func init() {
	Logger = elog.EgoLogger.With(elog.FieldComponent(PackageName))
}
