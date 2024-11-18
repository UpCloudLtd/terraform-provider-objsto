package provider

import (
	"context"
	"fmt"

	"github.com/aws/smithy-go/logging"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type logger struct {
	ctx context.Context
}

var _ logging.Logger = logger{}
var _ logging.ContextLogger = logger{}

func (l logger) WithContext(ctx context.Context) logging.Logger {
	return logger{ctx: ctx}
}

func (l logger) Logf(classification logging.Classification, format string, args ...interface{}) {
	if l.ctx == nil {
		return
	}

	msg := fmt.Sprintf("S3 API "+format, args...)
	switch classification {
	case logging.Debug:
		tflog.Debug(l.ctx, msg)
	case logging.Warn:
		tflog.Warn(l.ctx, msg)
	}
}
