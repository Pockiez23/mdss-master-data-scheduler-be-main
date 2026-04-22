package holiday

import (
	"app/internal/global"
	holidaymocks "app/internal/holiday/mocks"
	"app/internal/locker"
	"app/internal/model"
	redizmocks "app/internal/rediz/mocks"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServiceFetchHolidaysSuite struct {
	suite.Suite

	ctx  context.Context
	task string

	repo      *holidaymocks.MockIRepository
	redizRepo *redizmocks.MockIRepository

	service IService
}

func (suite *ServiceFetchHolidaysSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.task = "holiday"

	locker.Unlock(suite.task)

	suite.repo = new(holidaymocks.MockIRepository)
	suite.redizRepo = new(redizmocks.MockIRepository)

	suite.service = &Service{
		Repository:      suite.repo,
		RedisRepository: suite.redizRepo,
	}
}

func (suite *ServiceFetchHolidaysSuite) TearDownTest() {
	locker.Unlock(suite.task)

	suite.repo.AssertExpectations(suite.T())
	suite.redizRepo.AssertExpectations(suite.T())
}

func TestServiceFetchHolidaysSuite(t *testing.T) {
	suite.Run(t, new(ServiceFetchHolidaysSuite))
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysWhenLockedReturnErrLockedAndNoRepoCalls() {
	locker.Lock(suite.task)
	defer locker.Unlock(suite.task)

	err := suite.service.FetchHolidays(suite.ctx)
	suite.ErrorIs(err, global.ErrLocked)
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysLatestOffsetRedisNilFetchAllStoreAndStamp() {
	data := []model.Holiday{
		{}, {}, {},
	}

	suite.redizRepo.On("GetLatestOffset", mock.Anything, suite.task).Return(time.Time{}, redis.Nil).Once()
	suite.repo.On("FetchAll").Return(data, nil).Once()
	suite.redizRepo.On("Set", mock.Anything, mock.MatchedBy(
		func(key string) bool {
			return strings.HasPrefix(key, "holiday:")
		},
	), true, time.Duration(0)).Return(nil).Times(len(data))

	suite.redizRepo.On("SetLatestOffset", mock.Anything, suite.task, mock.MatchedBy(
		func(t time.Time) bool {
			return !t.IsZero()
		},
	)).Return(nil).Once()

	err := suite.service.FetchHolidays(suite.ctx)
	suite.NoError(err)
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysLatestOffsetExistsFetchScopeStoreAndStamp() {
	latest := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	data := []model.Holiday{{}}

	suite.redizRepo.On("GetLatestOffset", mock.Anything, suite.task).Return(latest, nil).Once()
	suite.repo.On("Fetch", latest).Return(data, nil).Once()
	suite.redizRepo.On("Set", mock.Anything, mock.MatchedBy(
		func(key string) bool {
			return strings.HasPrefix(key, "holiday:")
		},
	), true, time.Duration(0)).Return(nil).Times(len(data))
	suite.redizRepo.On("SetLatestOffset", mock.Anything, suite.task, mock.AnythingOfType("time.Time")).Return(nil).Once()

	err := suite.service.FetchHolidays(suite.ctx)
	suite.NoError(err)
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysGetLatestOffsetErrorReturnWrappedError() {
	mockError := assert.AnError

	suite.redizRepo.On("GetLatestOffset", mock.Anything, suite.task).Return(time.Time{}, mockError).Once()

	err := suite.service.FetchHolidays(suite.ctx)
	suite.Error(err)

	suite.True(errors.Is(err, mockError))
	suite.Contains(err.Error(), "get latest offset")
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysFetchAllErrorReturnWrappedError() {
	mockError := assert.AnError

	suite.redizRepo.On("GetLatestOffset", mock.Anything, suite.task).Return(time.Time{}, redis.Nil).Once()
	suite.repo.On("FetchAll").Return(nil, mockError).Once()

	err := suite.service.FetchHolidays(suite.ctx)
	suite.Error(err)

	suite.True(errors.Is(err, mockError))
	suite.Contains(err.Error(), "fetch data from HANA")
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysSetErrorReturnErrorAndNoStamp() {
	mockError := assert.AnError
	data := []model.Holiday{{}, {}}

	suite.redizRepo.On("GetLatestOffset", mock.Anything, suite.task).Return(time.Time{}, redis.Nil).Once()
	suite.repo.On("FetchAll").Return(data, nil).Once()
	suite.redizRepo.On("Set", mock.Anything, mock.MatchedBy(
		func(key string) bool {
			return strings.HasPrefix(key, "holiday:")
		},
	), true, time.Duration(0)).Return(mockError)

	err := suite.service.FetchHolidays(suite.ctx)
	suite.ErrorIs(err, mockError)
}

func (suite *ServiceFetchHolidaysSuite) TestFetchHolidaysSetLatestOffsetErrorReturnWrappedError() {
	mockError := assert.AnError
	data := []model.Holiday{{}}

	suite.redizRepo.On("GetLatestOffset", mock.Anything, "holiday").Return(time.Time{}, redis.Nil).Once()
	suite.repo.On("FetchAll").Return(data, nil).Once()
	suite.redizRepo.On("Set", mock.Anything, mock.MatchedBy(
		func(key string) bool {
			return strings.HasPrefix(key, "holiday:")
		},
	), true, time.Duration(0)).Return(nil).Times(len(data))
	suite.redizRepo.On("SetLatestOffset", mock.Anything, "holiday", mock.AnythingOfType("time.Time")).Return(mockError)

	err := suite.service.FetchHolidays(suite.ctx)
	suite.Error(err)

	suite.True(errors.Is(err, mockError))
	suite.Contains(err.Error(), "set latest offset")
}
