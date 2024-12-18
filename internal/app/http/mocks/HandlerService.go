// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	model "github.com/wot-oss/tmc/internal/model"

	repos "github.com/wot-oss/tmc/internal/repos"
)

// HandlerService is an autogenerated mock type for the HandlerService type
type HandlerService struct {
	mock.Mock
}

// CheckHealth provides a mock function with given fields: ctx
func (_m *HandlerService) CheckHealth(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CheckHealth")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckHealthLive provides a mock function with given fields: ctx
func (_m *HandlerService) CheckHealthLive(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CheckHealthLive")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckHealthReady provides a mock function with given fields: ctx
func (_m *HandlerService) CheckHealthReady(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CheckHealthReady")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckHealthStartup provides a mock function with given fields: ctx
func (_m *HandlerService) CheckHealthStartup(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CheckHealthStartup")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteAttachment provides a mock function with given fields: ctx, repo, ref, attachmentFileName
func (_m *HandlerService) DeleteAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string) error {
	ret := _m.Called(ctx, repo, ref, attachmentFileName)

	if len(ret) == 0 {
		panic("no return value specified for DeleteAttachment")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, model.AttachmentContainerRef, string) error); ok {
		r0 = rf(ctx, repo, ref, attachmentFileName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteThingModel provides a mock function with given fields: ctx, repo, tmID
func (_m *HandlerService) DeleteThingModel(ctx context.Context, repo string, tmID string) error {
	ret := _m.Called(ctx, repo, tmID)

	if len(ret) == 0 {
		panic("no return value specified for DeleteThingModel")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, repo, tmID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FetchAttachment provides a mock function with given fields: ctx, repo, ref, attachmentFileName, concat
func (_m *HandlerService) FetchAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string, concat bool) ([]byte, error) {
	ret := _m.Called(ctx, repo, ref, attachmentFileName, concat)

	if len(ret) == 0 {
		panic("no return value specified for FetchAttachment")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, model.AttachmentContainerRef, string, bool) ([]byte, error)); ok {
		return rf(ctx, repo, ref, attachmentFileName, concat)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, model.AttachmentContainerRef, string, bool) []byte); ok {
		r0 = rf(ctx, repo, ref, attachmentFileName, concat)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, model.AttachmentContainerRef, string, bool) error); ok {
		r1 = rf(ctx, repo, ref, attachmentFileName, concat)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchLatestThingModel provides a mock function with given fields: ctx, repo, fetchName, restoreId
func (_m *HandlerService) FetchLatestThingModel(ctx context.Context, repo string, fetchName string, restoreId bool) ([]byte, error) {
	ret := _m.Called(ctx, repo, fetchName, restoreId)

	if len(ret) == 0 {
		panic("no return value specified for FetchLatestThingModel")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) ([]byte, error)); ok {
		return rf(ctx, repo, fetchName, restoreId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) []byte); ok {
		r0 = rf(ctx, repo, fetchName, restoreId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, bool) error); ok {
		r1 = rf(ctx, repo, fetchName, restoreId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchThingModel provides a mock function with given fields: ctx, repo, tmID, restoreId
func (_m *HandlerService) FetchThingModel(ctx context.Context, repo string, tmID string, restoreId bool) ([]byte, error) {
	ret := _m.Called(ctx, repo, tmID, restoreId)

	if len(ret) == 0 {
		panic("no return value specified for FetchThingModel")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) ([]byte, error)); ok {
		return rf(ctx, repo, tmID, restoreId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) []byte); ok {
		r0 = rf(ctx, repo, tmID, restoreId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, bool) error); ok {
		r1 = rf(ctx, repo, tmID, restoreId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindInventoryEntries provides a mock function with given fields: ctx, repo, name
func (_m *HandlerService) FindInventoryEntries(ctx context.Context, repo string, name string) ([]model.FoundEntry, error) {
	ret := _m.Called(ctx, repo, name)

	if len(ret) == 0 {
		panic("no return value specified for FindInventoryEntries")
	}

	var r0 []model.FoundEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]model.FoundEntry, error)); ok {
		return rf(ctx, repo, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []model.FoundEntry); ok {
		r0 = rf(ctx, repo, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.FoundEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, repo, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCompletions provides a mock function with given fields: ctx, kind, args, toComplete
func (_m *HandlerService) GetCompletions(ctx context.Context, kind string, args []string, toComplete string) ([]string, error) {
	ret := _m.Called(ctx, kind, args, toComplete)

	if len(ret) == 0 {
		panic("no return value specified for GetCompletions")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []string, string) ([]string, error)); ok {
		return rf(ctx, kind, args, toComplete)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, []string, string) []string); ok {
		r0 = rf(ctx, kind, args, toComplete)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, []string, string) error); ok {
		r1 = rf(ctx, kind, args, toComplete)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestTMMetadata provides a mock function with given fields: ctx, repo, fetchName
func (_m *HandlerService) GetLatestTMMetadata(ctx context.Context, repo string, fetchName string) (model.FoundVersion, error) {
	ret := _m.Called(ctx, repo, fetchName)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestTMMetadata")
	}

	var r0 model.FoundVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (model.FoundVersion, error)); ok {
		return rf(ctx, repo, fetchName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) model.FoundVersion); ok {
		r0 = rf(ctx, repo, fetchName)
	} else {
		r0 = ret.Get(0).(model.FoundVersion)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, repo, fetchName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTMMetadata provides a mock function with given fields: ctx, repo, tmID
func (_m *HandlerService) GetTMMetadata(ctx context.Context, repo string, tmID string) ([]model.FoundVersion, error) {
	ret := _m.Called(ctx, repo, tmID)

	if len(ret) == 0 {
		panic("no return value specified for GetTMMetadata")
	}

	var r0 []model.FoundVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]model.FoundVersion, error)); ok {
		return rf(ctx, repo, tmID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []model.FoundVersion); ok {
		r0 = rf(ctx, repo, tmID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.FoundVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, repo, tmID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ImportAttachment provides a mock function with given fields: ctx, repo, ref, attachmentFileName, content, contentType, force
func (_m *HandlerService) ImportAttachment(ctx context.Context, repo string, ref model.AttachmentContainerRef, attachmentFileName string, content []byte, contentType string, force bool) error {
	ret := _m.Called(ctx, repo, ref, attachmentFileName, content, contentType, force)

	if len(ret) == 0 {
		panic("no return value specified for ImportAttachment")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, model.AttachmentContainerRef, string, []byte, string, bool) error); ok {
		r0 = rf(ctx, repo, ref, attachmentFileName, content, contentType, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ImportThingModel provides a mock function with given fields: ctx, repo, file, opts
func (_m *HandlerService) ImportThingModel(ctx context.Context, repo string, file []byte, opts repos.ImportOptions) (repos.ImportResult, error) {
	ret := _m.Called(ctx, repo, file, opts)

	if len(ret) == 0 {
		panic("no return value specified for ImportThingModel")
	}

	var r0 repos.ImportResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte, repos.ImportOptions) (repos.ImportResult, error)); ok {
		return rf(ctx, repo, file, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte, repos.ImportOptions) repos.ImportResult); ok {
		r0 = rf(ctx, repo, file, opts)
	} else {
		r0 = ret.Get(0).(repos.ImportResult)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, []byte, repos.ImportOptions) error); ok {
		r1 = rf(ctx, repo, file, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListAuthors provides a mock function with given fields: ctx, search
func (_m *HandlerService) ListAuthors(ctx context.Context, search *model.Filters) ([]string, error) {
	ret := _m.Called(ctx, search)

	if len(ret) == 0 {
		panic("no return value specified for ListAuthors")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Filters) ([]string, error)); ok {
		return rf(ctx, search)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *model.Filters) []string); ok {
		r0 = rf(ctx, search)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *model.Filters) error); ok {
		r1 = rf(ctx, search)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListInventory provides a mock function with given fields: ctx, repo, search
func (_m *HandlerService) ListInventory(ctx context.Context, repo string, search *model.Filters) (*model.SearchResult, error) {
	ret := _m.Called(ctx, repo, search)

	if len(ret) == 0 {
		panic("no return value specified for ListInventory")
	}

	var r0 *model.SearchResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *model.Filters) (*model.SearchResult, error)); ok {
		return rf(ctx, repo, search)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *model.Filters) *model.SearchResult); ok {
		r0 = rf(ctx, repo, search)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.SearchResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *model.Filters) error); ok {
		r1 = rf(ctx, repo, search)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListManufacturers provides a mock function with given fields: ctx, search
func (_m *HandlerService) ListManufacturers(ctx context.Context, search *model.Filters) ([]string, error) {
	ret := _m.Called(ctx, search)

	if len(ret) == 0 {
		panic("no return value specified for ListManufacturers")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Filters) ([]string, error)); ok {
		return rf(ctx, search)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *model.Filters) []string); ok {
		r0 = rf(ctx, search)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *model.Filters) error); ok {
		r1 = rf(ctx, search)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListMpns provides a mock function with given fields: ctx, search
func (_m *HandlerService) ListMpns(ctx context.Context, search *model.Filters) ([]string, error) {
	ret := _m.Called(ctx, search)

	if len(ret) == 0 {
		panic("no return value specified for ListMpns")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Filters) ([]string, error)); ok {
		return rf(ctx, search)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *model.Filters) []string); ok {
		r0 = rf(ctx, search)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *model.Filters) error); ok {
		r1 = rf(ctx, search)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListRepos provides a mock function with given fields: ctx
func (_m *HandlerService) ListRepos(ctx context.Context) ([]model.RepoDescription, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ListRepos")
	}

	var r0 []model.RepoDescription
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]model.RepoDescription, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []model.RepoDescription); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.RepoDescription)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SearchInventory provides a mock function with given fields: ctx, repo, query
func (_m *HandlerService) SearchInventory(ctx context.Context, repo string, query string) (*model.SearchResult, error) {
	ret := _m.Called(ctx, repo, query)

	if len(ret) == 0 {
		panic("no return value specified for SearchInventory")
	}

	var r0 *model.SearchResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*model.SearchResult, error)); ok {
		return rf(ctx, repo, query)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *model.SearchResult); ok {
		r0 = rf(ctx, repo, query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.SearchResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, repo, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewHandlerService creates a new instance of HandlerService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewHandlerService(t interface {
	mock.TestingT
	Cleanup(func())
}) *HandlerService {
	mock := &HandlerService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
