package webcast

import (
	// "log"
	"context"
	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"io"
	"slices"
)

func Run(ctx context.Context, port string, sIds []string, casterGetter CasterGetter) error {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	router, err := graceful.Default(graceful.WithAddr(port))
	if err != nil {
		return err
	}
	router.Use(CrossOrigin())

	router.GET("/", func(c *gin.Context) {
		c.File("./web-video-demo/index.html")
	})

	router.GET("/stream/:cam/:sid", func(c *gin.Context) {
		cam := c.Param("cam")
		sid := c.Param("sid")
		if slices.Contains(sIds, cam+"/"+sid) {
			caster := casterGetter(cam, sid)
			if caster != nil { // It will be nil if there was a stream.monitorMakeMutex deadlock interrupted by app termination
				client := NewClient(c, caster)
				client.Start()
				client.Wait()
			}
		} else {
			c.AbortWithStatus(404)
		}
	})
	return router.RunWithContext(ctx)
}

// CrossOrigin Access-Control-Allow-Origin any methods
func CrossOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
