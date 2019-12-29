package accesspoint_test

import (
	"testing"

	ap "bitbucket.org/au10/service/accesspoint/lib"
	mock_context "bitbucket.org/au10/service/mock/context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func Test_Accesspoint_Grpc(test *testing.T) {
	mock := gomock.NewController(test)
	defer mock.Finish()
	assert := assert.New(test)

	grpc := ap.CreateGrpc()

	ctx := mock_context.NewMockContext(mock)
	ctx.EXPECT().Value(gomock.Any()).Return(nil)
	err := grpc.SendHeader(ctx, metadata.MD{})
	assert.NotNil(err)

}
