// parkjunwoo.com/microstral/mist.go
package mist

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"parkjunwoo.com/microstral/pkg/env"
	"parkjunwoo.com/microstral/pkg/mttp"
	"parkjunwoo.com/microstral/pkg/services"
)

type Config struct {
	host      string
	httpsport int
	httpport  int
	fullchain string
	privkey   string
	http      bool
	https     bool
}

type Mist struct {
	cfg   Config
	conns []interface {
		Close() error
	}
	router *gin.Engine
	httpc  *mttp.Client
	awsCfg aws.Config
}

// New: Mist 서버 생성자
func New(http bool, https bool) (*Mist, error) {
	httpc := mttp.NewClient()

	region := env.GetEnv("REGION", "ap-northeast-2")
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	s := &Mist{
		cfg: Config{
			host:      env.GetEnv("HOST", "mist"),
			httpsport: env.GetEnvInt("HTTPS_PORT", 443),
			httpport:  env.GetEnvInt("HTTP_PORT", 80),
			fullchain: env.GetEnv("TLS_FULLCHAIN", ""),
			privkey:   env.GetEnv("TLS_PRIVKEY", ""),
			http:      http,
			https:     https,
		},
		router: gin.Default(),
		httpc:  httpc,
		awsCfg: awsCfg,
	}

	store := cookie.NewStore([]byte("secret"))
	s.router.Use(sessions.Sessions("s", store))

	// 헬스체크 엔드포인트
	s.GET("/healthcheck", nil, services.Healthcheck)
	s.GET("/live", nil, services.Healthcheck)

	return s, nil
}

// Run: 서버 실행
func (s *Mist) Run() error {
	errCh := make(chan error, 2)
	var httpsServer, httpServer *http.Server

	if s.cfg.http {
		httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.cfg.httpport),
			Handler: s.router,
		}
		go func() {
			log.Println("Starting HTTP server...")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}
	if s.cfg.https {
		httpsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.cfg.httpsport),
			Handler: s.router,
		}
		go func() {
			log.Println("Starting HTTPS server...")
			if err := httpsServer.ListenAndServeTLS(s.cfg.fullchain, s.cfg.privkey); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Println("now shutting down server...")
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer func() {
		for _, conn := range s.conns {
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
		}
	}()

	if s.cfg.http {
		if httpServer != nil {
			if err := httpServer.Shutdown(ctx); err != nil {
				log.Printf("HTTP shutdown error: %v", err)
			}
		}
	}
	if s.cfg.https {
		if httpsServer != nil {
			if err := httpsServer.Shutdown(ctx); err != nil {
				log.Printf("HTTPS shutdown error: %v", err)
			}
		}
	}

	log.Println("completed server shutdown")
	return nil
}

func (s *Mist) GetHost() string {
	return s.cfg.host
}

func (s *Mist) GetHTTPSPort() int {
	return s.cfg.httpsport
}

func (s *Mist) GetHTTPPort() int {
	return s.cfg.httpport
}

func (s *Mist) GetRouter() *gin.Engine {
	return s.router
}

func (s *Mist) GetHTTP() *mttp.Client {
	return s.httpc
}

func (s *Mist) Use(handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.Use(handlers...)
}

func (s *Mist) GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.GET(relativePath, handlers...)
}

func (s *Mist) POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.POST(relativePath, handlers...)
}

func (s *Mist) PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.PUT(relativePath, handlers...)
}

func (s *Mist) DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.DELETE(relativePath, handlers...)
}

func (s *Mist) PATCH(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.PATCH(relativePath, handlers...)
}

func (s *Mist) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.OPTIONS(relativePath, handlers...)
}

func (s *Mist) HEAD(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.router.HEAD(relativePath, handlers...)
}

func (s *Mist) Postgres() (*sql.DB, error) {
	//Postgres 연결
	host := env.GetEnv("POSTGRES_HOST", "db")
	port := env.GetEnvInt("POSTGRES_PORT", 5432)
	dbname := env.GetEnv("POSTGRES_DB", "mist")
	username := env.GetEnv("POSTGRES_USERNAME", "mist")
	password := env.GetEnv("POSTGRES_PASSWORD", "")
	openConns := env.GetEnvInt("POSTGRES_OPEN_CONNS", 15)
	maxIdleConns := env.GetEnvInt("POSTGRES_MAX_IDLE_CONNS", 15)
	connMaxLifetime := env.GetEnvInt("POSTGRES_CONN_MAX_LIFETIME", 0)

	postgresDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbname)
	conn, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		return nil, err
	}

	// Postgres 연결 풀 설정
	conn.SetMaxOpenConns(openConns)                                       // 최대 연결 수
	conn.SetMaxIdleConns(maxIdleConns)                                    // 최대 유휴 연결 수
	conn.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second) // 최대 연결 지속 시간

	// Postgres 연결 테스트
	err = conn.PingContext(context.Background())
	if err != nil {
		return nil, err
	}

	s.conns = append(s.conns, conn)

	return conn, nil
}

func (s *Mist) Redis() (*redis.Client, error) {
	//REDIS 연결
	host := env.GetEnv("REDIS_HOST", "redis")
	port := env.GetEnvInt("REDIS_PORT", 6379)
	password := env.GetEnv("REDIS_PASSWORD", "")
	db := env.GetEnvInt("REDIS_DB", 0)
	poolSize := env.GetEnvInt("REDIS_POOL_SIZE", 15)
	minIdleConns := env.GetEnvInt("REDIS_MIN_IDLE_CONNS", 5)

	conn := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})

	// REIDS 연결 테스트
	_, err := conn.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	s.conns = append(s.conns, conn)

	return conn, nil
}
