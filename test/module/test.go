package module

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
	mist "parkjunwoo.com/microstral"

	"parkjunwoo.com/microstral/pkg/flag"

	"parkjunwoo.com/microstral/test/ctrl"
	"parkjunwoo.com/microstral/test/model"
)

func Test(s *mist.Mist, postgres *sql.DB, redis *redis.Client) error {
	// 테스트 모델 생성
	testModel := model.NewTestModel(postgres, redis)
	// 테스트 컨트롤러 생성
	testCtrl := ctrl.NewTestController(testModel)
	// 테스트 라우터 등록
	s.GET("/test", testCtrl.GetTest).Flag("name", "", flag.ALPHA_HANGUL_NUM)

	return nil
}
