package main

import (
	mist "parkjunwoo.com/microstral"
	"parkjunwoo.com/microstral/test/module"
)

func main() {
	// Mist 서버 생성
	s, err := mist.New(true, false)
	if err != nil {
		panic(err)
	}

	// Postgres 연결
	postgres, err := s.Postgres()
	if err != nil {
		panic(err)
	}

	// Redis 연결
	redis, err := s.Redis()
	if err != nil {
		panic(err)
	}

	// 테스트 모듈 설치
	err = module.Test(s, postgres, redis)
	if err != nil {
		panic(err)
	}

	// 서버 실행
	s.Run()
}
