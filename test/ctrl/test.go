package ctrl

import (
	"github.com/gin-gonic/gin"
	"parkjunwoo.com/microstral/test/model"
)

type TestController struct {
	TM *model.TestModel
}

func NewTestController(tm *model.TestModel) *TestController {
	return &TestController{
		TM: tm,
	}
}

func (tc *TestController) GetTest(c *gin.Context) {
	result, err := tc.TM.GetTest()
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": result,
	})
}
