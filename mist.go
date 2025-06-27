// https://parkjunwoo.com/microstral/mist.go
package mist

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"parkjunwoo.com/microstral/pkg/env"
	"parkjunwoo.com/microstral/pkg/mttp"
	"parkjunwoo.com/microstral/pkg/param"
	"parkjunwoo.com/microstral/pkg/services"

	"parkjunwoo.com/microstral/internal/meta"
	"parkjunwoo.com/microstral/internal/middleware"
)

func NewMeta() *meta.Metadata {
	md := meta.Metadata{
		Roles:  make([]uint32, 0),
		Params: make(map[string]param.Param),
	}
	return &md
}

// Mist 서버  구조체
type Mist struct {
	host  string
	port  int
	metas meta.MetadataMap
	conns []interface {
		Close() error
	}

	router *gin.Engine
	httpc  *mttp.Client
}

// New: Mist 서버 생성자
func New(useMiddleware bool, strictURL bool) (*Mist, error) {
	httpc := mttp.NewClient()

	s := &Mist{
		host:   env.GetEnv("HOST", "mist"),
		port:   env.GetEnvInt("PORT", 80),
		router: gin.Default(),
		metas:  make(meta.MetadataMap),
		httpc:  httpc,
	}

	if useMiddleware {
		// CORS 미들웨어 적용
		s.router.Use(middleware.Origin())

		// 인증 미들웨어 적용
		s.router.Use(middleware.Auth())

		// 로깅 미들웨어 적용
		s.router.Use(middleware.Logger())
	}

	// 헬스체크 엔드포인트
	s.GET("/healthcheck", services.Healthcheck)
	s.GET("/live", services.Healthcheck)

	return s, nil
}

// Run: 서버 실행
func (s *Mist) Run() error {
	// 서버 실행
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	errCh := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// OS 종료 신호 대기 (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("now shutting down server...")
	case err := <-errCh:
		return err
	}

	// graceful shutdown (5초 타임아웃)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer func() {
		for _, conn := range s.conns {
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
		}
	}()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}
	log.Println("completed server shutdown")

	return nil
}

func (s *Mist) GetHost() string {
	return s.host
}

func (s *Mist) GetPort() int {
	return s.port
}

func (s *Mist) GetRouter() *gin.Engine {
	return s.router
}

func (s *Mist) GetHTTP() *mttp.Client {
	return s.httpc
}

func (s *Mist) Use(handlers ...gin.HandlerFunc) {
	s.router.Use(handlers...)
}
