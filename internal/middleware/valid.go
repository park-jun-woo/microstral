package middleware

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"parkjunwoo.com/microstral/internal/meta"
)

// Valid는 메타데이터에 등록된 파라미터를 검증하는 미들웨어입니다.
func Valid(metaMap meta.MetadataMap, strictURL bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) 메타데이터 조회
		md := metaMap[fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())]
		if md == nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "unknown path: " + c.FullPath(),
			})
			return
		}

		// Role 체크
		rols, exists := c.Get("roles")
		if exists {
			roles := rols.([]uint32)
			if !md.HasRoles(roles) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "forbidden",
				})
				return
			}
		}

		// 메타에서 정의되지 않은 파라미터 => 400
		switch c.Request.Method {
		case http.MethodGet, http.MethodDelete:
			// strictURL모드라면 GET, DELETE는 등록된 Query만 허용
			if strictURL {
				for qKey := range c.Request.URL.Query() {
					if _, found := md.Params[qKey]; !found {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": "unknown query param: " + qKey,
						})
						return
					}
				}
			}
		// POST, PUT은 모든 Query 를 금지하고, 등록한 Form만 허용
		case http.MethodPost, http.MethodPut:
			for qKey := range c.Request.URL.Query() {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "unknown query param: " + qKey,
				})
				return
			}
			// Form은 등록한 파라미터만 허용
			for fKey := range c.Request.PostForm {
				if _, found := md.Params[fKey]; !found {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": "unknown form param: " + fKey,
					})
					return
				}
			}
		}

		// 파라미터 로직에서 PostForm() 등을 사용할 수 있도록 명시적으로 parse
		// (Gin 내부적으로 c.PostForm(...) 부를 때 parse하긴 하지만, 확실히 하려면 호출)
		_ = c.Request.ParseForm()
		values := url.Values{}

		// 2) 메타데이터에 등록된 파라미터 목록 순회
		for _, p := range md.Params {
			var value string

			// "URL Param -> Query / PostForm" 순서
			isParam := true
			value = c.Param(p.Name)
			if value == "" {
				isParam = false
				switch c.Request.Method {
				case http.MethodGet, http.MethodDelete:
					value = c.Query(p.Name)
				case http.MethodPost, http.MethodPut:
					value = c.PostForm(p.Name)
				}
			}

			// (a) 필수 파라미터인지 확인
			if value == "" {
				if p.Required {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"param": p.Name,
						"error": "required",
					})
					return
				} else {
					value = p.Default
					if !isParam {
						switch c.Request.Method {
						case http.MethodGet, http.MethodDelete:
							values.Set(p.Name, p.Default)
						case http.MethodPost, http.MethodPut:
							c.Request.PostForm.Set(p.Name, p.Default)
						}
					}

				}
			}

			// (c) 파라미터 값 검증
			if value != "" {
				ok, err := p.Validate(value)
				if !ok || err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"param": p.Name,
						"error": err.Error(),
					})
					return
				}
			}
		}

		// 3) 마지막에 QueryString 재구성
		//    (주의) 위에서 Default를 세팅했을 수도 있으므로, 다시 URL.Query()를 읽어와 Encode()
		//    하지만 "Param"에는 Default가 들어갔을 수도 있음.
		//    최종 결정: Param "Query"만 재구성
		updated := values.Encode()
		if updated != "" {
			updated = "?" + updated
		}
		c.Request.URL.RawQuery = updated

		// 4) 다음 미들웨어/핸들러로 진행
		c.Next()
	}
}
