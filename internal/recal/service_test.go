package recal

import (
	"app/internal/helper"
	"app/internal/model"
	redizMock "app/internal/rediz/mocks"
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServiceCheckMasterDataExistingTestSuite struct {
	suite.Suite

	// Define
	ctx context.Context
	loc *time.Location

	// Dependencies
	service   IService
	redisRepo *redizMock.MockIRepository

	// Mock data
	resGetValueByHashField string
	errGetValueByHashField error
}

// This will run before each test
func (suite *ServiceCheckMasterDataExistingTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.redisRepo = new(redizMock.MockIRepository)
	suite.service = &Service{
		RedisRepository: suite.redisRepo,
	}

	// Load location
	suite.loc = helper.DefaultLocation()

	suite.resGetValueByHashField = `{"multiplier":250,"pea_no":"6200035411","start_date":1761325200,"end_date":1761757200}`
	suite.errGetValueByHashField = nil

	suite.redisRepo.On("GetValueByHashField", mock.Anything, mock.Anything, mock.Anything).Return(
		func(context.Context, string, string) (string, error) {
			return suite.resGetValueByHashField, suite.errGetValueByHashField
		},
	)
}

func (suite *ServiceCheckMasterDataExistingTestSuite) TestReturnDifferenceDatesFromStartAndEndDate() {
	datum := model.ZDMI{
		BIS:         "20251031",
		AB:          "20251020",
		VKONT:       "020017476321",
		BILL_FACTOR: 300,
	}

	expectedResults := model.DifferenceDates{
		model.DifferenceDate(time.Date(2025, 10, 20, 0, 0, 0, 0, suite.loc)),
		model.DifferenceDate(time.Date(2025, 10, 21, 0, 0, 0, 0, suite.loc)),
		model.DifferenceDate(time.Date(2025, 10, 22, 0, 0, 0, 0, suite.loc)),
		model.DifferenceDate(time.Date(2025, 10, 23, 0, 0, 0, 0, suite.loc)),
		model.DifferenceDate(time.Date(2025, 10, 24, 0, 0, 0, 0, suite.loc)),
		model.DifferenceDate(time.Date(2025, 10, 31, 0, 0, 0, 0, suite.loc)),
	}

	results, err := suite.service.CheckMasterDataExisting(suite.ctx, datum)
	suite.NoError(err)

	suite.Equal(expectedResults, results)

	suite.redisRepo.AssertCalled(suite.T(), "GetValueByHashField", suite.ctx, "020017476321", datum.Field())
}

func (suite *ServiceCheckMasterDataExistingTestSuite) TestReturnNonDifferenceDatesFromStartAndEndDate() {
	datum := model.ZDMI{
		BIS:         "20251030",
		AB:          "20251025",
		VKONT:       "020017476321",
		BILL_FACTOR: 300,
	}

	results, err := suite.service.CheckMasterDataExisting(suite.ctx, datum)
	suite.NoError(err)

	suite.Empty(results)

	suite.redisRepo.AssertCalled(suite.T(), "GetValueByHashField", suite.ctx, "020017476321", datum.Field())
}

func (suite *ServiceCheckMasterDataExistingTestSuite) TestReturnNilWhenRedisNil() {
	datum := model.ZDMI{
		BIS:         "20251031",
		AB:          "20251020",
		VKONT:       "020017476321",
		BILL_FACTOR: 300,
	}
	suite.resGetValueByHashField = ``
	suite.errGetValueByHashField = redis.Nil

	results, err := suite.service.CheckMasterDataExisting(suite.ctx, datum)
	suite.NoError(err)

	suite.Empty(results)

	suite.redisRepo.AssertCalled(suite.T(), "GetValueByHashField", suite.ctx, "020017476321", datum.Field())
}

func (suite *ServiceCheckMasterDataExistingTestSuite) TestReturnErrorWhenGetValueByHashField() {
	datum := model.ZDMI{
		BIS:         "20251031",
		AB:          "20251020",
		VKONT:       "020017476321",
		BILL_FACTOR: 300,
	}
	suite.resGetValueByHashField = ``
	suite.errGetValueByHashField = assert.AnError

	results, err := suite.service.CheckMasterDataExisting(suite.ctx, datum)
	suite.ErrorIs(err, assert.AnError)

	suite.Empty(results)

	suite.redisRepo.AssertCalled(suite.T(), "GetValueByHashField", suite.ctx, "020017476321", datum.Field())
}

func (suite *ServiceCheckMasterDataExistingTestSuite) TestReturnErrorWhenUnmarshalHashValue() {
	datum := model.ZDMI{
		BIS:         "20251031",
		AB:          "20251020",
		VKONT:       "020017476321",
		BILL_FACTOR: 300,
	}
	suite.resGetValueByHashField = ``

	results, err := suite.service.CheckMasterDataExisting(suite.ctx, datum)
	suite.EqualError(err, "unexpected end of JSON input")

	suite.Empty(results)

	suite.redisRepo.AssertCalled(suite.T(), "GetValueByHashField", suite.ctx, "020017476321", datum.Field())
}

func (suite *ServiceCheckMasterDataExistingTestSuite) TestReturnErrorWhenParseToHashValue() {
	datum := model.ZDMI{
		VKONT:       "020017476321",
		BILL_FACTOR: 300,
	}

	results, err := suite.service.CheckMasterDataExisting(suite.ctx, datum)
	suite.EqualError(err, "parse start date\nparsing time \" +0700\" as \"20060102 -0700\": cannot parse \" +0700\" as \"2006\"")

	suite.Empty(results)

	suite.redisRepo.AssertCalled(suite.T(), "GetValueByHashField", suite.ctx, "020017476321", datum.Field())
}

func TestServiceCheckMasterDataExisting(t *testing.T) {
	suite.Run(t, new(ServiceCheckMasterDataExistingTestSuite))
}
